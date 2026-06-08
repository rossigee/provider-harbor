/*
Copyright 2024 Crossplane Harbor Provider.
*/

package usergroup

import (
	"context"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/usergroup/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotUserGroup       = "managed resource is not a UserGroup custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Harbor client"
	errUserGroupCreate    = "cannot create Harbor user group"
	errUserGroupGet       = "cannot get Harbor user group"
	errUserGroupUpdate    = "cannot update Harbor user group"
	errUserGroupDelete    = "cannot delete Harbor user group"
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
		Complete(ratelimiter.NewReconciler(name, r, nil))
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

	// Check if the user group exists in Harbor
	groupName := cr.Spec.ForProvider.GroupName
	groups, err := c.service.ListUserGroups(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errUserGroupGet)
	}

	var group *harborclients.UserGroupStatus
	for _, g := range groups {
		if g.GroupName == groupName {
			group = g
			break
		}
	}

	if group == nil {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed state
	cr.Status.AtProvider.ID = &group.ID

	// Check if resource is up to date
	upToDate := cr.Spec.ForProvider.GroupType == group.GroupType

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"group_name": []byte(group.GroupName),
		},
	}, nil
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

	// Update status with created resource info
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

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("user group ID not found")
	}

	// Prepare updated user group spec
	spec := &harborclients.UserGroupSpec{
		GroupName:   cr.Spec.ForProvider.GroupName,
		GroupType:   cr.Spec.ForProvider.GroupType,
		LdapGroupDn: cr.Spec.ForProvider.LdapGroupDn,
	}

	_, err := c.service.UpdateUserGroup(ctx, *cr.Status.AtProvider.ID, spec)
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

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, errors.New("user group ID not found")
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteUserGroup(ctx, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUserGroupDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
