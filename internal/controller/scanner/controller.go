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
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/scanner/v1beta1"
	"github.com/rossigee/provider-harbor/internal/clients"
	"github.com/rossigee/provider-harbor/internal/features"
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
	// Features carries the provider feature gates (e.g. EnableBetaManagementPolicies).
	// This controller has its own Options struct rather than reusing the shared
	// crossplane-runtime controller.Options, so the feature set is threaded here.
	Features *feature.Flags
}

// Setup adds a controller that reconciles ScannerRegistration managed resources
func Setup(mgr ctrl.Manager, opts Options) error {
	name := managed.ControllerName(v1beta1.ScannerRegistrationGroupVersionKind.Kind)

	reconcilerOpts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:   mgr.GetClient(),
			logger: opts.Logger,
		}),
		managed.WithLogger(opts.Logger.WithValues("controller", name)),
		managed.WithPollInterval(10 * time.Minute),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorder(name))),
	}
	// Feature-gated options (e.g. Management Policies) appended when enabled.
	if opts.Features != nil && opts.Features.Enabled(features.EnableBetaManagementPolicies) {
		reconcilerOpts = append(reconcilerOpts, managed.WithManagementPolicies())
	}

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		For(&v1beta1.ScannerRegistration{}).
		Complete(managed.NewReconciler(mgr,
			resource.ManagedKind(v1beta1.ScannerRegistrationGroupVersionKind),
			reconcilerOpts...))
}

// connector is responsible for producing ExternalClients.
type connector struct {
	kube   client.Client
	logger logging.Logger
}

// Connect produces an ExternalClient by creating a Harbor client
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	_, ok := mg.(*v1beta1.ScannerRegistration)
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
	service clients.HarborClienter
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1beta1.ScannerRegistration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Observing Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	if cr.Spec.ForProvider.Name == "" {
		return managed.ExternalObservation{}, errors.New("scanner name is required")
	}

	// Use the UUID from status when available (set after creation); fall back to
	// a name-based list search for the initial observation before UUID is known.
	var status *clients.ScannerStatus
	var err error
	if cr.Status.AtProvider.UUID != nil && *cr.Status.AtProvider.UUID != "" {
		status, err = c.service.GetScannerRegistration(ctx, *cr.Status.AtProvider.UUID)
	} else {
		status, err = c.findByName(ctx, cr.Spec.ForProvider.Name)
	}
	if err != nil {
		// A real failure (auth/network/5xx) must surface, not be treated as absent.
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot observe Harbor scanner registration")
	}
	if status == nil {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	// Update status with observed values
	cr.Status.AtProvider.UUID = &status.UUID
	if status.CreateTime != (time.Time{}) {
		cr.Status.AtProvider.CreationTime = &metav1.Time{Time: status.CreateTime}
	}
	if status.UpdateTime != (time.Time{}) {
		cr.Status.AtProvider.UpdateTime = &metav1.Time{Time: status.UpdateTime}
	}

	upToDate := c.isUpToDate(cr, status)
	// Mark Available: the resource exists and is usable. Drift is signalled via
	// ResourceUpToDate (-> Update)/Synced, not by withholding Ready.
	cr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:    true,
		ResourceUpToDate:  upToDate,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

// findByName searches all scanner registrations for one matching name.
// Returns (nil, nil) when absent.
func (c *external) findByName(ctx context.Context, name string) (*clients.ScannerStatus, error) {
	all, err := c.service.ListScannerRegistrations(ctx)
	if err != nil {
		return nil, err
	}
	for _, s := range all {
		if s != nil && s.Name == name {
			return s, nil
		}
	}
	return nil, nil
}

func (c *external) isUpToDate(cr *v1beta1.ScannerRegistration, status *clients.ScannerStatus) bool {
	if cr.Spec.ForProvider.URL != status.URL {
		return false
	}
	if cr.Spec.ForProvider.Description != nil && status.Description != nil && *cr.Spec.ForProvider.Description != *status.Description {
		return false
	}
	if cr.Spec.ForProvider.Auth != nil && status.Auth != nil && *cr.Spec.ForProvider.Auth != *status.Auth {
		return false
	}
	if cr.Spec.ForProvider.Name != status.Name {
		return false
	}
	if cr.Spec.ForProvider.AccessCredential != nil && status.AccessCredential != nil && *cr.Spec.ForProvider.AccessCredential != *status.AccessCredential {
		return false
	}
	return true
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1beta1.ScannerRegistration)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Creating Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	cr.SetConditions(xpv1.Creating())

	spec := &clients.ScannerSpec{
		Name: cr.Spec.ForProvider.Name,
		URL:  cr.Spec.ForProvider.URL,
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

	// Store UUID in status so subsequent Observe calls use the stable ID.
	cr.Status.AtProvider.UUID = &status.UUID

	c.logger.Info("Successfully created Harbor scanner registration", "name", status.Name, "uuid", status.UUID)

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1beta1.ScannerRegistration)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Updating Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	spec := &clients.ScannerSpec{
		Name: cr.Spec.ForProvider.Name,
		URL:  cr.Spec.ForProvider.URL,
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
	scannerID := cr.Spec.ForProvider.Name // Fallback to name if UUID not available
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
	cr, ok := mg.(*v1beta1.ScannerRegistration)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotScannerRegistration)
	}

	c.logger.Debug("Deleting Harbor ScannerRegistration", "name", cr.Spec.ForProvider.Name)

	cr.SetConditions(xpv1.Deleting())

	// Use the UUID from the status for deletion
	scannerID := cr.Spec.ForProvider.Name // Fallback to name if UUID not available
	if cr.Status.AtProvider.UUID != nil {
		scannerID = *cr.Status.AtProvider.UUID
	}

	err := c.service.DeleteScannerRegistration(ctx, scannerID)
	if err != nil {
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete Harbor scanner registration")
	}

	c.logger.Info("Successfully deleted Harbor scanner registration", "name", cr.Spec.ForProvider.Name)

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return c.service.Close()
}
