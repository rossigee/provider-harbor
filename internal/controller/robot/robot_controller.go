/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"strings"
	"time"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	"github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	ctrlutil "github.com/rossigee/provider-harbor/internal/controller"
	"github.com/rossigee/provider-harbor/internal/tracing"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	errNotRobot    = "managed resource is not a Robot custom resource"
	errRobotDelete = "cannot delete Harbor robot"
	errNewClient   = "cannot create new Harbor client"
)

func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(v1beta1.RobotGroupVersionKind.Kind)
	log := logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))

	log.Info("setting up Robot controller")

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
			logger:       log,
		}),
		managed.WithLogger(log),
		managed.WithPollInterval(10 * time.Second),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.RobotGroupVersionKind),
		opts...,
	)

	err := ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Robot{}).
		Complete(r)

	if err != nil {
		log.Info("Robot controller setup failed", "error", err.Error())
	}
	return err
}

type connector struct {
	kube         client.Client
	newServiceFn func(context.Context, client.Client, resource.Managed) (harborclients.HarborClienter, error)
	logger       logging.Logger
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.Robot)
	if !ok {
		return nil, errors.New(errNotRobot)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		if c.logger != nil {
			c.logger.Info("Robot Connect failed", "name", mg.GetName(), "error", err.Error())
		}
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, logger: c.logger}, nil
}

type external struct {
	service harborclients.HarborClienter
	logger  logging.Logger
}

func (c *external) log() logging.Logger {
	if c.logger == nil {
		return logging.NewNopLogger()
	}
	return c.logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "robot.observe",
		tracing.SpanAttrs("Robot", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotRobot)
	}

	log := c.log().WithValues("name", cr.Name, "desiredName", cr.Spec.ForProvider.Name)
	log.Debug("observing Robot")

	robots, err := c.service.ListRobots(ctx, cr.Spec.ForProvider.ProjectID)
	if err != nil {
		log.Info("ListRobots failed", "error", err.Error())
		return managed.ExternalObservation{}, err
	}

	log.Debug("listed robots", "count", len(robots))

	externalName := ctrlutil.GetExternalName(cr)
	searchName := cr.Spec.ForProvider.Name
	if externalName != "" {
		searchName = externalName
	}
	if !strings.HasPrefix(searchName, "robot$") {
		searchName = "robot$" + searchName
	}

	log.Debug("searching for robot", "searchName", searchName)

	for _, robot := range robots {
		if robot.Name == searchName || robot.Name == cr.Spec.ForProvider.Name {
			log.Debug("found robot", "robotName", robot.Name, "id", robot.ID)

			ctrlutil.SetExternalName(cr, robot.Name)

			cr.Status.AtProvider.ID = &robot.ID
			if robot.Secret != "" {
				cr.Status.AtProvider.Secret = &robot.Secret
			}
			if robot.ExpiresAt != nil {
				et := metav1.NewTime(*robot.ExpiresAt)
				cr.Status.AtProvider.ExpiresAt = &et
			}
			t := metav1.NewTime(robot.CreationTime)
			cr.Status.AtProvider.CreationTime = &t
			ut := metav1.NewTime(robot.UpdateTime)
			cr.Status.AtProvider.UpdateTime = &ut

			upToDate := true
			if cr.Spec.ForProvider.Description != nil && robot.Description != nil && *cr.Spec.ForProvider.Description != *robot.Description {
				upToDate = false
			}
			if cr.Spec.ForProvider.ProjectID != nil && robot.ProjectID != nil && *cr.Spec.ForProvider.ProjectID != *robot.ProjectID {
				upToDate = false
			}

			log.Debug("robot exists", "upToDate", upToDate)
			cr.SetConditions(xpv1.Available())
			return managed.ExternalObservation{ResourceExists: true, ResourceUpToDate: upToDate}, nil
		}
	}

	log.Debug("robot not found, will create")
	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	_, span := tracing.StartSpan(ctx, "robot.create",
		tracing.SpanAttrs("Robot", tracing.ResourceName(mg), "create")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotRobot)
	}

	log := c.log().WithValues("name", cr.Name)
	log.Debug("creating Robot")

	spec := &harborclients.RobotSpec{
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		ExpiresIn:   cr.Spec.ForProvider.ExpiresIn,
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	robot, err := c.service.CreateRobot(ctx, spec)
	if err != nil {
		log.Info("CreateRobot failed", "error", err.Error())
		return managed.ExternalCreation{}, err
	}

	ctrlutil.SetExternalName(cr, robot.Name)

	log.Debug("created Robot", "robotName", robot.Name)
	return managed.ExternalCreation{}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	_, span := tracing.StartSpan(ctx, "robot.update",
		tracing.SpanAttrs("Robot", tracing.ResourceName(mg), "update")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotRobot)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalUpdate{}, errors.New("robot ID not set")
	}

	spec := &harborclients.RobotSpec{
		Name:        cr.Spec.ForProvider.Name,
		Description: cr.Spec.ForProvider.Description,
		ProjectID:   cr.Spec.ForProvider.ProjectID,
		ExpiresIn:   cr.Spec.ForProvider.ExpiresIn,
		Permissions: convertPermissions(cr.Spec.ForProvider.Permissions),
	}

	_, err := c.service.UpdateRobot(ctx, *cr.Status.AtProvider.ID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, err
	}

	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	_, span := tracing.StartSpan(ctx, "robot.delete",
		tracing.SpanAttrs("Robot", tracing.ResourceName(mg), "delete")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Robot)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotRobot)
	}

	if cr.Status.AtProvider.ID == nil {
		return managed.ExternalDelete{}, nil
	}

	err := c.service.DeleteRobot(ctx, *cr.Status.AtProvider.ID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, errRobotDelete)
	}

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}

func convertPermissions(perms []v1beta1.RobotPermission) []harborclients.RobotPermission {
	if len(perms) == 0 {
		return nil
	}
	result := make([]harborclients.RobotPermission, len(perms))
	for i, p := range perms {
		result[i] = harborclients.RobotPermission{
			Namespace: p.Namespace,
			Access:    p.Access,
		}
	}
	return result
}
