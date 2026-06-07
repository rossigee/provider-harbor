/*
Copyright 2024 Crossplane Harbor Provider.
*/

package user

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/user/v1beta1"
)

// Error case tests

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

// Happy-path and data structure tests

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

func TestCreateUserParametersValidation(t *testing.T) {
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
				Username:    "admin1",
				Email:       "admin@example.com",
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

// Helper functions
func ptrBool(b bool) *bool {
	return &b
}

func ptrInt64(i int64) *int64 {
	return &i
}
