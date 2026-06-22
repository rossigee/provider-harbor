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
	errNotGroupMember    = "managed resource is not a GroupMember custom resource"
	errGroupMemberGet    = "cannot get Harbor group member"
	errGroupMemberCreate = "cannot add Harbor group member"
	errGroupMemberUpdate = "cannot update Harbor group member"
	errGroupMemberDelete = "cannot delete Harbor group member"

	// defaultGroupType is Harbor's OIDC group source, used when the GroupMember
	// leaves groupType unset.
	defaultGroupType int64 = 3
)

// SetupGroupMember adds a controller that reconciles GroupMember managed resources.
func SetupGroupMember(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.GroupMemberGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.GroupMemberGroupVersionKind),
		managed.WithExternalConnector(&groupMemberConnector{
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
		For(&v1beta1.GroupMember{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
}

type groupMemberConnector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *groupMemberConnector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.GroupMember)
	if !ok {
		return nil, errors.New(errNotGroupMember)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &groupMemberExternal{service: svc}, nil
}

type groupMemberExternal struct {
	service harborclients.HarborClienter
}

// groupMemberID returns the Harbor member id stored as the external name, or ""
// when it has not been set yet (external name defaults to metadata.name).
func groupMemberID(cr *v1beta1.GroupMember) string {
	en := meta.GetExternalName(cr)
	if en == "" || en == cr.GetName() {
		return ""
	}
	return en
}

// groupType returns the configured Harbor group source, defaulting to OIDC (3).
func groupType(cr *v1beta1.GroupMember) int64 {
	if cr.Spec.ForProvider.GroupType != nil {
		return *cr.Spec.ForProvider.GroupType
	}
	return defaultGroupType
}

func (c *groupMemberExternal) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.GroupMember)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotGroupMember)
	}

	// Prefer the authoritative get-by-id once the member's Harbor id is known.
	if id := groupMemberID(cr); id != "" {
		status, err := c.service.GetProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errGroupMemberGet)
		}
		if status == nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return groupMemberObservation(cr, status), nil
	}

	// No id yet: adopt a pre-existing group member by entity name.
	status, err := c.service.FindProjectMember(ctx, cr.Spec.ForProvider.ProjectID, "g", cr.Spec.ForProvider.GroupName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGroupMemberGet)
	}
	if status == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	meta.SetExternalName(cr, status.ID)
	return groupMemberObservation(cr, status), nil
}

func groupMemberObservation(cr *v1beta1.GroupMember, status *harborclients.MemberStatus) managed.ExternalObservation {
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

func (c *groupMemberExternal) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.GroupMember)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotGroupMember)
	}

	cr.SetConditions(xpv1.Creating())

	id, err := c.service.AddProjectGroupMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.GroupName, groupType(cr), cr.Spec.ForProvider.Role)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGroupMemberCreate)
	}

	// Persist the member id as the external name so the next Observe gets by id.
	meta.SetExternalName(cr, id)
	cr.Status.AtProvider.ID = &id

	return managed.ExternalCreation{}, nil
}

func (c *groupMemberExternal) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.GroupMember)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotGroupMember)
	}

	id := groupMemberID(cr)
	if id == "" {
		return managed.ExternalUpdate{}, errors.New(errGroupMemberUpdate + ": member id unknown")
	}

	if err := c.service.UpdateProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id, cr.Spec.ForProvider.Role); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errGroupMemberUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *groupMemberExternal) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.GroupMember)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotGroupMember)
	}

	cr.SetConditions(xpv1.Deleting())

	id := groupMemberID(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}

	if err := c.service.DeleteProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errGroupMemberDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *groupMemberExternal) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
