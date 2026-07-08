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

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"

	"github.com/rossigee/provider-harbor/apis/project/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
)

const (
	errNotProject    = "managed resource is not a Project custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Harbor client"
	errProjectCreate = "cannot create Harbor project"
	errProjectGet    = "cannot get Harbor project"
	errProjectUpdate = "cannot update Harbor project"
	errProjectDelete = "cannot delete Harbor project"
)

// Setup adds a controller that reconciles Project managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.ProjectGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ProjectGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))))

	// Create the controller
	rl := ratelimiter.NewGlobal(10)
	_, err := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		For(&v1beta1.Project{}).
		Build(ratelimiter.NewReconciler(name, r, rl))
	if err != nil {
		return err
	}

	return nil
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
	_, ok := mg.(*v1beta1.Project)
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
	service harborclients.HarborClienter
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "project.observe",
		tracing.SpanAttrs("Project", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	// Check if the project exists in Harbor using external name if set, otherwise use desired name
	externalName := ctrlutil.GetExternalName(cr)
	projectName := cr.Spec.ForProvider.Name
	if externalName != "" {
		// Adoption scenario: use external name to find existing resource
		projectName = externalName
	}

	project, err := c.service.GetProject(ctx, projectName)
	if err != nil {
		// If project doesn't exist, we need to create it
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Set external name for future reference and adoption tracking
	ctrlutil.SetExternalName(cr, project.Name)

	// Update status with observed state
	cr.Status.AtProvider.ID = getStringPtr(project.ID)
	if project.CreatedAt != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: project.CreatedAt}
	}
	if project.UpdatedAt != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: project.UpdatedAt}
	}
	cr.Status.AtProvider.OwnerID = getInt64Ptr(project.OwnerID)
	cr.Status.AtProvider.OwnerName = getStringPtr(project.OwnerName)
	cr.Status.AtProvider.RepoCount = getInt64Ptr(project.RepoCount)
	cr.Status.AtProvider.ChartCount = getInt64Ptr(project.ChartCount)
	cr.Status.AtProvider.CurrentStorageUsage = getInt64Ptr(project.CurrentStorageUsage)

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
	_, span := tracing.StartSpan(ctx, "project.create",
		tracing.SpanAttrs("Project", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	cr.SetConditions(xpv1.Creating())

	// Prepare project spec with all parameters
	spec := &harborclients.ProjectSpec{
		Name:                     cr.Spec.ForProvider.Name,
		Public:                   getBoolValue(cr.Spec.ForProvider.Public),
		EnableContentTrust:       cr.Spec.ForProvider.EnableContentTrust,
		EnableContentTrustCosign: cr.Spec.ForProvider.EnableContentTrustCosign,
		AutoScanImages:           cr.Spec.ForProvider.AutoScanImages,
		PreventVulnerableImages:  cr.Spec.ForProvider.PreventVulnerableImages,
		Severity:                 cr.Spec.ForProvider.Severity,
		CVEAllowlist:             cr.Spec.ForProvider.CVEAllowlist,
		RegistryID:               cr.Spec.ForProvider.RegistryID,
		StorageLimit:             cr.Spec.ForProvider.StorageLimit,
		Metadata:                 cr.Spec.ForProvider.Metadata,
	}

	// Create project in Harbor
	status, err := c.service.CreateProject(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errProjectCreate)
	}

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, status.Name)

	// Update status with created resource info
	cr.Status.AtProvider.ID = getStringPtr("1") // Mock ID
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
	_, span := tracing.StartSpan(ctx, "project.update",
		tracing.SpanAttrs("Project", mg.GetName(), "update")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	// Prepare updated project spec with all parameters
	spec := &harborclients.ProjectSpec{
		Name:                     cr.Spec.ForProvider.Name,
		Public:                   getBoolValue(cr.Spec.ForProvider.Public),
		EnableContentTrust:       cr.Spec.ForProvider.EnableContentTrust,
		EnableContentTrustCosign: cr.Spec.ForProvider.EnableContentTrustCosign,
		AutoScanImages:           cr.Spec.ForProvider.AutoScanImages,
		PreventVulnerableImages:  cr.Spec.ForProvider.PreventVulnerableImages,
		Severity:                 cr.Spec.ForProvider.Severity,
		CVEAllowlist:             cr.Spec.ForProvider.CVEAllowlist,
		RegistryID:               cr.Spec.ForProvider.RegistryID,
		StorageLimit:             cr.Spec.ForProvider.StorageLimit,
		Metadata:                 cr.Spec.ForProvider.Metadata,
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
	_, span := tracing.StartSpan(ctx, "project.delete",
		tracing.SpanAttrs("Project", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Project)
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

func getStringPtr(s string) *string {
	return &s
}
