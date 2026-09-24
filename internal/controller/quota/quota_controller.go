/*
Copyright 2025 Crossplane Harbor Provider.
*/

package quota

import (
	"context"
	"time"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	quotav1beta1 "github.com/rossigee/provider-harbor/apis/quota/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	"github.com/rossigee/provider-harbor/internal/tracing"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotQuota        = "managed resource is not a Quota custom resource"
	errNewClient       = "cannot create new Harbor client"
	errProjectRefReq   = "project reference is required"
	errProjectNotReady = "project is not ready"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(quotav1beta1.QuotaGroupVersionKind.Kind)

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
		resource.ManagedKind(quotav1beta1.QuotaGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&quotav1beta1.Quota{}).
		Complete(r)
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*quotav1beta1.Quota)
	if !ok {
		return nil, errors.New(errNotQuota)
	}

	svc, err := c.newServiceFn(ctx, c.kube, cr)
	if err != nil {
		return nil, errors.Wrapf(err, "%s: %s", errNewClient, err.Error())
	}

	return &external{service: svc, kube: c.kube}, nil
}

type external struct {
	service harborclients.HarborClienter
	kube    client.Client
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "quota.observe",
		tracing.SpanAttrs("Quota", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*quotav1beta1.Quota)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotQuota)
	}

	projectID, err := c.resolveProjectRef(ctx, cr.Spec.ForProvider.ProjectRef)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	quota, err := c.service.GetQuotaForProject(ctx, projectID)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrapf(err, "failed to get quota")
	}

	cr.Status.AtProvider.ID = &quota.ID

	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: true}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, errors.New("quota is read-only")
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, errors.New("quota is read-only")
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	return managed.ExternalDelete{}, errors.New("quota is read-only")
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

func (c *external) resolveProjectRef(ctx context.Context, ref xpv1.Reference) (string, error) {
	if ref.Name == "" {
		return "", errors.New(errProjectRefReq)
	}

	project := &projectv1beta1.Project{}
	namespace := "default"

	if err := c.kube.Get(ctx, client.ObjectKey{Name: ref.Name, Namespace: namespace}, project); err != nil {
		return "", errors.Wrapf(err, "failed to resolve project reference")
	}

	if project.Status.AtProvider.ID == nil {
		return "", errors.New(errProjectNotReady)
	}

	return *project.Status.AtProvider.ID, nil
}
