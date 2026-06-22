/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

import (
	"context"
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

	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotUserMember    = "managed resource is not a UserMember custom resource"
	errUserMemberGet    = "cannot get Harbor user member"
	errUserMemberCreate = "cannot add Harbor user member"
	errUserMemberUpdate = "cannot update Harbor user member"
	errUserMemberDelete = "cannot delete Harbor user member"
)

// SetupUserMember adds a controller that reconciles UserMember managed resources.
func SetupUserMember(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.UserMemberGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.UserMemberGroupVersionKind),
		managed.WithExternalConnector(&userMemberConnector{
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
		For(&v1beta1.UserMember{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
}

type userMemberConnector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *userMemberConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.UserMember)
	if !ok {
		return nil, errors.New(errNotUserMember)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &userMemberExternal{service: svc}, nil
}

type userMemberExternal struct {
	service harborclients.HarborClienter
}

// userMemberID returns the Harbor member id stored as the external name, or ""
// when it has not been set yet (external name defaults to metadata.name).
func userMemberID(cr *v1beta1.UserMember) string {
	en := meta.GetExternalName(cr)
	if en == "" || en == cr.GetName() {
		return ""
	}
	return en
}

func (c *userMemberExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.UserMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserMember)
	}

	// Prefer the authoritative get-by-id once the member's Harbor id is known.
	if id := userMemberID(cr); id != "" {
		status, err := c.service.GetProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errUserMemberGet)
		}
		if status == nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return userMemberObservation(cr, status), nil
	}

	// No id yet: adopt a pre-existing user member by entity name.
	status, err := c.service.FindProjectMember(ctx, cr.Spec.ForProvider.ProjectID, "u", cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errUserMemberGet)
	}
	if status == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	meta.SetExternalName(cr, status.ID)
	return userMemberObservation(cr, status), nil
}

func userMemberObservation(cr *v1beta1.UserMember, status *harborclients.MemberStatus) managed.ExternalObservation {
	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.MemberName = &status.MemberName
	cr.Status.AtProvider.MemberType = &status.MemberType
	cr.Status.AtProvider.Role = &status.Role
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t

	upToDate := cr.Spec.ForProvider.Role == "" || status.Role == "" || cr.Spec.ForProvider.Role == status.Role

	// Mark Available: the resource exists and is usable. Drift is signalled via
	// ResourceUpToDate (-> Update)/Synced, not by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}
}

func (c *userMemberExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.UserMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserMember)
	}

	cr.SetConditions(xpv1.Creating())

	id, err := c.service.AddProjectUserMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username, cr.Spec.ForProvider.Role)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUserMemberCreate)
	}

	// Persist the member id as the external name so the next Observe gets by id.
	meta.SetExternalName(cr, id)
	cr.Status.AtProvider.ID = &id

	return managed.ExternalCreation{}, nil
}

func (c *userMemberExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.UserMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserMember)
	}

	id := userMemberID(cr)
	if id == "" {
		return managed.ExternalUpdate{}, errors.New(errUserMemberUpdate + ": member id unknown")
	}

	if err := c.service.UpdateProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id, cr.Spec.ForProvider.Role); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUserMemberUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *userMemberExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.UserMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserMember)
	}

	cr.SetConditions(xpv1.Deleting())

	id := userMemberID(cr)
	if id == "" {
		// Nothing was ever created (or already adopted-then-lost); treat as done.
		return managed.ExternalDelete{}, nil
	}

	if err := c.service.DeleteProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUserMemberDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *userMemberExternal) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
