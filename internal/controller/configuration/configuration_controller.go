/*
Copyright 2026 Crossplane Harbor Provider.
*/

// Package configuration reconciles Configuration resources, which manage
// Harbor's system-wide authentication (OIDC) settings.
package configuration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	xpcontroller "github.com/crossplane/crossplane-runtime/v2/pkg/controller"
	"github.com/crossplane/crossplane-runtime/v2/pkg/feature"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	sdkmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/configuration/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	"github.com/rossigee/provider-harbor/internal/tracing"
)

const (
	errNotConfiguration = "managed resource is not a Configuration custom resource"
	errNewClient        = "cannot create new Harbor client"
	errGetSecret        = "cannot get OIDC client secret"
	errGetConfig        = "cannot get Harbor configurations"
	errUpdateConfig     = "cannot update Harbor configurations"
)

// Setup adds a controller that reconciles Configuration managed resources.
func Setup(mgr ctrl.Manager, o xpcontroller.Options) error {
	name := managed.ControllerName(v1beta1.ConfigurationGroupVersionKind.Kind)

	opts := []managed.ReconcilerOption{
		managed.WithExternalConnector(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: harborclients.NewHarborClientFromProviderConfig,
		}),
		managed.WithLogger(logging.NewLogrLogger(mgr.GetLogger().WithValues("controller", name))),
		managed.WithPollInterval(1 * time.Minute),
	}

	if o.Features != nil && o.Features.Enabled(feature.EnableBetaManagementPolicies) {
		opts = append(opts, managed.WithManagementPolicies())
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1beta1.ConfigurationGroupVersionKind),
		opts...,
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1beta1.Configuration{}).
		Complete(r)
}

type connector struct {
	kube         client.Client
	newServiceFn func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error)
}

// Connect produces an ExternalClient for the Harbor instance named by the
// Configuration's ProviderConfig.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	if _, ok := mg.(*v1beta1.Configuration); !ok {
		return nil, errors.New(errNotConfiguration)
	}

	svc, err := c.newServiceFn(ctx, c.kube, mg)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{service: svc, kube: c.kube}, nil
}

type external struct {
	service harborclients.HarborClienter
	kube    client.Client
}

// Observe reads Harbor's configuration and reports whether it matches the
// desired state. Harbor does not return the OIDC client secret, so rotation is
// detected by comparing the hash of the referenced secret with the hash stored
// after the last successful write.
func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	_, span := tracing.StartSpan(ctx, "configuration.observe",
		tracing.SpanAttrs("Configuration", tracing.ResourceName(mg), "observe")...)
	defer span.End()

	cr, ok := mg.(*v1beta1.Configuration)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotConfiguration)
	}

	current, err := c.service.GetConfigurations(ctx)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, errGetConfig)
	}

	cr.Status.AtProvider.AuthMode = configStringValue(current.AuthMode)

	secretHash, err := c.clientSecretHash(ctx, cr)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	storedHash := ""
	if cr.Status.AtProvider.OIDCClientSecretHash != nil {
		storedHash = *cr.Status.AtProvider.OIDCClientSecretHash
	}

	upToDate := isUpToDate(cr.Spec.ForProvider, current, secretHash, storedHash)

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: upToDate,
	}, nil
}

// Create writes the desired configuration to Harbor.
func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	return managed.ExternalCreation{}, c.apply(ctx, mg)
}

// Update writes the desired configuration to Harbor.
func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	return managed.ExternalUpdate{}, c.apply(ctx, mg)
}

// Delete does not change Harbor's authentication settings. Removing the
// resource must not switch Harbor's login mode out from under its users.
func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	if _, ok := mg.(*v1beta1.Configuration); !ok {
		return managed.ExternalDelete{}, errors.New(errNotConfiguration)
	}
	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// apply writes the desired settings and records the client-secret hash that
// was written, so later rotations are detected.
func (c *external) apply(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1beta1.Configuration)
	if !ok {
		return errors.New(errNotConfiguration)
	}

	secret, err := c.clientSecret(ctx, cr)
	if err != nil {
		return err
	}

	if err := c.service.UpdateConfigurations(ctx, buildConfigurations(cr.Spec.ForProvider, secret)); err != nil {
		return errors.Wrap(err, errUpdateConfig)
	}

	cr.Status.AtProvider.OIDCClientSecretHash = nil
	if secret != nil {
		hash := hashSecret(*secret)
		cr.Status.AtProvider.OIDCClientSecretHash = &hash
	}

	return nil
}

// clientSecret returns the OIDC client secret value, or nil when no secret
// reference is set.
func (c *external) clientSecret(ctx context.Context, cr *v1beta1.Configuration) (*string, error) {
	oidc := cr.Spec.ForProvider.OIDC
	if oidc == nil || oidc.ClientSecretRef == nil {
		return nil, nil
	}

	ref := oidc.ClientSecretRef
	ns := ref.Namespace
	if ns == "" {
		ns = cr.GetNamespace()
	}

	secret := &corev1.Secret{}
	if err := c.kube.Get(ctx, client.ObjectKey{Namespace: ns, Name: ref.Name}, secret); err != nil {
		return nil, errors.Wrapf(err, "%s: %s/%s", errGetSecret, ns, ref.Name)
	}

	value, ok := secret.Data[ref.Key]
	if !ok {
		return nil, errors.Errorf("%s: key %q not found in secret %s/%s", errGetSecret, ref.Key, ns, ref.Name)
	}

	s := string(value)
	return &s, nil
}

// clientSecretHash returns the hash of the referenced client secret, or ""
// when no secret is referenced.
func (c *external) clientSecretHash(ctx context.Context, cr *v1beta1.Configuration) (string, error) {
	secret, err := c.clientSecret(ctx, cr)
	if err != nil || secret == nil {
		return "", err
	}
	return hashSecret(*secret), nil
}

// buildConfigurations maps the desired settings onto the Harbor request body.
// Only fields that are set are included, so unset fields keep their value in
// Harbor. An explicit empty string is sent as-is to clear a value.
func buildConfigurations(p v1beta1.ConfigurationParameters, clientSecret *string) *sdkmodels.Configurations {
	cfg := &sdkmodels.Configurations{
		AuthMode: p.AuthMode,
	}

	oidc := p.OIDC
	if oidc == nil {
		return cfg
	}

	cfg.OIDCName = oidc.Name
	cfg.OIDCEndpoint = oidc.Endpoint
	cfg.OIDCClientID = oidc.ClientID
	cfg.OIDCClientSecret = clientSecret
	cfg.OIDCScope = oidc.Scope
	cfg.OIDCUserClaim = oidc.UserClaim
	cfg.OIDCGroupsClaim = oidc.GroupsClaim
	cfg.OIDCAdminGroup = oidc.AdminGroup
	cfg.OIDCGroupFilter = oidc.GroupFilter
	cfg.OIDCVerifyCert = oidc.VerifyCert
	cfg.OIDCAutoOnboard = oidc.AutoOnboard
	cfg.OIDCLogout = oidc.Logout

	return cfg
}

// isUpToDate reports whether every field the spec sets matches Harbor's
// current configuration, and whether the client secret hash matches the last
// written value.
func isUpToDate(p v1beta1.ConfigurationParameters, current *sdkmodels.ConfigurationsResponse, secretHash, storedHash string) bool {
	if !stringPtrMatches(p.AuthMode, configStringValue(current.AuthMode)) {
		return false
	}

	oidc := p.OIDC
	if oidc == nil {
		return true
	}

	checks := []bool{
		stringPtrMatches(oidc.Name, configStringValue(current.OIDCName)),
		stringPtrMatches(oidc.Endpoint, configStringValue(current.OIDCEndpoint)),
		stringPtrMatches(oidc.ClientID, configStringValue(current.OIDCClientID)),
		stringPtrMatches(oidc.Scope, configStringValue(current.OIDCScope)),
		stringPtrMatches(oidc.UserClaim, configStringValue(current.OIDCUserClaim)),
		stringPtrMatches(oidc.GroupsClaim, configStringValue(current.OIDCGroupsClaim)),
		stringPtrMatches(oidc.AdminGroup, configStringValue(current.OIDCAdminGroup)),
		stringPtrMatches(oidc.GroupFilter, configStringValue(current.OIDCGroupFilter)),
		boolPtrMatches(oidc.VerifyCert, configBoolValue(current.OIDCVerifyCert)),
		boolPtrMatches(oidc.AutoOnboard, configBoolValue(current.OIDCAutoOnboard)),
		boolPtrMatches(oidc.Logout, configBoolValue(current.OIDCLogout)),
	}
	for _, ok := range checks {
		if !ok {
			return false
		}
	}

	if oidc.ClientSecretRef != nil && secretHash != storedHash {
		return false
	}

	return true
}

func stringPtrMatches(want *string, got *string) bool {
	if want == nil {
		return true
	}
	return got != nil && *got == *want
}

func boolPtrMatches(want *bool, got *bool) bool {
	if want == nil {
		return true
	}
	return got != nil && *got == *want
}

func configStringValue(item *sdkmodels.StringConfigItem) *string {
	if item == nil {
		return nil
	}
	v := item.Value
	return &v
}

func configBoolValue(item *sdkmodels.BoolConfigItem) *bool {
	if item == nil {
		return nil
	}
	v := item.Value
	return &v
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
