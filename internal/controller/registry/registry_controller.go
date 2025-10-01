/*
Copyright 2024 Crossplane Harbor Provider.
*/

package registry

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

	"github.com/rossigee/provider-harbor/apis/registry/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotRegistry       = "managed resource is not a Registry custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errNewClient         = "cannot create new Harbor client"
	errRegistryCreate    = "cannot create Harbor registry"
	errRegistryGet       = "cannot get Harbor registry"
	errRegistryUpdate    = "cannot update Harbor registry"
	errRegistryDelete    = "cannot delete Harbor registry"
)

// Setup adds a controller that reconciles Registry managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.RegistryGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RegistryGroupVersionKind),
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
		For(&v1beta1.Registry{}).
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
	_, ok := mg.(*v1beta1.Registry)
	if !ok {
		return nil, errors.New(errNotRegistry)
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
	cr, ok := mg.(*v1beta1.Registry)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRegistry)
	}

	// Check if the registry exists in Harbor
	registryName := cr.Spec.ForProvider.Name
	registry, err := c.service.GetRegistry(ctx, registryName)
	if err != nil {
		// If registry doesn't exist, we need to create it
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed state
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID for now
	if registry.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: registry.CreatedAt}
	}
	if registry.UpdatedAt != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: registry.UpdatedAt}
	}
	cr.Status.AtProvider.Status = getStringPtr("healthy") // Mock status

	// Check if resource is up to date
	upToDate := (cr.Spec.ForProvider.Description == nil || registry.Description == nil || *cr.Spec.ForProvider.Description == *registry.Description) &&
		cr.Spec.ForProvider.URL == registry.URL &&
		cr.Spec.ForProvider.Type == registry.Type

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"registry_name": []byte(registry.Name),
			"registry_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Registry)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRegistry)
	}

	cr.SetConditions(xpv1.Creating())

	// Prepare registry spec
	spec := &harborclients.RegistrySpec{
		Name: cr.Spec.ForProvider.Name,
		Type: cr.Spec.ForProvider.Type,
		URL:  cr.Spec.ForProvider.URL,
	}

	if cr.Spec.ForProvider.Description != nil {
		spec.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Insecure != nil {
		spec.Insecure = *cr.Spec.ForProvider.Insecure
	}

	// Handle credentials if provided
	if cr.Spec.ForProvider.Credential != nil {
		cred := &harborclients.RegistryCredential{}
		if cr.Spec.ForProvider.Credential.Type != nil {
			cred.Type = *cr.Spec.ForProvider.Credential.Type
		}
		if cr.Spec.ForProvider.Credential.AccessKey != nil {
			cred.AccessKey = *cr.Spec.ForProvider.Credential.AccessKey
		}
		// Handle secret reference
		if cr.Spec.ForProvider.Credential.AccessSecretRef != nil {
			secret, err := c.getSecretFromRef(ctx, cr)
			if err != nil {
				return managed.ExternalCreation{}, errors.Wrap(err, "cannot get access secret")
			}
			cred.AccessSecret = secret
		}
		spec.Credential = cred
	}

	// Create registry in Harbor
	status, err := c.service.CreateRegistry(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRegistryCreate)
	}

	// Update status with created resource info
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: status.CreatedAt}
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"registry_name": []byte(status.Name),
			"registry_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Registry)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRegistry)
	}

	// Prepare updated registry spec
	spec := &harborclients.RegistrySpec{
		Name: cr.Spec.ForProvider.Name,
		Type: cr.Spec.ForProvider.Type,
		URL:  cr.Spec.ForProvider.URL,
	}

	if cr.Spec.ForProvider.Description != nil {
		spec.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Insecure != nil {
		spec.Insecure = *cr.Spec.ForProvider.Insecure
	}

	// Handle credentials if provided
	if cr.Spec.ForProvider.Credential != nil {
		cred := &harborclients.RegistryCredential{}
		if cr.Spec.ForProvider.Credential.Type != nil {
			cred.Type = *cr.Spec.ForProvider.Credential.Type
		}
		if cr.Spec.ForProvider.Credential.AccessKey != nil {
			cred.AccessKey = *cr.Spec.ForProvider.Credential.AccessKey
		}
		if cr.Spec.ForProvider.Credential.AccessSecretRef != nil {
			secret, err := c.getSecretFromRef(ctx, cr)
			if err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, "cannot get access secret")
			}
			cred.AccessSecret = secret
		}
		spec.Credential = cred
	}

	// Update registry in Harbor
	status, err := c.service.UpdateRegistry(ctx, cr.Spec.ForProvider.Name, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRegistryUpdate)
	}

	// Update status
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: time.Now()}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			"registry_name": []byte(status.Name),
			"registry_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Registry)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRegistry)
	}

	cr.SetConditions(xpv1.Deleting())

	// Delete registry from Harbor
	err := c.service.DeleteRegistry(ctx, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRegistryDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// No cleanup needed for Harbor client
	return nil
}

// Helper function to get secret from secret reference
func (c *external) getSecretFromRef(ctx context.Context, cr *v1beta1.Registry) (string, error) {
	// This would need to be implemented to read from Kubernetes secret
	// For now, return a placeholder
	return "mock-secret", nil
}

// Helper functions
func getInt64Ptr(i int64) *int64 {
	return &i
}

func getStringPtr(s string) *string {
	return &s
}