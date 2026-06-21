/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv2 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1beta1 "github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
)

const (
	errNotRobot    = "managed resource is not a Robot custom resource"
	errRobotDelete = "cannot delete Harbor robot"
	errNewClient   = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.RobotGroupVersionKind.Kind)
	log := logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RobotGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
			logger:       log,
		}),
		managed.WithLogger(log),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Robot{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(10)))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
	logger       logging.Logger
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

	return &external{service: svc, logger: c.logger}, nil
}

type external struct {
	service harborclients.HarborClienter
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobot)
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe called for %s, desiredName=%s\n", cr.Name, cr.Spec.ForProvider.Name))

	// Get robot by name (simplified - Harbor API would need the robot ID)
	robots, err := c.service.ListRobots(ctx, cr.Spec.ForProvider.ProjectID)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe error calling ListRobots: %v\n", err))
		return managed.ExternalObservation{}, err
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe got %d robots\n", len(robots)))

	// Harbor robot names have "robot$" prefix, so we need to handle that
	// Use external name if set for adoption scenarios
	externalName := ctrlutil.GetExternalName(cr)
	searchName := cr.Spec.ForProvider.Name
	if externalName != "" {
		// Adoption scenario: use external name to find existing resource
		searchName = externalName
	}
	if !strings.HasPrefix(searchName, "robot$") {
		searchName = "robot$" + searchName
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe searching for %s\n", searchName))

	for _, robot := range robots {
		os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe checking %s\n", robot.Name))
		// Also check without prefix in case the name was stored differently
		if robot.Name == searchName || robot.Name == cr.Spec.ForProvider.Name {
			os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe FOUND %s id=%s\n", robot.Name, robot.ID))

			// Set external name for adoption tracking
			ctrlutil.SetExternalName(cr, robot.Name)

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

			upToDate := true
			if cr.Spec.ForProvider.Description != nil && robot.Description != nil && *cr.Spec.ForProvider.Description != *robot.Description {
				upToDate = false
			}
			if cr.Spec.ForProvider.ProjectID != nil && robot.ProjectID != nil && *cr.Spec.ForProvider.ProjectID != *robot.ProjectID {
				upToDate = false
			}

			os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe returning exists=true, upToDate=%v\n", upToDate))

			// Set the Ready condition to True since we found the resource
			cr.SetConditions(xpv2.Available())

			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
		}
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Observe not found, will need to create\n"))
	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobot)
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Create called for %s\n", cr.Name))

	spec := &harborclients.RobotSpec{
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		ExpiresIn:   cr.Spec.ForProvider.ExpiresIn,
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Create calling Harbor API for %s\n", cr.Spec.ForProvider.Name))
	robot, err := c.service.CreateRobot(ctx, spec)
	if err != nil {
		os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Create error: %v\n", err))
		return managed.ExternalCreation{}, err
	}

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, robot.Name)

	os.Stderr.WriteString(fmt.Sprintf("DEBUG_ROBOT: Create succeeded for %s\n", cr.Name))
	return managed.ExternalCreation{}, nil
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
