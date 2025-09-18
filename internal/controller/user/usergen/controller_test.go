/*
Copyright 2022 Upbound Inc.
*/

package usergen

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/crossplane-runtime/pkg/test"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/user/v1alpha1"
)

func TestObserve(t *testing.T) {
	cases := map[string]struct {
		mg     resource.Managed
		o      managed.ExternalObservation
		err    error
		reason string
		client client.Client
	}{
		"NotUserWithGeneratedPassword": {
			mg:     &v1alpha1.User{}, // Wrong type
			err:    errors.New(errNotUserWithGeneratedPassword),
			reason: "Should return error for wrong resource type",
		},
		"NoPasswordGeneration": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						// No generatePasswordInSecret field
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					// Simulate underlying user doesn't exist
					return kerrors.NewNotFound(schema.GroupResource{}, "test-user-user")
				},
			},
			o: managed.ExternalObservation{
				ResourceExists: false,
			},
			reason: "Should check user when no password generation needed",
		},
		"SecretNotFound": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						GeneratePasswordInSecret: &v1alpha1.GeneratePasswordConfig{
							Name: "test-secret",
						},
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					// Secret doesn't exist
					return kerrors.NewNotFound(schema.GroupResource{}, "test-secret")
				},
			},
			o: managed.ExternalObservation{
				ResourceExists: false,
			},
			reason: "Should return false when password secret doesn't exist (Phase 1)",
		},
		"SecretExistsUserNotFound": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						GeneratePasswordInSecret: &v1alpha1.GeneratePasswordConfig{
							Name: "test-secret",
						},
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					switch obj.(type) {
					case *corev1.Secret:
						// Secret exists
						return nil
					case *v1alpha1.User:
						// User doesn't exist
						return kerrors.NewNotFound(schema.GroupResource{}, "test-user-user")
					}
					return nil
				},
			},
			o: managed.ExternalObservation{
				ResourceExists: false,
			},
			reason: "Should return false when secret exists but user doesn't (Phase 2)",
		},
		"BothExist": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						GeneratePasswordInSecret: &v1alpha1.GeneratePasswordConfig{
							Name: "test-secret",
						},
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					// Both secret and user exist
					return nil
				},
			},
			o: managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: managed.ConnectionDetails{},
			},
			reason: "Should return true when both secret and user exist",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{
				kube:   tc.client,
				logger: logging.NewNopLogger(),
			}
			got, err := e.Observe(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.o, got); diff != "" {
				t.Errorf("\n%s\nObserve(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	cases := map[string]struct {
		mg     resource.Managed
		o      managed.ExternalCreation
		err    error
		reason string
		client client.Client
	}{
		"NotUserWithGeneratedPassword": {
			mg:     &v1alpha1.User{}, // Wrong type
			err:    errors.New(errNotUserWithGeneratedPassword),
			reason: "Should return error for wrong resource type",
		},
		"CreateSecretPhase": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					UID:  "test-uid",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						UserParameters: v1alpha1.UserParameters{
							Username:          stringPtr("testuser"),
							Email:             stringPtr("test@example.com"),
							FullName:          stringPtr("Test User"),
							PasswordSecretRef: xpv1.SecretKeySelector{},
						},
						GeneratePasswordInSecret: &v1alpha1.GeneratePasswordConfig{
							Name:   "test-secret",
							Length: intPtr(16),
						},
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					// Secret doesn't exist, triggering Phase 1
					return kerrors.NewNotFound(schema.GroupResource{}, "test-secret")
				},
				MockCreate: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
					// Verify secret creation
					secret, ok := obj.(*corev1.Secret)
					if !ok {
						return errors.New("expected Secret object")
					}
					if secret.Name != "test-secret" {
						return errors.Errorf("expected secret name 'test-secret', got %s", secret.Name)
					}
					if len(secret.Data["password"]) == 0 {
						return errors.New("password data is empty")
					}
					return nil
				},
			},
			o: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			reason: "Should create password secret in Phase 1",
		},
		"CreateUserPhase": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					UID:  "test-uid",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						UserParameters: v1alpha1.UserParameters{
							Username:          stringPtr("testuser"),
							Email:             stringPtr("test@example.com"),
							FullName:          stringPtr("Test User"),
							PasswordSecretRef: xpv1.SecretKeySelector{},
						},
						GeneratePasswordInSecret: &v1alpha1.GeneratePasswordConfig{
							Name: "test-secret",
						},
					},
				},
			},
			client: &test.MockClient{
				MockGet: func(ctx context.Context, key client.ObjectKey, obj client.Object) error {
					switch obj.(type) {
					case *corev1.Secret:
						// Secret exists, triggering Phase 2
						return nil
					default:
						return nil
					}
				},
				MockCreate: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
					// Verify user creation
					user, ok := obj.(*v1alpha1.User)
					if !ok {
						return errors.New("expected User object")
					}
					if user.Name != "test-user-user" {
						return errors.Errorf("expected user name 'test-user-user', got %s", user.Name)
					}
					if user.Spec.ForProvider.PasswordSecretRef.Name != "test-secret" {
						return errors.New("password secret reference not set correctly")
					}
					return nil
				},
			},
			o: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			reason: "Should create Harbor user in Phase 2",
		},
		"CreateUserPhaseWithExistingSecret": {
			mg: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					UID:  "test-uid",
				},
				Spec: v1alpha1.UserWithGeneratedPasswordSpec{
					ForProvider: v1alpha1.UserWithGeneratedPasswordParameters{
						UserParameters: v1alpha1.UserParameters{
							Username: stringPtr("testuser"),
							Email:    stringPtr("test@example.com"),
							FullName: stringPtr("Test User"),
							PasswordSecretRef: xpv1.SecretKeySelector{
								SecretReference: xpv1.SecretReference{
									Name:      "existing-secret",
									Namespace: "default",
								},
								Key: "password",
							},
						},
						// No generatePasswordInSecret - using existing secret
					},
				},
			},
			client: &test.MockClient{
				MockCreate: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
					// Should create user directly
					_, ok := obj.(*v1alpha1.User)
					if !ok {
						return errors.New("expected User object")
					}
					return nil
				},
			},
			o: managed.ExternalCreation{
				ConnectionDetails: managed.ConnectionDetails{},
			},
			reason: "Should create user directly when not generating password",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{
				kube:   tc.client,
				logger: logging.NewNopLogger(),
			}
			got, err := e.Create(context.Background(), tc.mg)

			if diff := cmp.Diff(tc.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\nCreate(...): -want error, +got error:\n%s", tc.reason, diff)
			}
			if diff := cmp.Diff(tc.o, got); diff != "" {
				t.Errorf("\n%s\nCreate(...): -want, +got:\n%s", tc.reason, diff)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	e := external{
		logger: logging.NewNopLogger(),
	}

	cases := map[string]struct {
		length int
		want   struct {
			minLength int
			maxLength int
			err       bool
		}
		reason string
	}{
		"DefaultLength": {
			length: defaultPasswordLength,
			want: struct {
				minLength int
				maxLength int
				err       bool
			}{
				minLength: defaultPasswordLength,
				maxLength: defaultPasswordLength,
				err:       false,
			},
			reason: "Should generate password of default length",
		},
		"CustomLength": {
			length: 24,
			want: struct {
				minLength int
				maxLength int
				err       bool
			}{
				minLength: 24,
				maxLength: 24,
				err:       false,
			},
			reason: "Should generate password of custom length",
		},
		"MinLength": {
			length: 6, // Below minimum, should be adjusted to default
			want: struct {
				minLength int
				maxLength int
				err       bool
			}{
				minLength: defaultPasswordLength,
				maxLength: defaultPasswordLength,
				err:       false,
			},
			reason: "Should use default length when requested length is too small",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got, err := e.generateSecurePassword(tc.length)

			if tc.want.err && err == nil {
				t.Errorf("\n%s\ngenerateSecurePassword(%d): expected error, got none", tc.reason, tc.length)
			}
			if !tc.want.err && err != nil {
				t.Errorf("\n%s\ngenerateSecurePassword(%d): expected no error, got %v", tc.reason, tc.length, err)
			}
			if !tc.want.err {
				if len(got) < tc.want.minLength || len(got) > tc.want.maxLength {
					t.Errorf("\n%s\ngenerateSecurePassword(%d): expected length between %d and %d, got %d",
						tc.reason, tc.length, tc.want.minLength, tc.want.maxLength, len(got))
				}
				// Check that password contains expected character types
				if got == "" {
					t.Errorf("\n%s\ngenerateSecurePassword(%d): generated empty password", tc.reason, tc.length)
				}
				// Test password uniqueness by generating multiple passwords
				got2, err2 := e.generateSecurePassword(tc.length)
				if err2 != nil {
					t.Errorf("\n%s\ngenerateSecurePassword(%d): second generation failed: %v", tc.reason, tc.length, err2)
				}
				if got == got2 {
					t.Errorf("\n%s\ngenerateSecurePassword(%d): generated identical passwords (should be random): %s",
						tc.reason, tc.length, got)
				}
			}
		})
	}
}

func TestCreatePasswordSecret(t *testing.T) {
	cases := map[string]struct {
		secretName      string
		secretNamespace string
		secretKey       string
		password        string
		user            *v1alpha1.UserWithGeneratedPassword
		want            error
		mockCreate      func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error
		reason          string
	}{
		"Success": {
			secretName:      "test-secret",
			secretNamespace: "test-namespace",
			secretKey:       "password",
			password:        "test-password",
			user: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
					UID:  "test-uid",
				},
				TypeMeta: metav1.TypeMeta{
					APIVersion: "user.harbor.crossplane.io/v1alpha1",
					Kind:       "UserWithGeneratedPassword",
				},
			},
			mockCreate: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
				secret, ok := obj.(*corev1.Secret)
				if !ok {
					return errors.New("expected Secret object")
				}
				if secret.Name != "test-secret" {
					return errors.Errorf("expected name 'test-secret', got %s", secret.Name)
				}
				if secret.Namespace != "test-namespace" {
					return errors.Errorf("expected namespace 'test-namespace', got %s", secret.Namespace)
				}
				if string(secret.Data["password"]) != "test-password" {
					return errors.New("password data incorrect")
				}
				if len(secret.OwnerReferences) != 1 {
					return errors.Errorf("expected 1 owner reference, got %d", len(secret.OwnerReferences))
				}
				if secret.OwnerReferences[0].Name != "test-user" {
					return errors.New("owner reference name incorrect")
				}
				return nil
			},
			reason: "Should create secret with correct properties",
		},
		"CreateFails": {
			secretName:      "test-secret",
			secretNamespace: "test-namespace",
			secretKey:       "password",
			password:        "test-password",
			user: &v1alpha1.UserWithGeneratedPassword{
				ObjectMeta: metav1.ObjectMeta{Name: "test-user"},
			},
			want: errors.New("create failed"),
			mockCreate: func(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
				return errors.New("create failed")
			},
			reason: "Should handle secret creation failure",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := external{
				kube: &test.MockClient{
					MockCreate: tc.mockCreate,
				},
				logger: logging.NewNopLogger(),
			}

			got := e.createPasswordSecret(context.Background(), tc.secretName, tc.secretNamespace, tc.secretKey, tc.password, tc.user)

			if diff := cmp.Diff(tc.want, got, test.EquateErrors()); diff != "" {
				t.Errorf("\n%s\ncreatePasswordSecret(...): -want error, +got error:\n%s", tc.reason, diff)
			}
		})
	}
}

// Helper functions for test cases
func stringPtr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}
