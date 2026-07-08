/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scan

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/scan/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
)

const (
	errNotScan    = "managed resource is not a Scan custom resource"
	errScanDelete = "cannot delete Harbor scan"
	errNewClient  = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.ScanGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ScanGroupVersionKind),
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
		For(&v1beta1.Scan{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Scan)
	if !ok {
		return nil, errors.New(errNotScan)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

type external struct {
	service harborclients.HarborClienter
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "scan.observe",
		tracing.SpanAttrs("Scan", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotScan)
	}

	status, err := c.service.GetScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.Status = &status.Status
	cr.Status.AtProvider.CriticalCount = &status.CriticalCount
	cr.Status.AtProvider.HighCount = &status.HighCount
	cr.Status.AtProvider.MediumCount = &status.MediumCount
	cr.Status.AtProvider.LowCount = &status.LowCount
	st := metav1.NewTime(status.StartTime)
	cr.Status.AtProvider.StartTime = &st
	et := metav1.NewTime(status.EndTime)
	cr.Status.AtProvider.EndTime = &et

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, status.ID)
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "scan.create",
		tracing.SpanAttrs("Scan", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotScan)
	}

	err := c.service.TriggerScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "scan.update",
		tracing.SpanAttrs("Scan", mg.GetName(), "update")...)
	defer span.End()

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "scan.delete",
		tracing.SpanAttrs("Scan", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotScan)
	}

	err := c.service.StopScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errScanDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
