/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

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
	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotMember    = "managed resource is not a Member custom resource"
	errMemberDelete = "cannot delete Harbor member"
	errNewClient    = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(v1beta1.MemberGroupVersionKind.Kind)
	log := logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))

	log.Info("setting up Member controller")

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
			logger:       log,
		}),
		managed.WithLogger(log),
		managed.WithPollInterval(1 * time.Minute),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.MemberGroupVersionKind),
		opts...,
	)

	err := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Member{}).
		Complete(r)

	if err != nil {
		log.Info("Member controller setup failed", "error", err.Error())
	}
	return err
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
	logger       logging.Logger
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Member)
	if !ok {
		return nil, errors.New(errNotMember)
	}

	log := c.log().WithValues("name", mg.GetName())
	log.Debug("connecting Member resource")

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		log.Info("Member Connect failed", "error", err.Error())
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, kube: c.kube, logger: c.logger}, nil
}

func (c *connector) log() logging.Logger {
	if c.logger == nil {
		return logging.NewNopLogger()
	}
	return c.logger
}

type external struct {
	service harborclients.HarborClienter
	kube    client.Client
	logger  logging.Logger
}

func (c *external) log() logging.Logger {
	if c.logger == nil {
		return logging.NewNopLogger()
	}
	return c.logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "member.observe",
		tracing.SpanAttrs("Member", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMember)
	}

	log := c.log().WithValues("name", cr.GetName(), "project", cr.Spec.ForProvider.ProjectID, "user", cr.Spec.ForProvider.Username)
	log.Debug("observing Member")

	status, err := c.service.GetProjectMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username)
	if err != nil {
		if harborclients.IsNotFound(err) {
			log.Debug("member or project not found")
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		log.Info("GetProjectMember failed", "error", err.Error())
		return managed.ExternalObservation{}, err
	}
	if status == nil {
		log.Debug("member not found")
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	log.Debug("found member", "id", status.ID, "role", status.Role)

	// Snapshot cr before mutation for use in MergeFrom patch
	original := cr.DeepCopy()

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.MemberName = &status.MemberName
	cr.Status.AtProvider.MemberType = &status.MemberType
	cr.Status.AtProvider.Role = &status.Role
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t

	// Mark resource as ready/synced so status is persisted
	cr.SetConditions(xpv1.Available())

	// Persist status to API server using status subresource
	if c.kube != nil {
		if err := c.kube.Status().Patch(ctx, cr, client.MergeFrom(original)); err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, "failed to patch status")
		}
	}

	upToDate := cr.Spec.ForProvider.Role == "" || status.Role == "" || cr.Spec.ForProvider.Role == status.Role

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, status.MemberName)
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "member.create",
		tracing.SpanAttrs("Member", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMember)
	}

	err := c.service.AddProjectMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username, cr.Spec.ForProvider.Role)
	if err != nil {
		return managed.ExternalCreation{}, err
	}

	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "member.update",
		tracing.SpanAttrs("Member", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMember)
	}

	err := c.service.UpdateProjectMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username, cr.Spec.ForProvider.Role)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "member.delete",
		tracing.SpanAttrs("Member", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMember)
	}

	err := c.service.DeleteProjectMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errMemberDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
