/*
Copyright 2024 Crossplane Harbor Provider.
*/

package user

import (
	"context"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/v2/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/user/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotUser           = "managed resource is not a User custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errNewClient         = "cannot create new Harbor client"
	errUserCreate        = "cannot create Harbor user"
	errUserGet           = "cannot get Harbor user"
	errUserUpdate        = "cannot update Harbor user"
	errUserDelete        = "cannot delete Harbor user"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.UserGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.UserGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewNopLogger().WithValues("controller", name)),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.User{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	newServiceFn func(ctx context.Context, kube client.Client, mg resource.Managed) (*harborclients.HarborClient, error)
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.User)
	if !ok {
		return nil, errors.New(errNotUser)
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
	service *harborclients.HarborClient
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// Check if the user exists in Harbor
	username := cr.Spec.ForProvider.Username
	user, err := c.service.GetUser(ctx, username)
	if err != nil {
		// If user doesn't exist, we need to create it
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed state
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID for now
	if user.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: user.CreatedAt}
	}

	// Check if resource is up to date
	upToDate := cr.Spec.ForProvider.Email == user.Email &&
		(cr.Spec.ForProvider.SysAdminFlag == nil || *cr.Spec.ForProvider.SysAdminFlag == user.AdminFlag)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"username": []byte(user.Username),
			"user_id":  []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	cr.SetConditions(xpv1.Creating())

	// Prepare user spec
	spec := &harborclients.UserSpec{
		Username:  cr.Spec.ForProvider.Username,
		Email:     cr.Spec.ForProvider.Email,
		AdminFlag: getBoolValue(cr.Spec.ForProvider.SysAdminFlag),
	}

	// Handle password secret
	if cr.Spec.ForProvider.PasswordSecretRef != nil {
		// Get password from secret
		secret, err := c.getPasswordFromSecret(ctx, cr)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, "cannot get password from secret")
		}
		spec.Password = secret
	}

	// Create user in Harbor
	status, err := c.service.CreateUser(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errUserCreate)
	}

	// Update status with created resource info
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: status.CreatedAt}
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"username": []byte(status.Username),
			"user_id":  []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	// Prepare updated user spec
	spec := &harborclients.UserSpec{
		Username:  cr.Spec.ForProvider.Username,
		Email:     cr.Spec.ForProvider.Email,
		AdminFlag: getBoolValue(cr.Spec.ForProvider.SysAdminFlag),
	}

	// Handle password secret if provided
	if cr.Spec.ForProvider.PasswordSecretRef != nil {
		secret, err := c.getPasswordFromSecret(ctx, cr)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get password from secret")
		}
		spec.Password = secret
	}

	// Update user in Harbor
	status, err := c.service.UpdateUser(ctx, cr.Spec.ForProvider.Username, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUserUpdate)
	}

	// Update status
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: time.Now()}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			"username": []byte(status.Username),
			"user_id":  []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.User)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotUser)
	}

	cr.SetConditions(xpv1.Deleting())

	// Delete user from Harbor
	err := c.service.DeleteUser(ctx, cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errUserDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// No cleanup needed for Harbor client
	return nil
}

// Helper function to get password from secret
func (c *external) getPasswordFromSecret(ctx context.Context, cr *v1beta1.User) (string, error) {
	// This would need to be implemented to read from Kubernetes secret
	// For now, return a placeholder
	return "mock-password", nil
}

// Helper functions
func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func getInt64Ptr(i int64) *int64 {
	return &i
}