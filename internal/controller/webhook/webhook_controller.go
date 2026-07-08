/*
Copyright 2024 Crossplane Harbor Provider.
*/

package webhook

import (
	"context"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-harbor/apis/webhook/v1beta1"
	"github.com/rossigee/provider-harbor/internal/clients"
	"github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"time"
)

const (
	errNotWebhook    = "managed resource is not a Webhook custom resource"
	errWebhookDelete = "cannot delete Harbor webhook"
	errNewClient     = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.WebhookGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.WebhookGroupVersionKind),
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
		For(&v1beta1.Webhook{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(10)))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Webhook)
	if !ok {
		return nil, errors.New(errNotWebhook)
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
	_, span := tracing.StartSpan(ctx, "webhook.observe",
		tracing.SpanAttrs("Webhook", mg.GetName(), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Webhook)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotWebhook)
	}

	webhooks, err := c.service.ListWebhooks(ctx, cr.Spec.ForProvider.ProjectID)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	for _, webhook := range webhooks {
		if webhook.Name == cr.Spec.ForProvider.Name {
			cr.Status.AtProvider.ID = &webhook.ID
			t := metav1.NewTime(webhook.CreationTime)
			cr.Status.AtProvider.CreationTime = &t
			ut := metav1.NewTime(webhook.UpdateTime)
			cr.Status.AtProvider.UpdateTime = &ut

			upToDate := true
			if cr.Spec.ForProvider.Description != nil && webhook.Description != nil && *cr.Spec.ForProvider.Description != *webhook.Description {
				upToDate = false
			}
			if cr.Spec.ForProvider.URL != "" && cr.Spec.ForProvider.URL != webhook.URL {
				upToDate = false
			}
			if len(cr.Spec.ForProvider.EventTypes) > 0 && len(webhook.EventTypes) > 0 {
				if len(cr.Spec.ForProvider.EventTypes) != len(webhook.EventTypes) {
					upToDate = false
				} else {
					for i, e := range cr.Spec.ForProvider.EventTypes {
						if e != webhook.EventTypes[i] {
							upToDate = false
							break
						}
					}
				}
			}

			// Set external name for adoption tracking
			ctrlutil.SetExternalName(cr, webhook.Name)
			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
		}
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "webhook.create",
		tracing.SpanAttrs("Webhook", mg.GetName(), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Webhook)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotWebhook)
	}

	spec := &harborclients.WebhookSpec{
		ProjectID:      cr.Spec.ForProvider.ProjectID,
		Name:           cr.Spec.ForProvider.Name,
		Description:    cr.Spec.ForProvider.Description,
		URL:            cr.Spec.ForProvider.URL,
		EventTypes:     cr.Spec.ForProvider.EventTypes,
		AuthHeader:     cr.Spec.ForProvider.AuthHeader,
		SkipCertVerify: *cr.Spec.ForProvider.SkipCertVerify,
	}

	_, err := c.service.CreateWebhook(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "webhook.update",
		tracing.SpanAttrs("Webhook", mg.GetName(), "update")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Webhook)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotWebhook)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("webhook ID not set")
	}

	spec := &harborclients.WebhookSpec{
		ProjectID:      cr.Spec.ForProvider.ProjectID,
		Name:           cr.Spec.ForProvider.Name,
		Description:    cr.Spec.ForProvider.Description,
		URL:            cr.Spec.ForProvider.URL,
		EventTypes:     cr.Spec.ForProvider.EventTypes,
		AuthHeader:     cr.Spec.ForProvider.AuthHeader,
		SkipCertVerify: *cr.Spec.ForProvider.SkipCertVerify,
	}

	_, err := c.service.UpdateWebhook(ctx, cr.Spec.ForProvider.ProjectID, *cr.Status.AtProvider.ID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "webhook.delete",
		tracing.SpanAttrs("Webhook", mg.GetName(), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Webhook)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotWebhook)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteWebhook(ctx, cr.Spec.ForProvider.ProjectID, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errWebhookDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
