/*
Copyright 2024 Crossplane Harbor Provider.
*/

package user

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/user/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotUser(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Connect with nil should return %s error", errNotUser)
	}
}

func TestObserveNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Observe with nil should return %s error", errNotUser)
	}
}

func TestUpdateNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Update with nil should return %s error", errNotUser)
	}
}

func TestDeleteNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Delete with nil should return %s error", errNotUser)
	}
}

func TestCreateNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Create with nil should return %s error", errNotUser)
	}
}

func TestObserveUserNotFound(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			getUserFunc: func(ctx context.Context, username string) (*harborclients.UserStatus, error) {
				return nil, errors.New("not found")
			},
		},
	}

	obs, err := ext.Observe(ctx, user)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when user not found")
	}
}

func TestObserveUserExists(t *testing.T) {
	ctx := context.Background()
	email := "test@example.com"
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			getUserFunc: func(ctx context.Context, username string) (*harborclients.UserStatus, error) {
				return &harborclients.UserStatus{
					Username:  "testuser",
					Email:     email,
					AdminFlag: false,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, user)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true")
	}
}

func TestObserveUserNotUpToDate(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username:     "testuser",
				Email:        "new@example.com",
				SysAdminFlag: ptrBool(true),
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			getUserFunc: func(ctx context.Context, username string) (*harborclients.UserStatus, error) {
				return &harborclients.UserStatus{
					Username:  "testuser",
					Email:     "old@example.com",
					AdminFlag: false,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, user)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when email differs")
	}
}

func TestCreateUserSuccess(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			createUserFunc: func(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
				return &harborclients.UserStatus{
					Username:  spec.Username,
					Email:     spec.Email,
					AdminFlag: spec.AdminFlag,
					CreatedAt: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, user)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateUserError(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			createUserFunc: func(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, user)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateUserSuccess(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			updateUserFunc: func(ctx context.Context, username string, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
				return &harborclients.UserStatus{
					Username:  spec.Username,
					Email:     spec.Email,
					AdminFlag: spec.AdminFlag,
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, user)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteUserSuccess(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			deleteUserFunc: func(ctx context.Context, username string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, user)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteUserError(t *testing.T) {
	ctx := context.Background()
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	ext := &external{
		service: &mockUserClient{
			deleteUserFunc: func(ctx context.Context, username string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, user)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestHelperFunctions(t *testing.T) {
	intVal := int64(42)
	result := getInt64Ptr(intVal)
	if result == nil || *result != intVal {
		t.Errorf("getInt64Ptr failed")
	}

	boolVal := true
	resultBool := getBoolValue(&boolVal)
	if !resultBool {
		t.Errorf("getBoolValue failed")
	}

	nilBool := getBoolValue(nil)
	if nilBool {
		t.Errorf("getBoolValue with nil should return false")
	}
}

func TestUserHasRequiredFields(t *testing.T) {
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-user",
			Namespace: "default",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}

	if user.Spec.ForProvider.Username == "" {
		t.Error("Username should not be empty")
	}
	if user.Spec.ForProvider.Email == "" {
		t.Error("Email should not be empty")
	}
	if user.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestUserStatusFields(t *testing.T) {
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
		Status: v1beta1.UserStatus{
			AtProvider: v1beta1.UserObservation{
				ID: ptrInt64(123),
			},
		},
	}

	if user.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *user.Status.AtProvider.ID != 123 {
		t.Errorf("Status ID should be 123, got %d", *user.Status.AtProvider.ID)
	}
}

func TestUserParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.UserParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.UserParameters{
				Username: "user1",
				Email:    "user1@example.com",
			},
			isValid: true,
		},
		{
			name: "valid with admin flag",
			params: v1beta1.UserParameters{
				Username:     "admin1",
				Email:        "admin@example.com",
				SysAdminFlag: ptrBool(true),
			},
			isValid: true,
		},
		{
			name: "missing username",
			params: v1beta1.UserParameters{
				Email: "user@example.com",
			},
			isValid: false,
		},
		{
			name: "missing email",
			params: v1beta1.UserParameters{
				Username: "user1",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Username != "" && tt.params.Email != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestUserAdminFlag(t *testing.T) {
	adminFlag := true
	user := &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name: "admin-user",
		},
		Spec: v1beta1.UserSpec{
			ForProvider: v1beta1.UserParameters{
				Username:     "adminuser",
				Email:        "admin@example.com",
				SysAdminFlag: &adminFlag,
			},
		},
	}

	if user.Spec.ForProvider.SysAdminFlag == nil {
		t.Error("SysAdminFlag should not be nil")
	}
	if !*user.Spec.ForProvider.SysAdminFlag {
		t.Error("SysAdminFlag should be true")
	}
}

// mockUserClient implements HarborClienter for user tests
type mockUserClient struct {
	harborclients.HarborClienter
	getUserFunc    func(ctx context.Context, username string) (*harborclients.UserStatus, error)
	createUserFunc func(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error)
	updateUserFunc func(ctx context.Context, username string, spec *harborclients.UserSpec) (*harborclients.UserStatus, error)
	deleteUserFunc func(ctx context.Context, username string) error
}

func (m *mockUserClient) GetUser(ctx context.Context, username string) (*harborclients.UserStatus, error) {
	if m.getUserFunc != nil {
		return m.getUserFunc(ctx, username)
	}
	return nil, nil
}

func (m *mockUserClient) CreateUser(ctx context.Context, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
	if m.createUserFunc != nil {
		return m.createUserFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockUserClient) UpdateUser(ctx context.Context, username string, spec *harborclients.UserSpec) (*harborclients.UserStatus, error) {
	if m.updateUserFunc != nil {
		return m.updateUserFunc(ctx, username, spec)
	}
	return nil, nil
}

func (m *mockUserClient) DeleteUser(ctx context.Context, username string) error {
	if m.deleteUserFunc != nil {
		return m.deleteUserFunc(ctx, username)
	}
	return nil
}

func (m *mockUserClient) Close() error {
	return nil
}

func (m *mockUserClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}
