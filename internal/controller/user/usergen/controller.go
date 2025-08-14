/*
Copyright 2022 Upbound Inc.
*/

package usergen

import (
	"context"
	"crypto/rand"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/apis/common/v1"
	tjcontroller "github.com/crossplane/upjet/pkg/controller"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/user/v1alpha1"
)

const (
	errNotUserWithGeneratedPassword = "managed resource is not a UserWithGeneratedPassword custom resource"
	errGeneratePassword            = "cannot generate secure password"
	errCreateSecret               = "cannot create password secret"
	errCreateUser                 = "cannot create underlying user resource"
	errTrackUsage                 = "cannot track provider config usage"

	defaultPasswordLength = 16
	defaultSecretKey     = "password"
)

// Setup adds a controller that reconciles UserWithGeneratedPassword managed resources.
func Setup(mgr ctrl.Manager, o tjcontroller.Options) error {
	name := managed.ControllerName("userwithgeneratedpassword")
	
	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.UserWithGeneratedPassword_GroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:   mgr.GetClient(),
			logger: o.Logger.WithValues("controller", name),
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.UserWithGeneratedPassword{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube   client.Client
	logger logging.Logger
}

// Connect produces an ExternalClient by reading the managed resource's provider config.
func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	user, ok := mg.(*v1alpha1.UserWithGeneratedPassword)
	if !ok {
		return nil, errors.New(errNotUserWithGeneratedPassword)
	}

	return &external{
		kube:   c.kube,
		logger: c.logger,
		user:   user,
	}, nil
}

// An ExternalClient observes, then either creates, updates, or deletes an
// external resource to ensure it reflects the managed resource's desired state.
type external struct {
	kube   client.Client
	logger logging.Logger
	user   *v1alpha1.UserWithGeneratedPassword
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	user, ok := mg.(*v1alpha1.UserWithGeneratedPassword)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotUserWithGeneratedPassword)
	}

	// Multi-phase approach:
	// Phase 1: Create password secret if needed
	// Phase 2: Create Harbor user if secret exists
	
	needsPasswordGeneration := user.Spec.ForProvider.GeneratePasswordInSecret != nil
	
	if needsPasswordGeneration {
		// Check if the secret already exists
		secretName := user.Spec.ForProvider.GeneratePasswordInSecret.Name
		secretNamespace := user.Spec.ForProvider.GeneratePasswordInSecret.Namespace
		if secretNamespace == "" {
			secretNamespace = user.GetNamespace()
			if secretNamespace == "" {
				secretNamespace = "default"
			}
		}

		secret := &corev1.Secret{}
		err := c.kube.Get(ctx, types.NamespacedName{
			Name:      secretName,
			Namespace: secretNamespace,
		}, secret)

		if err != nil && !kerrors.IsNotFound(err) {
			return managed.ExternalObservation{}, errors.Wrap(err, "cannot check if password secret exists")
		}

		if kerrors.IsNotFound(err) {
			// Phase 1: Secret doesn't exist, we're in the secret creation phase
			c.logger.Info("Password secret not found, in secret creation phase", "secret", secretName, "namespace", secretNamespace)
			return managed.ExternalObservation{
				ResourceExists: false,
			}, nil
		}

		c.logger.Info("Password secret exists, checking user", "secret", secretName, "namespace", secretNamespace)
	}

	// Phase 2: Check if the underlying user resource exists
	underlyingUser := &v1alpha1.User{}
	err := c.kube.Get(ctx, types.NamespacedName{
		Name: user.GetName() + "-user",
	}, underlyingUser)

	if err != nil && !kerrors.IsNotFound(err) {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot check if underlying user exists")
	}

	if kerrors.IsNotFound(err) {
		// User doesn't exist but secret does (or isn't needed) - ready to create user
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	// Both secret (if needed) and user exist - we're done
	return managed.ExternalObservation{
		ResourceExists:   true,
		ResourceUpToDate: true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	user, ok := mg.(*v1alpha1.UserWithGeneratedPassword)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotUserWithGeneratedPassword)
	}

	c.logger.Info("Creating UserWithGeneratedPassword", "name", user.GetName())

	// Determine which phase we're in based on what exists
	needsPasswordGeneration := user.Spec.ForProvider.GeneratePasswordInSecret != nil
	
	if needsPasswordGeneration {
		// Check if secret exists yet
		genConfig := user.Spec.ForProvider.GeneratePasswordInSecret
		secretNamespace := genConfig.Namespace
		if secretNamespace == "" {
			secretNamespace = user.GetNamespace()
			if secretNamespace == "" {
				secretNamespace = "default"
			}
		}

		secret := &corev1.Secret{}
		err := c.kube.Get(ctx, types.NamespacedName{
			Name:      genConfig.Name,
			Namespace: secretNamespace,
		}, secret)

		if kerrors.IsNotFound(err) {
			// Phase 1: Create the password secret first
			return c.createPasswordSecretPhase(ctx, user, genConfig, secretNamespace)
		} else if err != nil {
			return managed.ExternalCreation{}, errors.Wrap(err, "cannot check if password secret exists")
		}

		// Secret exists, proceed to Phase 2
		c.logger.Info("Password secret exists, proceeding to user creation", "secret", genConfig.Name)
	}

	// Phase 2: Create the Harbor User resource
	return c.createHarborUserPhase(ctx, user)
}

func (c *external) createPasswordSecretPhase(ctx context.Context, user *v1alpha1.UserWithGeneratedPassword, genConfig *v1alpha1.GeneratePasswordConfig, secretNamespace string) (managed.ExternalCreation, error) {
	c.logger.Info("Phase 1: Creating password secret", "secret", genConfig.Name, "namespace", secretNamespace)
	
	// Generate secure password
	length := defaultPasswordLength
	if genConfig.Length != nil {
		length = *genConfig.Length
	}
	
	password, err := c.generateSecurePassword(length)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errGeneratePassword)
	}

	secretKey := genConfig.Key
	if secretKey == "" {
		secretKey = defaultSecretKey
	}

	if err := c.createPasswordSecret(ctx, genConfig.Name, secretNamespace, secretKey, password, user); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateSecret)
	}

	c.logger.Info("Created password secret, Harbor user will be created on next reconcile", "secret", genConfig.Name)
	
	// Return success but indicate more work is needed (Harbor user creation)
	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) createHarborUserPhase(ctx context.Context, user *v1alpha1.UserWithGeneratedPassword) (managed.ExternalCreation, error) {
	c.logger.Info("Phase 2: Creating Harbor User resource", "name", user.GetName())

	// Prepare password secret reference  
	var passwordSecretRef *v1.SecretKeySelector
	
	if user.Spec.ForProvider.GeneratePasswordInSecret != nil {
		genConfig := user.Spec.ForProvider.GeneratePasswordInSecret
		secretNamespace := genConfig.Namespace
		if secretNamespace == "" {
			secretNamespace = user.GetNamespace()
			if secretNamespace == "" {
				secretNamespace = "default"
			}
		}
		
		secretKey := genConfig.Key
		if secretKey == "" {
			secretKey = defaultSecretKey
		}

		passwordSecretRef = &v1.SecretKeySelector{
			SecretReference: v1.SecretReference{
				Name:      genConfig.Name,
				Namespace: secretNamespace,
			},
			Key: secretKey,
		}
	} else {
		// Use the provided password secret ref
		passwordSecretRef = &user.Spec.ForProvider.PasswordSecretRef
	}

	// Create the underlying User resource
	underlyingUser := &v1alpha1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: user.GetName() + "-user",
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: user.APIVersion,
					Kind:       user.Kind,
					Name:       user.GetName(),
					UID:        user.GetUID(),
					Controller: func() *bool { b := true; return &b }(),
				},
			},
		},
		Spec: v1alpha1.UserSpec{
			ResourceSpec: user.Spec.ResourceSpec,
			ForProvider: v1alpha1.UserParameters{
				Admin:             user.Spec.ForProvider.Admin,
				Comment:           user.Spec.ForProvider.Comment,
				Email:             user.Spec.ForProvider.Email,
				FullName:          user.Spec.ForProvider.FullName,
				Username:          user.Spec.ForProvider.Username,
				PasswordSecretRef: *passwordSecretRef,
			},
			InitProvider: v1alpha1.UserInitParameters{
				Admin:    user.Spec.InitProvider.Admin,
				Comment:  user.Spec.InitProvider.Comment,
				Email:    user.Spec.InitProvider.Email,
				FullName: user.Spec.InitProvider.FullName,
				Username: user.Spec.InitProvider.Username,
			},
		},
	}

	if err := c.kube.Create(ctx, underlyingUser); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, errCreateUser)
	}

	c.logger.Info("Successfully created Harbor User resource", "user", underlyingUser.GetName())

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	// Updates are handled by the underlying User resource
	return managed.ExternalUpdate{}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	// Deletion is handled by owner references
	c.logger.Info("Deleting UserWithGeneratedPassword - underlying resources will be cleaned up by owner references")
	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	// No connection to disconnect
	return nil
}

// generateSecurePassword creates a cryptographically secure random password
func (c *external) generateSecurePassword(length int) (string, error) {
	if length < 8 {
		length = defaultPasswordLength
	}
	
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	bytes := make([]byte, length)
	
	if _, err := rand.Read(bytes); err != nil {
		return "", errors.Wrap(err, "failed to generate random bytes")
	}
	
	for i, b := range bytes {
		bytes[i] = charset[b%byte(len(charset))]
	}
	
	return string(bytes), nil
}

// createPasswordSecret creates a Kubernetes secret with the generated password
func (c *external) createPasswordSecret(ctx context.Context, secretName, secretNamespace, secretKey, password string, user *v1alpha1.UserWithGeneratedPassword) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: user.APIVersion,
					Kind:       user.Kind,
					Name:       user.GetName(),
					UID:        user.GetUID(),
					Controller: func() *bool { b := true; return &b }(),
				},
			},
			Labels: map[string]string{
				"app.kubernetes.io/managed-by":     "provider-harbor",
				"harbor.crossplane.io/user":        user.GetName(),
				"harbor.crossplane.io/generated":   "true",
				"harbor.crossplane.io/secret-type": "password",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			secretKey: []byte(password),
		},
	}
	
	return c.kube.Create(ctx, secret)
}