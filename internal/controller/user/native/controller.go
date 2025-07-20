package native

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/globallogicuki/provider-harbor/apis/user/v1alpha1"
	apisv1alpha1 "github.com/globallogicuki/provider-harbor/apis/v1alpha1"
	harborclients "github.com/globallogicuki/provider-harbor/internal/clients"
	"github.com/globallogicuki/provider-harbor/internal/features"
)

const (
	errNotUser      = "managed resource is not a User custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Client"
	errGetUser      = "cannot get user"
	errCreateUser   = "cannot create user"
	errUpdateUser   = "cannot update user"
	errDeleteUser   = "cannot delete user"
	errGetPassword  = "cannot get password from secret"
)

// Setup adds a controller that reconciles User managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.UserGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:        mgr.GetClient(),
			usage:       resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newClientFn: NewHarborClient,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.User{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube        client.Client
	usage       resource.Tracker
	newClientFn func(c *harborclients.HarborCLI) Client
}

// Connect typically produces an ExternalClient by:
// 1. Tracking that the managed resource is using a ProviderConfig.
// 2. Getting the managed resource's ProviderConfig.
// 3. Getting the credentials specified by the ProviderConfig.
// 4. Using the credentials to form a client.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return nil, errors.New(errNotUser)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	v2Client, err := harborclients.NewHarborCLI(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{
		client: c.newClientFn(v2Client),
		kube:   c.kube,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	client Client
	kube   client.Client
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUser)
	}

	// If we have an external name, try to get the user by username
	username := meta.GetExternalName(cr)
	if username == "" && cr.Spec.ForProvider.Username != nil {
		username = *cr.Spec.ForProvider.Username
	}

	if username == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	user, err := e.client.GetUser(ctx, username)
	if err != nil {
		if IsUserNotFound(err) {
			return managed.ExternalObservation{ResourceExists: false}, nil
		}
		return managed.ExternalObservation{}, errors.Wrap(err, errGetUser)
	}

	// Update status
	cr.Status.AtProvider.ID = ptrString(strconv.FormatInt(user.UserID, 10))
	cr.Status.AtProvider.Username = &user.Username
	cr.Status.AtProvider.Email = &user.Email
	cr.Status.AtProvider.FullName = &user.Realname
	cr.Status.AtProvider.Admin = &user.SysadminFlag
	cr.Status.AtProvider.Comment = &user.Comment

	// Set external name if not set
	if meta.GetExternalName(cr) == "" {
		meta.SetExternalName(cr, user.Username)
	}

	// Check if update is needed
	isUpToDate := true
	if cr.Spec.ForProvider.Email != nil && *cr.Spec.ForProvider.Email != user.Email {
		isUpToDate = false
	}
	if cr.Spec.ForProvider.FullName != nil && *cr.Spec.ForProvider.FullName != user.Realname {
		isUpToDate = false
	}
	if cr.Spec.ForProvider.Admin != nil && *cr.Spec.ForProvider.Admin != user.SysadminFlag {
		isUpToDate = false
	}
	if cr.Spec.ForProvider.Comment != nil && *cr.Spec.ForProvider.Comment != user.Comment {
		isUpToDate = false
	}

	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: isUpToDate,
	}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUser)
	}

	// Get parameters
	username := ptrValue(cr.Spec.ForProvider.Username)
	if username == "" && meta.GetExternalName(cr) != "" {
		username = meta.GetExternalName(cr)
	}

	email := ptrValue(cr.Spec.ForProvider.Email)
	fullName := ptrValue(cr.Spec.ForProvider.FullName)
	admin := ptrValueBool(cr.Spec.ForProvider.Admin)
	comment := ptrValue(cr.Spec.ForProvider.Comment)

	// Get password from secret
	password, err := e.getPassword(ctx, cr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGetPassword)
	}

	userID, err := e.client.CreateUser(ctx, username, email, fullName, password, admin, comment)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	// Set external name
	meta.SetExternalName(cr, username)
	cr.Status.AtProvider.ID = ptrString(strconv.FormatInt(userID, 10))

	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotUser)
	}

	userID, err := strconv.ParseInt(ptrValue(cr.Status.AtProvider.ID), 10, 64)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to parse user ID")
	}

	// Update user profile
	email := ptrValue(cr.Spec.ForProvider.Email)
	fullName := ptrValue(cr.Spec.ForProvider.FullName)
	admin := ptrValueBool(cr.Spec.ForProvider.Admin)
	comment := ptrValue(cr.Spec.ForProvider.Comment)

	if err := e.client.UpdateUser(ctx, userID, email, fullName, admin, comment); err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, errUpdateUser)
	}

	// Update password if provided
	if cr.Spec.ForProvider.PasswordSecretRef.Name != "" {
		password, err := e.getPassword(ctx, cr)
		if err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, errGetPassword)
		}
		if password != "" {
			if err := e.client.UpdateUserPassword(ctx, userID, password); err != nil {
				return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update password")
			}
		}
	}

	return managed.ExternalUpdate{}, nil
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) error {
	cr, ok := mg.(*v1alpha1.User)
	if !ok {
		return errors.New(errNotUser)
	}

	userID, err := strconv.ParseInt(ptrValue(cr.Status.AtProvider.ID), 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse user ID")
	}

	if err := e.client.DeleteUser(ctx, userID); err != nil {
		if IsUserNotFound(err) {
			// Already deleted
			return nil
		}
		return errors.Wrap(err, errDeleteUser)
	}

	return nil
}

// getPassword retrieves the password from the secret reference
func (e *external) getPassword(ctx context.Context, cr *v1alpha1.User) (string, error) {
	if cr.Spec.ForProvider.PasswordSecretRef.Name == "" {
		return "", nil
	}

	nn := types.NamespacedName{
		Name:      cr.Spec.ForProvider.PasswordSecretRef.Name,
		Namespace: cr.Spec.ForProvider.PasswordSecretRef.Namespace,
	}

	secret := &v1.Secret{}
	if err := e.kube.Get(ctx, nn, secret); err != nil {
		return "", err
	}

	password, ok := secret.Data[cr.Spec.ForProvider.PasswordSecretRef.Key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret", cr.Spec.ForProvider.PasswordSecretRef.Key)
	}

	return string(password), nil
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrValueBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// IsUserNotFound returns true if the error indicates the user was not found
func IsUserNotFound(err error) bool {
	return err != nil && (err.Error() == "user not found" || contains(err.Error(), "not found"))
}

func contains(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}