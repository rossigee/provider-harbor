/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotRobot    = "managed resource is not a Robot custom resource"
	errRobotDelete = "cannot delete Harbor robot"
	errNewClient   = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.RobotGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RobotGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Robot{}).
		// A non-nil rate limiter is required: ratelimiter.Reconciler.When()
		// dereferences it on every reconcile (nil -> panic).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Robot)
	if !ok {
		return nil, errors.New(errNotRobot)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

type external struct {
	service harborclients.HarborClienter
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobot)
	}

	// Prefer the authoritative get-by-id once the robot's Harbor id is known
	// (stored as the external name). Harbor's system robot list (GET /robots)
	// does NOT return project-scoped robots, so a name/list lookup cannot
	// reliably re-match a project robot after creation — the id path is what
	// keeps Observe stable and avoids a create→409 loop.
	if id := robotExternalID(cr); id != "" {
		robot, err := c.service.GetRobot(ctx, id)
		if err != nil {
			return managed.ExternalObservation{}, err
		}
		if robot == nil {
			// Deleted out of band — let Crossplane recreate it.
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return robotObservation(cr, robot), nil
	}

	// No id yet: fall back to a name match over the list to adopt a pre-existing
	// robot. Harbor names a project robot robot$<project>+<shortname>.
	robots, err := c.service.ListRobots(ctx, cr.Spec.ForProvider.ProjectID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}
	for _, robot := range robots {
		if !matchesRobotName(robot.Name, cr.Spec.ForProvider.Name) {
			continue
		}
		meta.SetExternalName(cr, robot.ID)
		return robotObservation(cr, robot), nil
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

// robotExternalID returns the Harbor robot id stored as the external name, or ""
// if it has not been set yet. Crossplane defaults the external name to
// metadata.name, which is not a numeric Harbor id, so we treat a non-numeric
// external name as "not created yet".
func robotExternalID(cr *v1beta1.Robot) string {
	en := meta.GetExternalName(cr)
	if en == "" || en == cr.GetName() {
		return ""
	}
	// Treat a non-positive id as "not set" — Harbor robot ids are positive, and a
	// stray "0" would otherwise poison Observe (GetRobot(0) always errors).
	if id, err := strconv.ParseInt(en, 10, 64); err != nil || id <= 0 {
		return ""
	}
	return en
}

// robotObservation records observed state on the CR and computes the
// ExternalObservation (drift + readiness).
func robotObservation(cr *v1beta1.Robot, robot *harborclients.RobotStatus) managed.ExternalObservation {
	cr.Status.AtProvider.ID = &robot.ID
	if robot.Secret != "" {
		cr.Status.AtProvider.Secret = &robot.Secret
	}
	if robot.ExpiresAt != nil {
		et := metav1.NewTime(*robot.ExpiresAt)
		cr.Status.AtProvider.ExpiresAt = &et
	}
	t := metav1.NewTime(robot.CreationTime)
	cr.Status.AtProvider.CreationTime = &t
	ut := metav1.NewTime(robot.UpdateTime)
	cr.Status.AtProvider.UpdateTime = &ut

	descDrifted := cr.Spec.ForProvider.Description != nil && robot.Description != nil && *cr.Spec.ForProvider.Description != *robot.Description
	// A robot's project is fixed at creation (Harbor scopes it immutably) and the
	// observed ProjectID is the project NAME, not the numeric-id spec value, so it
	// is not a meaningful drift signal — compare description only.
	upToDate := !descDrifted

	// Mark Available: the resource exists and is usable. Drift is signalled via
	// ResourceUpToDate (-> Update)/Synced, not by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}
}

// matchesRobotName reports whether a Harbor robot's full name corresponds to the
// CR's short name: either an exact match or the robot$<project>+<short> form.
func matchesRobotName(full, short string) bool {
	return full == short || strings.HasSuffix(full, "+"+short)
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobot)
	}

	cr.SetConditions(xpv1.Creating())

	spec := &harborclients.RobotSpec{
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		ExpiresIn:   cr.Spec.ForProvider.ExpiresIn,
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	status, err := c.service.CreateRobot(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	// The robot secret is only returned at creation — record the observed id and
	// publish the credential as connection details (Harbor never returns it again).
	// Persist the id as the external name: crossplane-runtime writes it back
	// immediately after Create, so the next Observe can get-by-id (Harbor's robot
	// list won't surface project robots).
	meta.SetExternalName(cr, status.ID)
	cr.Status.AtProvider.ID = &status.ID
	if status.Secret != "" {
		cr.Status.AtProvider.Secret = &status.Secret
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"name":   []byte(status.Name),
			"secret": []byte(status.Secret),
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRobot)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("robot ID not set")
	}

	spec := &harborclients.RobotSpec{
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		ExpiresIn:   cr.Spec.ForProvider.ExpiresIn,
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	_, err := c.service.UpdateRobot(ctx, *cr.Status.AtProvider.ID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRobot)
	}

	cr.SetConditions(xpv1.Deleting())

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteRobot(ctx, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRobotDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}

func convertPermissions(perms []v1beta1.RobotPermission) []harborclients.RobotPermission {
	if len(perms) == 0 {
		return nil
	}
	result := make([]harborclients.RobotPermission, len(perms))
	for i, p := range perms {
		result[i] = harborclients.RobotPermission{
			Namespace: p.Namespace,
			Access:    p.Access,
		}
	}
	return result
}
