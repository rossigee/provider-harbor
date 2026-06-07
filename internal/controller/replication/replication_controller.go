/*
Copyright 2024 Crossplane Harbor Provider.
*/

package replication

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

	"github.com/rossigee/provider-harbor/apis/replication/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotReplication    = "managed resource is not a Replication custom resource"
	errReplicationDelete = "cannot delete Harbor replication policy"
	errNewClient         = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.ReplicationGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ReplicationGroupVersionKind),
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewNopLogger().WithValues("controller", name)),
		managed.WithPollInterval(1*time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Replication{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Replication)
	if !ok {
		return nil, errors.New(errNotReplication)
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
	cr, ok := mg.(*v1beta1.Replication)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotReplication)
	}

	policies, err := c.service.ListReplicationPolicies(ctx)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	for _, policy := range policies {
		if policy.Name == cr.Spec.ForProvider.Name {
			cr.Status.AtProvider.ID = &policy.ID
			cr.Status.AtProvider.Enabled = &policy.Enabled
			t := metav1.NewTime(policy.CreationTime)
			cr.Status.AtProvider.CreationTime = &t
			ut := metav1.NewTime(policy.UpdateTime)
			cr.Status.AtProvider.UpdateTime = &ut

			upToDate := true
			if cr.Spec.ForProvider.Description != nil && policy.Description != nil && *cr.Spec.ForProvider.Description != *policy.Description {
				upToDate = false
			}
			if cr.Spec.ForProvider.Enabled != nil && *cr.Spec.ForProvider.Enabled != policy.Enabled {
				upToDate = false
			}

			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
		}
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Replication)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotReplication)
	}

	spec := &harborclients.ReplicationPolicySpec{
		Name:            cr.Spec.ForProvider.Name,
		Description:     cr.Spec.ForProvider.Description,
		SourceRegistry:  cr.Spec.ForProvider.SourceRegistry,
		Trigger:         cr.Spec.ForProvider.Trigger,
		DeleteSourceTag: cr.Spec.ForProvider.DeleteSourceTag,
		Override:        cr.Spec.ForProvider.Override,
		Enabled:         cr.Spec.ForProvider.Enabled,
	}

	if len(cr.Spec.ForProvider.Filters) > 0 {
		spec.Filters = make([]harborclients.ReplicationPolicyFilter, len(cr.Spec.ForProvider.Filters))
		for i, f := range cr.Spec.ForProvider.Filters {
			spec.Filters[i] = harborclients.ReplicationPolicyFilter{
				Type:  f.Type,
				Value: f.Value,
			}
		}
	}

	spec.DestinationReg = &harborclients.ReplicationPolicyDestination{
		Name:      cr.Spec.ForProvider.DestinationReg.Name,
		Namespace: cr.Spec.ForProvider.DestinationReg.Namespace,
		URL:       cr.Spec.ForProvider.DestinationReg.URL,
	}

	_, err := c.service.CreateReplicationPolicy(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Replication)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotReplication)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("policy ID not set")
	}

	spec := &harborclients.ReplicationPolicySpec{
		Name:            cr.Spec.ForProvider.Name,
		Description:     cr.Spec.ForProvider.Description,
		Trigger:         cr.Spec.ForProvider.Trigger,
		DeleteSourceTag: cr.Spec.ForProvider.DeleteSourceTag,
		Override:        cr.Spec.ForProvider.Override,
		Enabled:         cr.Spec.ForProvider.Enabled,
	}

	_, err := c.service.UpdateReplicationPolicy(ctx, *cr.Status.AtProvider.ID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Replication)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotReplication)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteReplicationPolicy(ctx, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errReplicationDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
