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

package scanner

import (
	"context"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/scanner/v1alpha1"
	"github.com/rossigee/provider-harbor/internal/clients"
)

const (
	errNotScannerRegistration = "managed resource is not a ScannerRegistration custom resource"
	errTrackPCUsage           = "cannot track ProviderConfig usage"
	errGetPC                  = "cannot get ProviderConfig"
	errGetCreds               = "cannot get credentials"
	errNewClient              = "cannot create new Service"
)

// Options contains options for controller setup
type Options struct {
	Logger       logging.Logger
	PollInterval string
}

// Setup adds a controller that reconciles ScannerRegistration managed resources
func Setup(mgr ctrl.Manager, opts Options) error {
	name := managed.ControllerName(v1alpha1.ScannerRegistration_GroupVersionKind.String())

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1alpha1.ScannerRegistration{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1alpha1.ScannerRegistration_GroupVersionKind),
			managed.WithExternalConnecter(&connector{
				kube:   mgr.GetClient(),
				logger: opts.Logger,
			}),
			managed.WithLogger(opts.Logger.WithValues("controller", name)),
			managed.WithPollInterval(10*time.Minute),
			managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name)))))
}

// connector is responsible for producing ExternalClients.
type connector struct {
	kube   client.Client
	logger logging.Logger
}

// Connect produces an ExternalClient by creating a Harbor client
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1alpha1.ScannerRegistration)
	if !ok {
		return nil, errors.New(errNotScannerRegistration)
	}

	harborClient, err := clients.NewHarborClientFromProviderConfig(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: harborClient, logger: c.logger}, nil
}

// external observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	service *clients.HarborClient
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.ScannerRegistration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Observing Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	if cr.Spec.ForProvider.Name == nil {
		return managed.ExternalObservation{}, errors.New("scanner name is required")
	}

	scannerName := *cr.Spec.ForProvider.Name

	// Check if scanner exists in Harbor
	status, err := c.service.GetScannerRegistration(ctx, scannerName)
	if err != nil {
		// Scanner doesn't exist yet
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Update status with observed values
	cr.Status.AtProvider.UUID = &status.UUID
	cr.Status.AtProvider.Name = &status.Name
	cr.Status.AtProvider.Description = status.Description
	cr.Status.AtProvider.URL = &status.URL
	cr.Status.AtProvider.Auth = status.Auth
	cr.Status.AtProvider.AccessCredential = status.AccessCredential

	// Format time strings
	createTimeStr := status.CreateTime.Format(time.RFC3339)
	updateTimeStr := status.UpdateTime.Format(time.RFC3339)
	cr.Status.AtProvider.CreateTime = &createTimeStr
	cr.Status.AtProvider.UpdateTime = &updateTimeStr

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  c.isUpToDate(cr, status),
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) isUpToDate(cr *v1alpha1.ScannerRegistration, status *clients.ScannerStatus) bool {
	if cr.Spec.ForProvider.URL != nil && *cr.Spec.ForProvider.URL != status.URL {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && status.Description != nil && *cr.Spec.ForProvider.Description != *status.Description {
		return false
	}
	if cr.Spec.ForProvider.Auth != nil && status.Auth != nil && *cr.Spec.ForProvider.Auth != *status.Auth {
		return false
	}
	// TODO: Add more fields to compare
	return true
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.ScannerRegistration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Creating Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	spec := &clients.ScannerSpec{
		Name: *cr.Spec.ForProvider.Name,
		URL:  *cr.Spec.ForProvider.URL,
	}

	if cr.Spec.ForProvider.Description != nil {
		spec.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Auth != nil {
		spec.Auth = cr.Spec.ForProvider.Auth
	}
	if cr.Spec.ForProvider.AccessCredential != nil {
		spec.AccessCredential = cr.Spec.ForProvider.AccessCredential
	}

	status, err := c.service.CreateScannerRegistration(ctx, spec)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create Harbor scanner registration")
	}

	c.logger.Info("Successfully created Harbor scanner registration", "name", status.Name, "uuid", status.UUID)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.ScannerRegistration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Updating Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	spec := &clients.ScannerSpec{
		Name: *cr.Spec.ForProvider.Name,
		URL:  *cr.Spec.ForProvider.URL,
	}

	if cr.Spec.ForProvider.Description != nil {
		spec.Description = cr.Spec.ForProvider.Description
	}
	if cr.Spec.ForProvider.Auth != nil {
		spec.Auth = cr.Spec.ForProvider.Auth
	}
	if cr.Spec.ForProvider.AccessCredential != nil {
		spec.AccessCredential = cr.Spec.ForProvider.AccessCredential
	}

	// Use the UUID from the status for updates
	scannerID := *cr.Spec.ForProvider.Name // Fallback to name if UUID not available
	if cr.Status.AtProvider.UUID != nil {
		scannerID = *cr.Status.AtProvider.UUID
	}

	status, err := c.service.UpdateScannerRegistration(ctx, scannerID, spec)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "cannot update Harbor scanner registration")
	}

	c.logger.Info("Successfully updated Harbor scanner registration", "name", status.Name, "uuid", status.UUID)

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.ScannerRegistration)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Deleting Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	// Use the UUID from the status for deletion
	scannerID := *cr.Spec.ForProvider.Name // Fallback to name if UUID not available
	if cr.Status.AtProvider.UUID != nil {
		scannerID = *cr.Status.AtProvider.UUID
	}

	err := c.service.DeleteScannerRegistration(ctx, scannerID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete Harbor scanner registration")
	}

	c.logger.Info("Successfully deleted Harbor scanner registration", "name", *cr.Spec.ForProvider.Name)

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
