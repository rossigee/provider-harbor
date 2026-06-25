/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/member/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	controllerpkg "github.com/rossigee/provider-harbor/internal/controller"
)

const (
	errNotMember    = "managed resource is not a Member custom resource"
	errMemberGet    = "cannot get Harbor member"
	errMemberCreate = "cannot add Harbor member"
	errMemberUpdate = "cannot update Harbor member"
	errMemberDelete = "cannot delete Harbor member"
	errNewClient    = "cannot create new Harbor client"

	memberTypeUser  = "user"
	memberTypeGroup = "group"

	defaultGroupType int64 = 3
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(v1beta1.MemberGroupVersionKind.Kind)

	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(1 * time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))),
	}
	reconcilerOpts = append(reconcilerOpts, controllerpkg.ReconcilerOptions(o)...)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.MemberGroupVersionKind),
		reconcilerOpts...)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Member{}).
		Complete(ratelimiter.NewReconciler(name, r, ratelimiter.NewGlobal(1)))
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

// memberID returns the Harbor member id from the external name, or "" when not yet set.
func memberID(cr *v1beta1.Member) string {
	en := meta.GetExternalName(cr)
	if en == "" || en == cr.GetName() {
		return ""
	}
	return en
}

// entityKey returns the Harbor member entity type char ("u" or "g") and name.
func entityKey(cr *v1beta1.Member) (entityType string, entityName string, err error) {
	switch cr.Spec.ForProvider.Type {
	case memberTypeUser:
		if cr.Spec.ForProvider.Username == nil || *cr.Spec.ForProvider.Username == "" {
			return "", "", fmt.Errorf("username is required when type is %q", memberTypeUser)
		}
		return "u", *cr.Spec.ForProvider.Username, nil
	case memberTypeGroup:
		if cr.Spec.ForProvider.GroupName == nil || *cr.Spec.ForProvider.GroupName == "" {
			return "", "", fmt.Errorf("groupName is required when type is %q", memberTypeGroup)
		}
		return "g", *cr.Spec.ForProvider.GroupName, nil
	default:
		return "", "", fmt.Errorf("unknown member type %q: must be %q or %q", cr.Spec.ForProvider.Type, memberTypeUser, memberTypeGroup)
	}
}

func resolvedGroupType(cr *v1beta1.Member) int64 {
	if cr.Spec.ForProvider.GroupType != nil {
		return *cr.Spec.ForProvider.GroupType
	}
	return defaultGroupType
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotMember)
	}

	if id := memberID(cr); id != "" {
		status, err := c.service.GetProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id)
		if err != nil {
			return managed.ExternalObservation{}, errors.Wrap(err, errMemberGet)
		}
		if status == nil {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return applyObservation(cr, status), nil
	}

	// Adopt pre-existing member by entity name.
	eType, eName, err := entityKey(cr)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errMemberGet)
	}
	status, err := c.service.FindProjectMember(ctx, cr.Spec.ForProvider.ProjectID, eType, eName)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errMemberGet)
	}
	if status == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}
	meta.SetExternalName(cr, status.ID)
	return applyObservation(cr, status), nil
}

func applyObservation(cr *v1beta1.Member, status *harborclients.MemberStatus) managed.ExternalObservation {
	cr.Status.AtProvider.ID = &status.ID
	cr.Status.AtProvider.MemberName = &status.MemberName
	cr.Status.AtProvider.MemberType = &status.MemberType
	cr.Status.AtProvider.Role = &status.Role
	t := metav1.NewTime(status.CreationTime)
	cr.Status.AtProvider.CreationTime = &t
	upToDate := cr.Spec.ForProvider.Role == "" || status.Role == "" || cr.Spec.ForProvider.Role == status.Role
	cr.SetConditions(xpv1.Available())
	return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotMember)
	}
	cr.SetConditions(xpv1.Creating())

	var (
		id  string
		err error
	)
	switch cr.Spec.ForProvider.Type {
	case memberTypeUser:
		if cr.Spec.ForProvider.Username == nil {
			return managed.ExternalCreation{}, errors.New(errMemberCreate + ": username required for type=user")
		}
		id, err = c.service.AddProjectUserMember(ctx, cr.Spec.ForProvider.ProjectID, *cr.Spec.ForProvider.Username, cr.Spec.ForProvider.Role)
	case memberTypeGroup:
		if cr.Spec.ForProvider.GroupName == nil {
			return managed.ExternalCreation{}, errors.New(errMemberCreate + ": groupName required for type=group")
		}
		id, err = c.service.AddProjectGroupMember(ctx, cr.Spec.ForProvider.ProjectID, *cr.Spec.ForProvider.GroupName, resolvedGroupType(cr), cr.Spec.ForProvider.Role)
	default:
		return managed.ExternalCreation{}, fmt.Errorf("%s: unknown type %q", errMemberCreate, cr.Spec.ForProvider.Type)
	}
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errMemberCreate)
	}

	meta.SetExternalName(cr, id)
	cr.Status.AtProvider.ID = &id
	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotMember)
	}
	id := memberID(cr)
	if id == "" {
		return managed.ExternalUpdate{}, errors.New(errMemberUpdate + ": member id unknown")
	}
	if err := c.service.UpdateProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id, cr.Spec.ForProvider.Role); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errMemberUpdate)
	}
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1beta1.Member)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotMember)
	}
	cr.SetConditions(xpv1.Deleting())

	id := memberID(cr)
	if id == "" {
		return managed.ExternalDelete{}, nil
	}
	if err := c.service.DeleteProjectMemberByID(ctx, cr.Spec.ForProvider.ProjectID, id); err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errMemberDelete)
	}
	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
