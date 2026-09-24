/*
Copyright 2025 Crossplane Harbor Provider.
*/

package systeminfo

import (
	"context"
	"time"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	systeminfoav1beta1 "github.com/rossigee/provider-harbor/apis/systeminfo/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	"github.com/rossigee/provider-harbor/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotSystemInfo = "managed resource is not a SystemInfo custom resource"
	errNewClient     = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(systeminfoav1beta1.SystemInfoGroupVersionKind.Kind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(30 * time.Second),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(systeminfoav1beta1.SystemInfoGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&systeminfoav1beta1.SystemInfo{}).
		Complete(r)
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*systeminfoav1beta1.SystemInfo)
	if !ok {
		return nil, errors.New(errNotSystemInfo)
	}

	svc, err := c.newServiceFn(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: %s", errNewClient, err.Error())
	}

	return &external{service: svc}, nil
}

type external struct {
	service harborclients.HarborClienter
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "systeminfo.observe",
		tracing.SpanAttrs("SystemInfo", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*systeminfoav1beta1.SystemInfo)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotSystemInfo)
	}

	info, err := c.service.GetSystemInfo(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrapf(err, "failed to get system info")
	}

	cr.Status.AtProvider.HarborVersion = info.HarborVersion

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, errors.New("systeminfo is read-only")
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, errors.New("systeminfo is read-only")
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, errors.New("systeminfo is read-only")
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
