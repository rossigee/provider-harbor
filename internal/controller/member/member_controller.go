/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

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

	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
)

const (
	errNotMember    = "managed resource is not a Member custom resource"
	errMemberDelete = "cannot delete Harbor member"
	errNewClient    = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1beta1.MemberGroupVersionKind.Kind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.MemberGroupVersionKind),
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
		For(&v1beta1.Member{}).
		Complete(ratelimiter.NewReconciler(name, r, nil))
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Member)
	if !ok {
		return nil, errors.New(errNotMember)
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
	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMember)
	}

	status, err := c.service.GetProjectMember(ctx, cr.Spec.ForProvider.ProjectID, cr.Spec.ForProvider.Username)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.MemberName = &status.MemberName
	cr.Status.AtProvider.MemberType = &status.MemberType
	cr.Status.AtProvider.Role = &status.Role
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t

	upToDate := cr.Spec.ForProvider.Role == "" || status.Role == "" || cr.Spec.ForProvider.Role == status.Role

	// Set external name for adoption tracking
	ctrlutil.SetExternalName(cr, "") // TODO: set appropriate identifier
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
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
