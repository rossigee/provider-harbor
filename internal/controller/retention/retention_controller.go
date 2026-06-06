/*
Copyright 2024 Crossplane Harbor Provider.
*/

package retention

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

	"github.com/rossigee/provider-harbor/apis/retention/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotRetention    = "managed resource is not a Retention custom resource"
	errRetentionDelete = "cannot delete Harbor retention policy"
	errNewClient       = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.RetentionGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RetentionGroupVersionKind),
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
		For(&v1beta1.Retention{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (*harborclients.HarborClient, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Retention)
	if !ok {
		return nil, errors.New(errNotRetention)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc}, nil
}

type external struct {
	service *harborclients.HarborClient
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.Retention)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRetention)
	}

	policies, err := c.service.ListRetentionPolicies(ctx, cr.Spec.ForProvider.ProjectID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	for _, policy := range policies {
		if policy.ProjectID == cr.Spec.ForProvider.ProjectID {
			cr.Status.AtProvider.ID = &policy.ID
			cr.Status.AtProvider.Enabled = &policy.Enabled
			t := metav1.NewTime(policy.CreationTime)
			cr.Status.AtProvider.CreationTime = &t
			ut := metav1.NewTime(policy.UpdateTime)
			cr.Status.AtProvider.UpdateTime = &ut
			return managed.ExternalObservation{ResourceExists: true}, nil
		}
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Retention)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRetention)
	}

	spec := &harborclients.RetentionPolicySpec{
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		Description: cr.Spec.ForProvider.Description,
		Trigger:     cr.Spec.ForProvider.Trigger,
		Enabled:     cr.Spec.ForProvider.Enabled,
	}

	if len(cr.Spec.ForProvider.Rules) > 0 {
		spec.Rules = make([]harborclients.RetentionPolicyRule, len(cr.Spec.ForProvider.Rules))
		for i, r := range cr.Spec.ForProvider.Rules {
			spec.Rules[i] = harborclients.RetentionPolicyRule{
				RuleType:     r.RuleType,
				TagSelectors: r.TagSelectors,
				Parameters:   convertStringMap(r.Parameters),
			}
		}
	}

	_, err := c.service.CreateRetentionPolicy(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Retention)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRetention)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("policy ID not set")
	}

	spec := &harborclients.RetentionPolicySpec{
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		Description: cr.Spec.ForProvider.Description,
		Trigger:     cr.Spec.ForProvider.Trigger,
		Enabled:     cr.Spec.ForProvider.Enabled,
	}

	_, err := c.service.UpdateRetentionPolicy(ctx, cr.Spec.ForProvider.ProjectID, *cr.Status.AtProvider.ID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Retention)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRetention)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteRetentionPolicy(ctx, cr.Spec.ForProvider.ProjectID, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRetentionDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}

func convertStringMap(m map[string]string) map[string]interface{} {
	if len(m) == 0 {
		return nil
	}
	result := make(map[string]interface{}, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}
