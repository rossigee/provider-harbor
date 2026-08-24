/*
Copyright 2024 Crossplane Harbor Provider.
*/

package artifact

import (
	"context"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-harbor/apis/artifact/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
)

const (
	errNotArtifact    = "managed resource is not an Artifact custom resource"
	errArtifactDelete = "cannot delete Harbor artifact"
	errNewClient      = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.ArtifactGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ArtifactGroupVersionKind),
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
		For(&v1beta1.Artifact{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Artifact)
	if !ok {
		return nil, errors.New(errNotArtifact)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: %s", errNewClient, err.Error())
	}

	if svc == nil {
		return nil, errors.New("artifact: Connect: service is nil after creation")
	}

	return &external{service: svc, kube: c.kube}, nil
}

type external struct {
	service harborclients.HarborClienter
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "artifact.observe",
		tracing.SpanAttrs("Artifact", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Artifact)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotArtifact)
	}

	// Verify service is initialized
	if c.service == nil {
		return managed.ExternalObservation{}, errors.New("artifact: service is nil")
	}

	projectID := cr.Spec.ForProvider.ProjectID
	repoName := cr.Spec.ForProvider.RepositoryName
	reference := cr.Spec.ForProvider.Reference

	status, err := c.service.GetArtifact(ctx, projectID, repoName, reference)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrapf(err, "failed to get artifact projectID=%s repo=%s ref=%s", projectID, repoName, reference)
	}

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.Digest = &status.Digest
	cr.Status.AtProvider.Size = &status.Size
	cr.Status.AtProvider.PullCount = &status.PullCount
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t
	ut := metav1.NewTime(status.UpdateTime)
	cr.Status.AtProvider.UpdateTime = &ut
	cr.Status.AtProvider.VulnerabilityCount = &status.VulnerabilityCount

	ctrlutil.SetExternalName(cr, status.Digest)

	// Explicitly patch status to API server - managed reconciler may not auto-persist custom status fields
	if err := c.kube.Status().Patch(ctx, cr, client.MergeFrom(cr)); err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "failed to patch artifact status")
	}

	// Report as up-to-date so the managed reconciler sets the Ready condition
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "artifact.create",
		tracing.SpanAttrs("Artifact", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Artifact)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotArtifact)
	}

	// For read-only artifacts, immediately populate status
	// so the resource becomes ready after creation
	status, err := c.service.GetArtifact(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to get artifact during create")
	}

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.Digest = &status.Digest
	cr.Status.AtProvider.Size = &status.Size
	cr.Status.AtProvider.PullCount = &status.PullCount
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t
	ut := metav1.NewTime(status.UpdateTime)
	cr.Status.AtProvider.UpdateTime = &ut
	cr.Status.AtProvider.VulnerabilityCount = &status.VulnerabilityCount

	ctrlutil.SetExternalName(cr, status.Digest)

	// Explicitly patch status to API server
	if err := c.kube.Status().Patch(ctx, cr, client.MergeFrom(cr)); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to patch artifact status during create")
	}

	// Artifact is read-only - nothing to create externally
	return managed.ExternalCreation{ConnectionDetails: managed.ConnectionDetails{}}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "artifact.update",
		tracing.SpanAttrs("Artifact", tracing.ResourceName(mg), "update")...)
	defer span.End()

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "artifact.delete",
		tracing.SpanAttrs("Artifact", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Artifact)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotArtifact)
	}

	err := c.service.DeleteArtifact(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errArtifactDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
