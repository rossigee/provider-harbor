/*
Copyright 2024 Crossplane Harbor Provider.
*/

package usergroup

import (
	"context"
	"strconv"
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

	"github.com/rossigee/provider-harbor/apis/usergroup/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotUserGroup    = "managed resource is not a UserGroup custom resource"
	errTrackPCUsage    = "cannot track ProviderConfig usage"
	errGetPC           = "cannot get ProviderConfig"
	errGetCreds        = "cannot get credentials"
	errNewClient       = "cannot create new Harbor client"
	errUserGroupCreate = "cannot create Harbor user group"
	errUserGroupGet    = "cannot get Harbor user group"
	errUserGroupUpdate = "cannot update Harbor user group"
	errUserGroupDelete = "cannot delete Harbor user group"
)

// Setup adds a controller that reconciles UserGroup managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.UserGroupGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.UserGroupGroupVersionKind),
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
		For(&v1beta1.UserGroup{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	newServiceFn func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.UserGroup)
	if !ok {
		return nil, errors.New(errNotUserGroup)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, kube: c.kube}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	service harborclients.HarborClienter
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.UserGroup)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserGroup)
	}

	// Prefer the authoritative get-by-id once the group's Harbor id is known
	// (stored as the external name). A list+name match cannot reliably re-match a
	// just-created group, which causes a create→409 loop; the id path fixes that.
	if id := userGroupExternalID(cr); id != "" {
		gid, _ := strconv.ParseInt(id, 10, 64)
		group, err := c.service.GetUserGroup(ctx, gid)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errUserGroupGet)
		}
		if group == nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return userGroupObservation(cr, group), nil
	}

	// No id yet: adopt a pre-existing group by name. Use the server-side
	// group_name filter (not an unfiltered list) — in OIDC mode Harbor accrues
	// many auto-created groups, so a paged unfiltered list may not contain ours.
	group, err := c.service.GetUserGroupByName(ctx, cr.Spec.ForProvider.GroupName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errUserGroupGet)
	}
	if group == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	meta.SetExternalName(cr, strconv.FormatInt(group.ID, 10))
	return userGroupObservation(cr, group), nil
}

// userGroupExternalID returns the Harbor group id stored as the external name,
// or "" if not set yet (external name defaults to metadata.name, non-numeric).
func userGroupExternalID(cr *v1beta1.UserGroup) string {
	en := meta.GetExternalName(cr)
	if en == "" || en == cr.GetName() {
		return ""
	}
	// Treat a non-positive id as "not set" — Harbor group ids are positive, and a
	// stray "0" would otherwise poison Observe (GetUserGroup(0) always errors).
	if id, err := strconv.ParseInt(en, 10, 64); err != nil || id <= 0 {
		return ""
	}
	return en
}

// userGroupID returns the Harbor group id for Update/Delete. The external name is
// authoritative (crossplane-runtime persists it reliably after Create); the
// status subresource is a best-effort fallback (it does not always persist, which
// is exactly why Delete must not rely on it).
func userGroupID(cr *v1beta1.UserGroup) int64 {
	if s := userGroupExternalID(cr); s != "" {
		if v, err := strconv.ParseInt(s, 10, 64); err == nil {
			return v
		}
	}
	if cr.Status.AtProvider.ID != nil {
		return *cr.Status.AtProvider.ID
	}
	return 0
}

func userGroupObservation(cr *v1beta1.UserGroup, group *harborclients.UserGroupStatus) managed.ExternalObservation {
	cr.Status.AtProvider.ID = &group.ID

	upToDate := cr.Spec.ForProvider.GroupType == group.GroupType &&
		cr.Spec.ForProvider.GroupName == group.GroupName

	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"group_name": []byte(group.GroupName),
			"group_id":   []byte(strconv.FormatInt(group.ID, 10)),
		},
	}
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.UserGroup)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserGroup)
	}

	cr.SetConditions(xpv1.Creating())

	// Prepare user group spec
	spec := &harborclients.UserGroupSpec{
		GroupName:   cr.Spec.ForProvider.GroupName,
		GroupType:   cr.Spec.ForProvider.GroupType,
		LdapGroupDn: cr.Spec.ForProvider.LdapGroupDn,
	}

	result, err := c.service.CreateUserGroup(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUserGroupCreate)
	}

	// Persist the id as the external name so the next Observe can get-by-id.
	meta.SetExternalName(cr, strconv.FormatInt(result.ID, 10))
	cr.Status.AtProvider.ID = &result.ID

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"group_name": []byte(result.GroupName),
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.UserGroup)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUserGroup)
	}

	id := userGroupID(cr)
	if id <= 0 {
		return managed.ExternalUpdate{}, errors.New("user group ID not found")
	}

	// Prepare updated user group spec
	spec := &harborclients.UserGroupSpec{
		GroupName:   cr.Spec.ForProvider.GroupName,
		GroupType:   cr.Spec.ForProvider.GroupType,
		LdapGroupDn: cr.Spec.ForProvider.LdapGroupDn,
	}

	_, err := c.service.UpdateUserGroup(ctx, id, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUserGroupUpdate)
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			"group_name": []byte(cr.Spec.ForProvider.GroupName),
		},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.UserGroup)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUserGroup)
	}

	cr.SetConditions(xpv1.Deleting())

	id := userGroupID(cr)
	if id <= 0 {
		// Never observed/created — nothing to delete.
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteUserGroup(ctx, id)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUserGroupDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
