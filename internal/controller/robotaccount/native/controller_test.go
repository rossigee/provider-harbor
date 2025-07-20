package native

import (
	"context"
	"testing"

	"github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"

	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/test"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/globallogicuki/provider-harbor/apis/robotaccount/v1alpha1"
)

// MockHarborClient is a mock implementation of Harbor client
type MockHarborClient struct {
	mock.Mock
}

func (m *MockHarborClient) CreateRobotAccount(spec RobotAccountSpec) (*models.Robot, error) {
	args := m.Called(spec)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Robot), args.Error(1)
}

func (m *MockHarborClient) GetRobotAccount(robotID int64) (*models.Robot, error) {
	args := m.Called(robotID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Robot), args.Error(1)
}

func (m *MockHarborClient) GetRobotAccountByName(name string) (*models.Robot, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Robot), args.Error(1)
}

func (m *MockHarborClient) DeleteRobotAccount(robotID int64) error {
	args := m.Called(robotID)
	return args.Error(0)
}

func TestObserve(t *testing.T) {
	type args struct {
		cr     *v1alpha1.RobotAccount
		client func() HarborClientInterface
	}
	type want struct {
		o   managed.ExternalObservation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"NotFound": {
			args: args{
				cr: &v1alpha1.RobotAccount{
					Spec: v1alpha1.RobotAccountSpec{
						ForProvider: v1alpha1.RobotAccountParameters{
							Name: strPtr("test-robot"),
						},
					},
				},
				client: func() HarborClientInterface {
					m := &MockHarborClient{}
					m.On("GetRobotAccountByName", "test-robot").Return(nil, errors.New("not found"))
					return m
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists: false,
				},
			},
		},
		"Found": {
			args: args{
				cr: &v1alpha1.RobotAccount{
					Spec: v1alpha1.RobotAccountSpec{
						ForProvider: v1alpha1.RobotAccountParameters{
							Name:  strPtr("test-robot"),
							Level: strPtr("project"),
						},
					},
				},
				client: func() HarborClientInterface {
					m := &MockHarborClient{}
					robot := &models.Robot{
						ID:    123,
						Name:  "robot$project+test-robot",
						Level: "project",
					}
					m.On("GetRobotAccountByName", "test-robot").Return(robot, nil)
					return m
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
		"FoundWithExternalName": {
			args: args{
				cr: func() *v1alpha1.RobotAccount {
					cr := &v1alpha1.RobotAccount{
						Spec: v1alpha1.RobotAccountSpec{
							ForProvider: v1alpha1.RobotAccountParameters{
								Name:  strPtr("test-robot"),
								Level: strPtr("project"),
							},
						},
					}
					meta.SetExternalName(cr, "/robots/123")
					return cr
				}(),
				client: func() HarborClientInterface {
					m := &MockHarborClient{}
					robot := &models.Robot{
						ID:    123,
						Name:  "robot$project+test-robot",
						Level: "project",
					}
					m.On("GetRobotAccount", int64(123)).Return(robot, nil)
					return m
				},
			},
			want: want{
				o: managed.ExternalObservation{
					ResourceExists:    true,
					ResourceUpToDate:  true,
					ConnectionDetails: managed.ConnectionDetails{},
				},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.args.client(),
			}
			o, err := e.Observe(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("e.Observe(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("e.Observe(...): -want, +got:\n%s", diff)
			}
		})
	}
}

func TestCreate(t *testing.T) {
	type args struct {
		cr     *v1alpha1.RobotAccount
		client func() HarborClientInterface
	}
	type want struct {
		o   managed.ExternalCreation
		err error
	}

	cases := map[string]struct {
		args args
		want want
	}{
		"Successful": {
			args: args{
				cr: &v1alpha1.RobotAccount{
					Spec: v1alpha1.RobotAccountSpec{
						ForProvider: v1alpha1.RobotAccountParameters{
							Name:        strPtr("test-robot"),
							Description: strPtr("Test robot account"),
							Level:       strPtr("project"),
							Permissions: []v1alpha1.PermissionsParameters{
								{
									Kind:      strPtr("project"),
									Namespace: strPtr("library"),
									Access: []v1alpha1.AccessParameters{
										{
											Resource: strPtr("repository"),
											Action:   strPtr("pull"),
										},
									},
								},
							},
						},
					},
				},
				client: func() HarborClientInterface {
					m := &MockHarborClient{}
					robot := &models.Robot{
						ID:     123,
						Name:   "robot$project+test-robot",
						Secret: "secret-password",
					}
					m.On("CreateRobotAccount", mock.Anything).Return(robot, nil)
					return m
				},
			},
			want: want{
				o: managed.ExternalCreation{
					ConnectionDetails: managed.ConnectionDetails{
						xpv1.ResourceCredentialsSecretPasswordKey: []byte("secret-password"),
						xpv1.ResourceCredentialsSecretUserKey:     []byte("robot$project+test-robot"),
					},
				},
			},
		},
		"Failed": {
			args: args{
				cr: &v1alpha1.RobotAccount{
					Spec: v1alpha1.RobotAccountSpec{
						ForProvider: v1alpha1.RobotAccountParameters{
							Name:  strPtr("test-robot"),
							Level: strPtr("project"),
							Permissions: []v1alpha1.PermissionsParameters{
								{
									Kind:      strPtr("project"),
									Namespace: strPtr("library"),
									Access: []v1alpha1.AccessParameters{
										{
											Resource: strPtr("repository"),
											Action:   strPtr("pull"),
										},
									},
								},
							},
						},
					},
				},
				client: func() HarborClientInterface {
					m := &MockHarborClient{}
					m.On("CreateRobotAccount", mock.Anything).Return(nil, errors.New("api error"))
					return m
				},
			},
			want: want{
				err: errors.Wrap(errors.New("api error"), "cannot create robot account"),
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := &external{
				client: tc.args.client(),
			}
			o, err := e.Create(context.Background(), tc.args.cr)

			if diff := cmp.Diff(tc.want.err, err, test.EquateErrors()); diff != "" {
				t.Errorf("e.Create(...): -want error, +got error:\n%s", diff)
			}
			if diff := cmp.Diff(tc.want.o, o); diff != "" {
				t.Errorf("e.Create(...): -want, +got:\n%s", diff)
			}
		})
	}
}

// Helper functions
func strPtr(s string) *string {
	return &s
}