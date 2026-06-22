/*
Copyright 2024 Crossplane Harbor Provider.
*/

package repository

import (
	"context"
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
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/repository/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotRepository    = "managed resource is not a Repository custom resource"
	errRepositoryCreate = "cannot create Harbor repository"
	errRepositoryGet    = "cannot get Harbor repository"
	errRepositoryUpdate = "cannot update Harbor repository"
	errRepositoryDelete = "cannot delete Harbor repository"
	errNewClient        = "cannot create new Harbor client"
)

// Setup adds a controller that reconciles Repository managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.RepositoryGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RepositoryGroupVersionKind),
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
		For(&v1beta1.Repository{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
}

// connector is responsible for producing ExternalClients.
type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

// Connect produces an ExternalClient by creating a Harbor client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Repository)
	if !ok {
		return nil, errors.New(errNotRepository)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

// external observes, then either creates, updates, or deletes an external resource.
type external struct {
	service harborclients.HarborClienter
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	status, err := c.service.GetRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err != nil {
		// A real failure (auth/network/5xx) must surface, not be treated as absent.
		return managed.ExternalObservation{}, errors.Wrap(err, errRepositoryGet)
	}
	if status == nil {
		// Not found -> let Crossplane create it.
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.FullName = &status.FullName
	cr.Status.AtProvider.ProjectID = &status.ProjectID
	cr.Status.AtProvider.ArtifactCount = &status.ArtifactCount
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t
	ut := metav1.NewTime(status.UpdateTime)
	cr.Status.AtProvider.UpdateTime = &ut
	if status.Description != "" {
		cr.Status.AtProvider.Description = &status.Description
	}

	upToDate := cr.Spec.ForProvider.Description == nil || status.Description == "" || *cr.Spec.ForProvider.Description == status.Description

	// Mark Available: the resource exists and is usable. Drift is signalled via
	// ResourceUpToDate (-> Update)/Synced, not by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	cr.SetConditions(xpv1.Creating())

	spec := &harborclients.RepositorySpec{
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
	}

	// Harbor repositories are auto-created on first push and cannot be explicitly
	// POSTed. GetRepository returning (nil, nil) means it does not yet exist;
	// UpdateRepository sets the description on an existing one.
	existing, err := c.service.GetRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRepositoryCreate)
	}
	if existing != nil {
		// Repository already exists (e.g. was pushed to before the CR was created).
		return managed.ExternalCreation{}, nil
	}

	// Repository does not exist yet. Harbor creates it lazily on first image push,
	// so we cannot force-create it via the API. Set description if provided.
	if spec.Description != nil {
		_, err = c.service.UpdateRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name, spec)
		if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, errRepositoryCreate)
		}
	}

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRepository)
	}

	spec := &harborclients.RepositorySpec{
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
	}

	_, err := c.service.UpdateRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errRepositoryUpdate)
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.DeleteRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRepositoryDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
