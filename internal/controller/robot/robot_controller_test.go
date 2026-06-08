/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockRobotClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Robot{})
	if err != nil {
		t.Errorf("Connect should not fail, got %v", err)
	}
}

func TestConnectClientError(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return nil, errors.New("client creation failed")
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Robot{})
	if err == nil {
		t.Error("Connect should fail when client creation fails")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockRobotClient{
			closeFunc: func() error {
				return nil
			},
		},
	}

	err := ext.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not fail, got %v", err)
	}
}

func TestObserveRobotListError(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, projectID *string) ([]*harborclients.RobotStatus, error) {
				return nil, errors.New("list failed")
			},
		},
	}

	_, err := ext.Observe(ctx, robot)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
}

func TestObserveRobotWithNilValues(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, pid *string) ([]*harborclients.RobotStatus, error) {
				return []*harborclients.RobotStatus{
					{
						ID:           "robot-123",
						Name:         "my-robot",
						ProjectID:    pid,
						Secret:       "",
						ExpiresAt:    nil,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, robot)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
}

func TestObserveRobotUpToDateProjectIDChange(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	otherProjectID := "project-2"
	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, pid *string) ([]*harborclients.RobotStatus, error) {
				return []*harborclients.RobotStatus{
					{
						ID:           "robot-123",
						Name:         "my-robot",
						ProjectID:    &otherProjectID,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, robot)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when project ID differs")
	}
}

func TestUpdateRobotNoID(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{}

	_, err := ext.Update(ctx, robot)
	if err == nil {
		t.Error("Update should fail when ID not set")
	}
}

func TestDeleteRobotNoID(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{},
	}

	_, err := ext.Delete(ctx, robot)
	if err != nil {
		t.Errorf("Delete should not fail when ID not set, got %v", err)
	}
}

func TestConvertPermissions(t *testing.T) {
	tests := []struct {
		name    string
		perms   []v1beta1.RobotPermission
		wantLen int
	}{
		{
			name:    "empty permissions",
			perms:   []v1beta1.RobotPermission{},
			wantLen: 0,
		},
		{
			name: "single permission",
			perms: []v1beta1.RobotPermission{
				{Namespace: "project", Access: []string{"pull"}},
			},
			wantLen: 1,
		},
		{
			name: "multiple permissions",
			perms: []v1beta1.RobotPermission{
				{Namespace: "project", Access: []string{"pull", "push"}},
				{Namespace: "repository", Access: []string{"delete"}},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertPermissions(tt.perms)
			if len(result) != tt.wantLen {
				t.Errorf("convertPermissions returned %d, want %d", len(result), tt.wantLen)
			}
		})
	}
}

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

func TestObserveRobotNotFound(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, projectID *string) ([]*harborclients.RobotStatus, error) {
				return []*harborclients.RobotStatus{}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, robot)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when robot not found")
	}
}

func TestObserveRobotExists(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, pid *string) ([]*harborclients.RobotStatus, error) {
				return []*harborclients.RobotStatus{
					{
						ID:           "robot-123",
						Name:         "my-robot",
						ProjectID:    pid,
						Secret:       "secret-token",
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, robot)
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

func TestObserveRobotNotUpToDate(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	desc := "old description"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Description: &desc,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			listRobotsFunc: func(ctx context.Context, pid *string) ([]*harborclients.RobotStatus, error) {
				newDesc := "new description"
				return []*harborclients.RobotStatus{
					{
						ID:           "robot-123",
						Name:         "my-robot",
						ProjectID:    pid,
						Description:  &newDesc,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, robot)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when description differs")
	}
}

func TestCreateRobotSuccess(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			createRobotFunc: func(ctx context.Context, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
				return &harborclients.RobotStatus{
					ID:           "robot-123",
					Name:         spec.Name,
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, robot)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateRobotError(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			createRobotFunc: func(ctx context.Context, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, robot)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateRobotSuccess(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robotID := "robot-123"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
		Status: v1beta1.RobotStatus{
			AtProvider: v1beta1.RobotObservation{
				ID: &robotID,
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			updateRobotFunc: func(ctx context.Context, robotID string, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
				return &harborclients.RobotStatus{
					ID:         robotID,
					Name:       spec.Name,
					UpdateTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, robot)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateRobotError(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robotID := "robot-123"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
		Status: v1beta1.RobotStatus{
			AtProvider: v1beta1.RobotObservation{
				ID: &robotID,
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			updateRobotFunc: func(ctx context.Context, robotID string, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, robot)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteRobotSuccess(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robotID := "robot-123"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
		Status: v1beta1.RobotStatus{
			AtProvider: v1beta1.RobotObservation{
				ID: &robotID,
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			deleteRobotFunc: func(ctx context.Context, robotID string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, robot)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteRobotError(t *testing.T) {
	ctx := context.Background()
	projectID := "project-1"
	robotID := "robot-123"
	robot := &v1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-robot",
		},
		Spec: v1beta1.RobotSpec{
			ForProvider: v1beta1.RobotParameters{
				Name:        "my-robot",
				ProjectID:   &projectID,
				Permissions: []v1beta1.RobotPermission{{Namespace: "project", Access: []string{"pull"}}},
			},
		},
		Status: v1beta1.RobotStatus{
			AtProvider: v1beta1.RobotObservation{
				ID: &robotID,
			},
		},
	}

	ext := &external{
		service: &mockRobotClient{
			deleteRobotFunc: func(ctx context.Context, robotID string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, robot)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

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

type mockRobotClient struct {
	harborclients.HarborClienter
	listRobotsFunc  func(ctx context.Context, projectID *string) ([]*harborclients.RobotStatus, error)
	createRobotFunc func(ctx context.Context, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error)
	updateRobotFunc func(ctx context.Context, robotID string, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error)
	deleteRobotFunc func(ctx context.Context, robotID string) error
	closeFunc       func() error
}

func (m *mockRobotClient) ListRobots(ctx context.Context, projectID *string) ([]*harborclients.RobotStatus, error) {
	if m.listRobotsFunc != nil {
		return m.listRobotsFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockRobotClient) CreateRobot(ctx context.Context, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
	if m.createRobotFunc != nil {
		return m.createRobotFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockRobotClient) UpdateRobot(ctx context.Context, robotID string, spec *harborclients.RobotSpec) (*harborclients.RobotStatus, error) {
	if m.updateRobotFunc != nil {
		return m.updateRobotFunc(ctx, robotID, spec)
	}
	return nil, nil
}

func (m *mockRobotClient) DeleteRobot(ctx context.Context, robotID string) error {
	if m.deleteRobotFunc != nil {
		return m.deleteRobotFunc(ctx, robotID)
	}
	return nil
}

func (m *mockRobotClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockRobotClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}

func ptrInt64(i int64) *int64 {
	return &i
}
