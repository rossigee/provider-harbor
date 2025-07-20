package native

import (
	"context"
	"testing"

	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	"github.com/globallogicuki/provider-harbor/apis/user/v1alpha1"
)

// MockClient is a mock implementation of the Client interface
type MockClient struct {
	MockGetUser            func(ctx context.Context, username string) (*models.UserResp, error)
	MockCreateUser         func(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error)
	MockUpdateUser         func(ctx context.Context, userID int64, email, realname string, admin bool, comment string) error
	MockUpdateUserPassword func(ctx context.Context, userID int64, newPassword string) error
	MockDeleteUser         func(ctx context.Context, userID int64) error
}

func (m *MockClient) GetUser(ctx context.Context, username string) (*models.UserResp, error) {
	return m.MockGetUser(ctx, username)
}

func (m *MockClient) CreateUser(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error) {
	return m.MockCreateUser(ctx, username, email, realname, password, admin, comment)
}

func (m *MockClient) UpdateUser(ctx context.Context, userID int64, email, realname string, admin bool, comment string) error {
	return m.MockUpdateUser(ctx, userID, email, realname, admin, comment)
}

func (m *MockClient) UpdateUserPassword(ctx context.Context, userID int64, newPassword string) error {
	return m.MockUpdateUserPassword(ctx, userID, newPassword)
}

func (m *MockClient) DeleteUser(ctx context.Context, userID int64) error {
	return m.MockDeleteUser(ctx, userID)
}

func TestObserve(t *testing.T) {
	type args struct {
		mg     resource.Managed
		client Client
		kube   client.Client
	}
	type want struct {
		o   managed.ExternalObservation
		err error
		mg  resource.Managed
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"UserNotFound": {
			args: args{
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
						},
					},
				},
				client: &MockClient{
					MockGetUser: func(ctx context.Context, username string) (*models.UserResp, error) {
						return nil, errors.New("user not found")
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
			},
		},
		"UserExists": {
			args: args{
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
							Email:    ptrString("test@example.com"),
							FullName: ptrString("Test User"),
							Admin:    ptrBool(false),
							Comment:  ptrString("Test comment"),
						},
					},
				},
				client: &MockClient{
					MockGetUser: func(ctx context.Context, username string) (*models.UserResp, error) {
						return &models.UserResp{
							UserID:       123,
							Username:     "testuser",
							Email:        "test@example.com",
							Realname:     "Test User",
							SysadminFlag: false,
							Comment:      "Test comment",
						}, nil
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: true,
				},
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
						Annotations: map[string]string{
							meta.AnnotationKeyExternalName: "testuser",
						},
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
							Email:    ptrString("test@example.com"),
							FullName: ptrString("Test User"),
							Admin:    ptrBool(false),
							Comment:  ptrString("Test comment"),
						},
					},
					Status: v1alpha1.UserStatus{
						AtProvider: v1alpha1.UserObservation{
							ID:       ptrString("123"),
							Username: ptrString("testuser"),
							Email:    ptrString("test@example.com"),
							FullName: ptrString("Test User"),
							Admin:    ptrBool(false),
							Comment:  ptrString("Test comment"),
						},
					},
				},
			},
		},
		"UserNeedsUpdate": {
			args: args{
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
							Email:    ptrString("newemail@example.com"),
							FullName: ptrString("Updated User"),
							Admin:    ptrBool(true),
							Comment:  ptrString("Updated comment"),
						},
					},
				},
				client: &MockClient{
					MockGetUser: func(ctx context.Context, username string) (*models.UserResp, error) {
						return &models.UserResp{
							UserID:       123,
							Username:     "testuser",
							Email:        "test@example.com",
							Realname:     "Test User",
							SysadminFlag: false,
							Comment:      "Test comment",
						}, nil
					},
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:   true,
					ResourceUpToDate: false,
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.args.client,
				kube:   tc.args.kube,
			}
			o, err := e.Observe(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("e.Observe(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("e.Observe(...): -want, +got:\n%s", diff)
			}
			if tc.want.mg != nil {
				if diff := cmp.Diff(tc.want.mg, tc.args.mg); diff != "" {
					t.Errorf("e.Observe(...): -want mg, +got mg:\n%s", diff)
				}
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type args struct {
		mg     resource.Managed
		client Client
		kube   client.Client
	}
	type want struct {
		c   managed.ExternalCreation
		err error
	}

	scheme := runtime.NewScheme()
	_ = v1.AddToScheme(scheme)

	passwordSecret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-password",
			Namespace: "default",
		},
		Data: map[string][]byte{
			"password": []byte("secretpassword"),
		},
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"CreateUser": {
			args: args{
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
							Email:    ptrString("test@example.com"),
							FullName: ptrString("Test User"),
							Admin:    ptrBool(false),
							Comment:  ptrString("Test comment"),
							PasswordSecretRef: v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "user-password",
								},
								Namespace: ptrString("default"),
								Key:       "password",
							},
						},
					},
				},
				client: &MockClient{
					MockCreateUser: func(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error) {
						if username != "testuser" || email != "test@example.com" || realname != "Test User" || password != "secretpassword" {
							return 0, errors.New("unexpected parameters")
						}
						return 123, nil
					},
				},
				kube: fake.NewClientBuilder().WithScheme(scheme).WithObjects(passwordSecret).Build(),
			},
			want: want{
				c: managed.ExternalCreation{},
			},
		},
		"CreateUserError": {
			args: args{
				mg: &v1alpha1.User{
					ObjectMeta: metav1.ObjectMeta{
						Name: "test-user",
					},
					Spec: v1alpha1.UserSpec{
						ForProvider: v1alpha1.UserParameters{
							Username: ptrString("testuser"),
							Email:    ptrString("test@example.com"),
							PasswordSecretRef: v1.SecretKeySelector{
								LocalObjectReference: v1.LocalObjectReference{
									Name: "user-password",
								},
								Namespace: ptrString("default"),
								Key:       "password",
							},
						},
					},
				},
				client: &MockClient{
					MockCreateUser: func(ctx context.Context, username, email, realname, password string, admin bool, comment string) (int64, error) {
						return 0, errors.New("failed to create user")
					},
				},
				kube: fake.NewClientBuilder().WithScheme(scheme).WithObjects(passwordSecret).Build(),
			},
			want: want{
				err: errors.Wrap(errors.New("failed to create user"), errCreateUser),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.args.client,
				kube:   tc.args.kube,
			}
			c, err := e.Create(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("e.Create(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.c, c); diff != "" {
				t.Errorf("e.Create(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	type args struct {
		mg     resource.Managed
		client Client
	}
	type want struct {
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"DeleteUser": {
			args: args{
				mg: &v1alpha1.User{
					Status: v1alpha1.UserStatus{
						AtProvider: v1alpha1.UserObservation{
							ID: ptrString("123"),
						},
					},
				},
				client: &MockClient{
					MockDeleteUser: func(ctx context.Context, userID int64) error {
						if userID != 123 {
							return errors.New("unexpected user ID")
						}
						return nil
					},
				},
			},
			want: want{},
		},
		"DeleteUserNotFound": {
			args: args{
				mg: &v1alpha1.User{
					Status: v1alpha1.UserStatus{
						AtProvider: v1alpha1.UserObservation{
							ID: ptrString("123"),
						},
					},
				},
				client: &MockClient{
					MockDeleteUser: func(ctx context.Context, userID int64) error {
						return errors.New("user not found")
					},
				},
			},
			want: want{},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.args.client,
			}
			err := e.Delete(context.Background(), tc.args.mg)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("e.Delete(...): -want error, +got error:\n%s", diff)
			}
		})
	}
}

func ptrBool(b bool) *bool {
	return &b
}