/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"encoding/json"
	"time"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/project/v1alpha1"
	"github.com/rossigee/provider-harbor/apis/v1beta1"
	"github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotProject    = "managed resource is not a Project custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Service"
)

// Options contains options for controller setup
type Options struct {
	Logger       logging.Logger
	PollInterval string
}

// Setup creates all Harbor controllers using the native Harbor client
func Setup(mgr ctrl.Manager, opts Options) error {
	opts.Logger.Info("Setting up Harbor controllers")
	
	if err := setupProjectController(mgr, opts); err != nil {
		return err
	}
	
	opts.Logger.Info("Harbor controllers setup completed")
	return nil
}

// setupProjectController sets up a controller for Harbor projects
func setupProjectController(mgr ctrl.Manager, opts Options) error {
	opts.Logger.Info("Setting up Harbor Project controller")
	
	name := managed.ControllerName(v1alpha1.Project_GroupVersionKind.String())
	
	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	
	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.Project{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.Project_GroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:   mgr.GetClient(),
				usage:  resource.NewProviderConfigUsageTracker(mgr.GetClient(), &v1beta1.ProviderConfigUsage{}),
				logger: opts.Logger,
			}),
			managed.WithLogger(opts.Logger.WithValues("controller", name)),
			managed.WithPollInterval(10*time.Minute),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
			managed.WithConnectionPublishers(cps...)))
}

// connector is responsible for producing ExternalClients.
type connector struct {
	kube   client.Client
	usage  resource.Tracker
	logger logging.Logger
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return nil, errors.New(errNotProject)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &v1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	// Parse Harbor credentials from extracted data
	harborCreds := map[string]string{}
	if err := json.Unmarshal(data, &harborCreds); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal credentials")
	}
	
	config := &clients.HarborConfig{
		URL:      harborCreds["url"],
		Username: harborCreds["username"], 
		Password: harborCreds["password"],
		Insecure: harborCreds["insecure"] == "true",
	}

	svc, err := clients.NewHarborClient(config)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, logger: c.logger}, nil
}

// external observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	service *clients.HarborClient
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotProject)
	}

	// These fmt statements should be removed in the real implementation.
	c.logger.Debug("Observing Harbor Project", "name", cr.Spec.ForProvider.Name)

	if cr.Spec.ForProvider.Name == nil {
		return managed.ExternalObservation{}, errors.New("project name is required")
	}

	projectName := *cr.Spec.ForProvider.Name
	
	// Check if project exists in Harbor
	status, err := c.service.GetProject(ctx, projectName)
	if err != nil {
		// Project doesn't exist yet
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed values
	cr.Status.AtProvider.ID = &projectName
	cr.Status.AtProvider.Name = &status.Name
	cr.Status.AtProvider.Public = &status.Public
	cr.Status.AtProvider.ProjectID = new(float64)
	*cr.Status.AtProvider.ProjectID = 1 // TODO: Get actual project ID

	return managed.ExternalObservation{
		// Return false when the external resource does not exist. This lets
		// the managed resource reconciler know that it needs to call Create to
		// (re)create the resource, or that it has successfully been deleted.
		ResourceExists: true,

		// Return false when the external resource exists, but it not up to date
		// with the desired managed resource state. This lets the managed
		// resource reconciler know that it needs to call Update.
		ResourceUpToDate: c.isUpToDate(cr, status),

		// Return any details that may be required to connect to the external
		// resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) isUpToDate(cr *v1alpha1.Project, status *clients.ProjectStatus) bool {
	if cr.Spec.ForProvider.Public != nil && *cr.Spec.ForProvider.Public != status.Public {
		return false
	}
	// TODO: Add more fields to compare
	return true
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotProject)
	}

	c.logger.Debug("Creating Harbor Project", "name", cr.Spec.ForProvider.Name)

	spec := &clients.ProjectSpec{
		Name:   *cr.Spec.ForProvider.Name,
		Public: false, // Default to private
	}
	
	if cr.Spec.ForProvider.Public != nil {
		spec.Public = *cr.Spec.ForProvider.Public
	}

	status, err := c.service.CreateProject(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create Harbor project")
	}

	c.logger.Info("Successfully created Harbor project", "name", status.Name, "public", status.Public)

	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotProject)
	}

	c.logger.Debug("Updating Harbor Project", "name", cr.Spec.ForProvider.Name)

	spec := &clients.ProjectSpec{
		Name:   *cr.Spec.ForProvider.Name,
		Public: false, // Default to private
	}
	
	if cr.Spec.ForProvider.Public != nil {
		spec.Public = *cr.Spec.ForProvider.Public
	}

	status, err := c.service.UpdateProject(ctx, *cr.Spec.ForProvider.Name, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update Harbor project")
	}

	c.logger.Info("Successfully updated Harbor project", "name", status.Name, "public", status.Public)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Project)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotProject)
	}

	c.logger.Debug("Deleting Harbor Project", "name", cr.Spec.ForProvider.Name)

	err := c.service.DeleteProject(ctx, *cr.Spec.ForProvider.Name)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete Harbor project")
	}

	c.logger.Info("Successfully deleted Harbor project", "name", *cr.Spec.ForProvider.Name)
	
	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}