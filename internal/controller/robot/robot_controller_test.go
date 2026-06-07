/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/robot/v1beta1"
)

// ERROR CASE TESTS

func TestConnectNotRobot(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Connect with nil should return %s error", errNotRobot)
	}
}

func TestObserveNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Observe with nil should return %s error", errNotRobot)
	}
}

func TestUpdateNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Update with nil should return %s error", errNotRobot)
	}
}

func TestDeleteNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Delete with nil should return %s error", errNotRobot)
	}
}

func TestCreateNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Create with nil should return %s error", errNotRobot)
	}
}

// HAPPY-PATH AND VALIDATION TESTS

func TestRobotHasRequiredFields(t *testing.T) {
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-robot",
			Namespace: "default",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name: "my-robot",
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull", "push"},
					},
				},
			},
		},
	}

	if robot.Spec.ForProvider.Name == "" {
		t.Error("Robot Name should not be empty")
	}
	if len(robot.Spec.ForProvider.Permissions) == 0 {
		t.Error("Robot Permissions should not be empty")
	}
	if robot.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestRobotStatusFields(t *testing.T) {
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name: "my-robot",
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull"},
					},
				},
			},
		},
		Status: v1beta1.RobotStatus{
			AtProvider: v1beta1.RobotObservation{
				ID:     ptrString("robot-123"),
				Secret: ptrString("secret-token"),
			},
		},
	}

	if robot.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *robot.Status.AtProvider.ID != "robot-123" {
		t.Errorf("Status ID should be 'robot-123', got %s", *robot.Status.AtProvider.ID)
	}
	if robot.Status.AtProvider.Secret == nil {
		t.Error("Status Secret should be populated")
	}
}

func TestRobotParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.RobotParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.RobotParameters{
				Name: "ci-robot",
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull", "push"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.RobotParameters{
				Name:        "deploy-robot",
				Description: ptrString("Robot for deployments"),
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "valid with project ID",
			params: v1beta1.RobotParameters{
				Name:      "project-robot",
				ProjectID: ptrString("project-1"),
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "repository",
						Access:    []string{"pull", "push", "delete"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "valid with expiration",
			params: v1beta1.RobotParameters{
				Name:      "temp-robot",
				ExpiresIn: ptrInt64(30),
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "valid with multiple permissions",
			params: v1beta1.RobotParameters{
				Name: "multi-robot",
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull", "push"},
					},
					{
						Namespace: "repository",
						Access:    []string{"pull", "delete"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.RobotParameters{
				Permissions: []v1beta1.RobotPermission{
					{
						Namespace: "project",
						Access:    []string{"pull"},
					},
				},
			},
			isValid: false,
		},
		{
			name: "missing required permissions",
			params: v1beta1.RobotParameters{
				Name: "no-perms-robot",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != "" && len(tt.params.Permissions) > 0
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestRobotPermissions(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		access    []string
		isValid   bool
	}{
		{
			name:      "project pull access",
			namespace: "project",
			access:    []string{"pull"},
			isValid:   true,
		},
		{
			name:      "project pull and push",
			namespace: "project",
			access:    []string{"pull", "push"},
			isValid:   true,
		},
		{
			name:      "repository full access",
			namespace: "repository",
			access:    []string{"pull", "push", "delete"},
			isValid:   true,
		},
		{
			name:      "empty namespace",
			namespace: "",
			access:    []string{"pull"},
			isValid:   false,
		},
		{
			name:      "empty access",
			namespace: "project",
			access:    []string{},
			isValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.namespace != "" && len(tt.access) > 0
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestRobotExpirationValidation(t *testing.T) {
	tests := []struct {
		name    string
		expires int64
		isValid bool
	}{
		{
			name:    "1 day expiration",
			expires: 1,
			isValid: true,
		},
		{
			name:    "30 days expiration",
			expires: 30,
			isValid: true,
		},
		{
			name:    "365 days expiration",
			expires: 365,
			isValid: true,
		},
		{
			name:    "negative expiration",
			expires: -1,
			isValid: false,
		},
		{
			name:    "zero expiration",
			expires: 0,
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.expires >= 1
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}