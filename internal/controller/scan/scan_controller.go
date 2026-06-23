/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scan

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/scan/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	controllerpkg "github.com/rossigee/provider-harbor/internal/controller"
)

const (
	errNotScan    = "managed resource is not a Scan custom resource"
	errScanDelete = "cannot delete Harbor scan"
	errNewClient  = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(v1beta1.ScanGroupVersionKind.Kind)

	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(1 * time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))),
	}
	// Feature-gated options (e.g. Management Policies) appended when enabled.
	reconcilerOpts = append(reconcilerOpts, controllerpkg.ReconcilerOptions(o)...)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ScanGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Scan{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
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
	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotScan)
	}

	status, err := c.service.GetScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	// (nil, nil) from GetScan means the underlying artifact does not exist
	// in Harbor yet. Signal not-exists so the reconciler triggers Create
	// (which calls TriggerScan once the artifact is present).
	if status == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
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

	// A scan is up-to-date and Available only when it completed successfully.
	// While still running (Scanning, Pending, …) or when not yet started (empty
	// status), we report ResourceExists:true but ResourceUpToDate:false so the
	// reconciler knows to keep polling without re-triggering.
	// Harbor success status values: "Success" or "stopped" (for stopped scans).
	upToDate := strings.EqualFold(status.Status, "success")
	if upToDate {
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotScan)
	}

	cr.SetConditions(xpv1.Creating())

	err := c.service.TriggerScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Scan)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotScan)
	}

	cr.SetConditions(xpv1.Deleting())

	err := c.service.StopScan(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.RepositoryName, cr.Spec.ForProvider.Reference)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errScanDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
