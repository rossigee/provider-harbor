/*
Copyright 2024 Crossplane Harbor Provider.
*/

package project

import (
	"context"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/project/v1alpha1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotProject         = "managed resource is not a Project custom resource"
	errTrackPCUsage       = "cannot track ProviderConfig usage"
	errGetPC              = "cannot get ProviderConfig"
	errGetCreds           = "cannot get credentials"
	errNewClient          = "cannot create new Harbor client"
	errProjectCreate      = "cannot create Harbor project"
	errProjectGet         = "cannot get Harbor project"
	errProjectUpdate      = "cannot update Harbor project"
	errProjectDelete      = "cannot delete Harbor project"
)

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.ProjectGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.ProjectGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewNopLogger().WithValues("controller", name)),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Project{}).
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
	_, ok := mg.(*v1alpha1.Project)
	if !ok {
		return nil, errors.New(errNotProject)
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
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	// Check if the project exists in Harbor
	projectName := cr.Spec.ForProvider.Name
	project, err := c.service.GetProject(ctx, projectName)
	if err != nil {
		// If project doesn't exist, we need to create it
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed state
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID for now
	if project.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: project.CreatedAt}
	}
	cr.Status.AtProvider.RepoCount = getInt64Ptr(0) // Mock count

	// Check if resource is up to date
	upToDate := cr.Spec.ForProvider.Public == nil || *cr.Spec.ForProvider.Public == project.Public

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
		ConnectionDetails: managed.ConnectionDetails{
			"project_name": []byte(project.Name),
			"project_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	cr.SetConditions(xpv1.Creating())

	// Prepare project spec
	spec := &harborclients.ProjectSpec{
		Name:   cr.Spec.ForProvider.Name,
		Public: getBoolValue(cr.Spec.ForProvider.Public),
	}

	// Create project in Harbor
	status, err := c.service.CreateProject(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errProjectCreate)
	}

	// Update status with created resource info
	cr.Status.AtProvider.ID = getInt64Ptr(1) // Mock ID
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: status.CreatedAt}
	}

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{
			"project_name": []byte(status.Name),
			"project_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	// Prepare updated project spec
	spec := &harborclients.ProjectSpec{
		Name:   cr.Spec.ForProvider.Name,
		Public: getBoolValue(cr.Spec.ForProvider.Public),
	}

	// Update project in Harbor
	status, err := c.service.UpdateProject(ctx, cr.Spec.ForProvider.Name, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errProjectUpdate)
	}

	// Update status
	if status.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: time.Now()}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{
			"project_name": []byte(status.Name),
			"project_id":   []byte("1"), // Mock ID
		},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProject)
	}

	cr.SetConditions(xpv1.Deleting())

	// Delete project from Harbor
	err := c.service.DeleteProject(ctx, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errProjectDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// No cleanup needed for Harbor client
	return nil
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