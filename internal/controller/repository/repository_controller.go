/*
Copyright 2024 Crossplane Harbor Provider.
*/

package repository

import (
	"context"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-harbor/apis/repository/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
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
		Complete(ratelimiter.NewReconciler(name, r, nil))
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
	_, span := tracing.StartSpan(ctx, "repository.observe",
		tracing.SpanAttrs("Repository", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRepository)
	}

	status, err := c.service.GetRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errRepositoryGet)
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

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, status.FullName)
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "repository.create",
		tracing.SpanAttrs("Repository", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRepository)
	}

	spec := &harborclients.RepositorySpec{
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
	}

	_, err := c.service.GetRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err == nil {
		// Repository already exists
		return managed.ExternalCreation{}, nil
	}

	_, err = c.service.UpdateRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errRepositoryCreate)
	}

	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "repository.update",
		tracing.SpanAttrs("Repository", tracing.ResourceName(mg), "update")...)
	defer span.End()

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
	_, span := tracing.StartSpan(ctx, "repository.delete",
		tracing.SpanAttrs("Repository", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Repository)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRepository)
	}

	err := c.service.DeleteRepository(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRepositoryDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
