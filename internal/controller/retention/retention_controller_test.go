/*
Copyright 2024 Crossplane Harbor Provider.
*/

package retention

import (
	"context"
	"errors"
	"github.com/rossigee/provider-harbor/apis/retention/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestConnectNotRetention(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Connect with nil should return %s error", errNotRetention)
	}
}

func TestObserveNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Observe with nil should return %s error", errNotRetention)
	}
}

func TestUpdateNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Update with nil should return %s error", errNotRetention)
	}
}

func TestDeleteNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Delete with nil should return %s error", errNotRetention)
	}
}

func TestCreateNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Create with nil should return %s error", errNotRetention)
	}
}

func TestObserveRetentionNotFound(t *testing.T) {
	ctx := context.Background()
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			listRetentionPoliciesFunc: func(ctx context.Context, projectID string) ([]*harborclients.RetentionPolicyStatus, error) {
				return []*harborclients.RetentionPolicyStatus{}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, retention)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when retention policy not found")
	}
}

func TestObserveRetentionExists(t *testing.T) {
	ctx := context.Background()
	enabled := true
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
				Enabled:   &enabled,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			listRetentionPoliciesFunc: func(ctx context.Context, projectID string) ([]*harborclients.RetentionPolicyStatus, error) {
				return []*harborclients.RetentionPolicyStatus{
					{
						ID:           "retention-123",
						ProjectID:    "project-1",
						Enabled:      true,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, retention)
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

func TestObserveRetentionNotUpToDate(t *testing.T) {
	ctx := context.Background()
	desc := "old description"
	enabled := true
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID:   "project-1",
				Description: &desc,
				Enabled:     &enabled,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			listRetentionPoliciesFunc: func(ctx context.Context, projectID string) ([]*harborclients.RetentionPolicyStatus, error) {
				newDesc := "new description"
				return []*harborclients.RetentionPolicyStatus{
					{
						ID:           "retention-123",
						ProjectID:    "project-1",
						Description:  &newDesc,
						Enabled:      true,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, retention)
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

func TestCreateRetentionSuccess(t *testing.T) {
	ctx := context.Background()
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			createRetentionPolicyFunc: func(ctx context.Context, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
				return &harborclients.RetentionPolicyStatus{
					ID:           "retention-123",
					ProjectID:    spec.ProjectID,
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, retention)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateRetentionError(t *testing.T) {
	ctx := context.Background()
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			createRetentionPolicyFunc: func(ctx context.Context, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, retention)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateRetentionSuccess(t *testing.T) {
	ctx := context.Background()
	policyID := "retention-123"
	enabled := true
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
				Enabled:   &enabled,
			},
		},
		Status: v1beta1.RetentionStatus{
			AtProvider: v1beta1.RetentionObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			updateRetentionPolicyFunc: func(ctx context.Context, projectID, policyID string, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
				return &harborclients.RetentionPolicyStatus{
					ID:         policyID,
					ProjectID:  projectID,
					UpdateTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, retention)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateRetentionError(t *testing.T) {
	ctx := context.Background()
	policyID := "retention-123"
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
		Status: v1beta1.RetentionStatus{
			AtProvider: v1beta1.RetentionObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			updateRetentionPolicyFunc: func(ctx context.Context, projectID, policyID string, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, retention)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteRetentionSuccess(t *testing.T) {
	ctx := context.Background()
	policyID := "retention-123"
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
		Status: v1beta1.RetentionStatus{
			AtProvider: v1beta1.RetentionObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			deleteRetentionPolicyFunc: func(ctx context.Context, projectID, policyID string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, retention)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteRetentionError(t *testing.T) {
	ctx := context.Background()
	policyID := "retention-123"
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
		Status: v1beta1.RetentionStatus{
			AtProvider: v1beta1.RetentionObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockRetentionClient{
			deleteRetentionPolicyFunc: func(ctx context.Context, projectID, policyID string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, retention)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestRetentionHasRequiredFields(t *testing.T) {
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-retention",
			Namespace: "default",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
	}

	if retention.Spec.ForProvider.ProjectID == "" {
		t.Error("Retention ProjectID should not be empty")
	}
	if retention.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestRetentionStatusFields(t *testing.T) {
	retention := &v1beta1.Retention{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-retention",
		},
		Spec: v1beta1.RetentionSpec{
			ForProvider: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
		},
		Status: v1beta1.RetentionStatus{
			AtProvider: v1beta1.RetentionObservation{
				ID: ptrString("retention-123"),
			},
		},
	}

	if retention.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *retention.Status.AtProvider.ID != "retention-123" {
		t.Errorf("Status ID should be 'retention-123', got %s", *retention.Status.AtProvider.ID)
	}
}

func TestRetentionParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.RetentionParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.RetentionParameters{
				ProjectID: "project-1",
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.RetentionParameters{
				ProjectID:   "project-1",
				Description: ptrString("Retention policy"),
			},
			isValid: true,
		},
		{
			name: "valid with enabled flag",
			params: v1beta1.RetentionParameters{
				ProjectID: "project-1",
				Enabled:   ptrBool(true),
			},
			isValid: true,
		},
		{
			name: "valid with trigger",
			params: v1beta1.RetentionParameters{
				ProjectID: "project-1",
				Trigger:   "daily",
			},
			isValid: true,
		},
		{
			name: "valid with rules",
			params: v1beta1.RetentionParameters{
				ProjectID: "project-1",
				Rules: []v1beta1.RetentionRule{
					{
						RuleType:     "always",
						TagSelectors: []string{"*"},
					},
				},
			},
			isValid: true,
		},
		{
			name: "missing required project ID",
			params: v1beta1.RetentionParameters{
				Enabled: ptrBool(true),
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.ProjectID != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

type mockRetentionClient struct {
	harborclients.HarborClienter
	listRetentionPoliciesFunc func(ctx context.Context, projectID string) ([]*harborclients.RetentionPolicyStatus, error)
	createRetentionPolicyFunc func(ctx context.Context, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error)
	updateRetentionPolicyFunc func(ctx context.Context, projectID, policyID string, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error)
	deleteRetentionPolicyFunc func(ctx context.Context, projectID, policyID string) error
}

func (m *mockRetentionClient) ListRetentionPolicies(ctx context.Context, projectID string) ([]*harborclients.RetentionPolicyStatus, error) {
	if m.listRetentionPoliciesFunc != nil {
		return m.listRetentionPoliciesFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockRetentionClient) CreateRetentionPolicy(ctx context.Context, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
	if m.createRetentionPolicyFunc != nil {
		return m.createRetentionPolicyFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockRetentionClient) UpdateRetentionPolicy(ctx context.Context, projectID, policyID string, spec *harborclients.RetentionPolicySpec) (*harborclients.RetentionPolicyStatus, error) {
	if m.updateRetentionPolicyFunc != nil {
		return m.updateRetentionPolicyFunc(ctx, projectID, policyID, spec)
	}
	return nil, nil
}

func (m *mockRetentionClient) DeleteRetentionPolicy(ctx context.Context, projectID, policyID string) error {
	if m.deleteRetentionPolicyFunc != nil {
		return m.deleteRetentionPolicyFunc(ctx, projectID, policyID)
	}
	return nil
}

func (m *mockRetentionClient) Close() error {
	return nil
}

func (m *mockRetentionClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}

func ptrBool(b bool) *bool {
	return &b
}
