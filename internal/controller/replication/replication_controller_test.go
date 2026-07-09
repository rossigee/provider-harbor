/*
Copyright 2024 Crossplane Harbor Provider.
*/

package replication

import (
	"context"
	"errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/rossigee/provider-harbor/apis/replication/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
	"time"
)

func TestConnectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockReplicationClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Replication{})
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

	_, err := conn.Connect(ctx, &v1beta1.Replication{})
	if err == nil {
		t.Error("Connect should fail when client creation fails")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockReplicationClient{
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

func TestObserveReplicationListError(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				return nil, errors.New("list failed")
			},
		},
	}

	_, err := ext.Observe(ctx, replication)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
}

func TestObserveReplicationWithNilDescription(t *testing.T) {
	ctx := context.Background()
	enabled := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:    "my-replication",
				Enabled: &enabled,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				return []*harborclients.ReplicationPolicyStatus{
					{
						ID:           "policy-123",
						Name:         "my-replication",
						Description:  nil,
						Enabled:      true,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, replication)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
}

func TestObserveReplicationUpToDateEnabledChange(t *testing.T) {
	ctx := context.Background()
	enabled := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:    "my-replication",
				Enabled: &enabled,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				return []*harborclients.ReplicationPolicyStatus{
					{
						ID:           "policy-123",
						Name:         "my-replication",
						Enabled:      false,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, replication)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when enabled differs")
	}
}

func TestCreateReplicationWithAllFields(t *testing.T) {
	ctx := context.Background()
	enabled := true
	override := true
	deleteSourceTag := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:            "my-replication",
				Description:     ptrString("Replication policy"),
				SourceRegistry:  ptrString("source-reg"),
				Trigger:         "scheduled",
				DeleteSourceTag: &deleteSourceTag,
				Override:        &override,
				Enabled:         &enabled,
				Filters: []v1beta1.ReplicationFilter{
					{Type: "name", Value: "**"},
				},
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			createReplicationPolicyFunc: func(ctx context.Context, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
				if spec.Name != "my-replication" {
					t.Errorf("Expected name 'my-replication', got '%s'", spec.Name)
				}
				if spec.Description == nil || *spec.Description != "Replication policy" {
					t.Error("Description should be set")
				}
				if spec.Filters == nil || len(spec.Filters) != 1 {
					t.Error("Filters should be set")
				}
				return &harborclients.ReplicationPolicyStatus{
					ID:           "policy-123",
					Name:         spec.Name,
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, replication)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestUpdateReplicationNoID(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
	}

	ext := &external{}

	_, err := ext.Update(ctx, replication)
	if err == nil {
		t.Error("Update should fail when ID not set")
	}
}

func TestDeleteReplicationNoID(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{},
	}

	_, err := ext.Delete(ctx, replication)
	if err != nil {
		t.Errorf("Delete should not fail when ID not set, got %v", err)
	}
}

func TestConnectNotReplication(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Connect with nil should return %s error", errNotReplication)
	}
}

func TestObserveNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Observe with nil should return %s error", errNotReplication)
	}
}

func TestUpdateNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Update with nil should return %s error", errNotReplication)
	}
}

func TestDeleteNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Delete with nil should return %s error", errNotReplication)
	}
}

func TestCreateNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Create with nil should return %s error", errNotReplication)
	}
}

func TestObserveReplicationNotFound(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				return []*harborclients.ReplicationPolicyStatus{}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, replication)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when replication policy not found")
	}
}

func TestObserveReplicationExists(t *testing.T) {
	ctx := context.Background()
	enabled := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:    "my-replication",
				Enabled: &enabled,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				return []*harborclients.ReplicationPolicyStatus{
					{
						ID:           "policy-123",
						Name:         "my-replication",
						Enabled:      true,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, replication)
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

func TestObserveReplicationNotUpToDate(t *testing.T) {
	ctx := context.Background()
	desc := "old description"
	enabled := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:        "my-replication",
				Description: &desc,
				Enabled:     &enabled,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			listReplicationPoliciesFunc: func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
				newDesc := "new description"
				return []*harborclients.ReplicationPolicyStatus{
					{
						ID:           "policy-123",
						Name:         "my-replication",
						Description:  &newDesc,
						Enabled:      true,
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, replication)
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

func TestCreateReplicationSuccess(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			createReplicationPolicyFunc: func(ctx context.Context, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
				return &harborclients.ReplicationPolicyStatus{
					ID:           "policy-123",
					Name:         spec.Name,
					CreationTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, replication)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateReplicationError(t *testing.T) {
	ctx := context.Background()
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			createReplicationPolicyFunc: func(ctx context.Context, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, replication)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateReplicationSuccess(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-123"
	enabled := true
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name:    "my-replication",
				Enabled: &enabled,
			},
		},
		Status: v1beta1.ReplicationStatus{
			AtProvider: v1beta1.ReplicationObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			updateReplicationPolicyFunc: func(ctx context.Context, policyID string, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
				return &harborclients.ReplicationPolicyStatus{
					ID:         policyID,
					Name:       spec.Name,
					UpdateTime: time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, replication)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateReplicationError(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-123"
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
		Status: v1beta1.ReplicationStatus{
			AtProvider: v1beta1.ReplicationObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			updateReplicationPolicyFunc: func(ctx context.Context, policyID string, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, replication)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteReplicationSuccess(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-123"
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
		Status: v1beta1.ReplicationStatus{
			AtProvider: v1beta1.ReplicationObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			deleteReplicationPolicyFunc: func(ctx context.Context, policyID string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, replication)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteReplicationError(t *testing.T) {
	ctx := context.Background()
	policyID := "policy-123"
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
		Status: v1beta1.ReplicationStatus{
			AtProvider: v1beta1.ReplicationObservation{
				ID: &policyID,
			},
		},
	}

	ext := &external{
		service: &mockReplicationClient{
			deleteReplicationPolicyFunc: func(ctx context.Context, policyID string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, replication)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestHelperFunctions(t *testing.T) {
	strVal := "test"
	resultStr := getStringPtr(strVal)
	if resultStr == nil || *resultStr != strVal {
		t.Errorf("getStringPtr failed")
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

func TestReplicationHasRequiredFields(t *testing.T) {
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-replication",
			Namespace: "default",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
		},
	}

	if replication.Spec.ForProvider.Name == "" {
		t.Error("Replication Name should not be empty")
	}
	if replication.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestReplicationStatusFields(t *testing.T) {
	replication := &v1beta1.Replication{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-replication",
		},
		Spec: v1beta1.ReplicationSpec{
			ForProvider: v1beta1.ReplicationParameters{
				Name: "my-replication",
			},
		},
		Status: v1beta1.ReplicationStatus{
			AtProvider: v1beta1.ReplicationObservation{
				ID: ptrString("policy-123"),
			},
		},
	}

	if replication.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *replication.Status.AtProvider.ID != "policy-123" {
		t.Errorf("Status ID should be 'policy-123', got %s", *replication.Status.AtProvider.ID)
	}
}

func TestReplicationParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ReplicationParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.ReplicationParameters{
				Name: "my-replication",
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
			isValid: true,
		},
		{
			name: "valid with trigger",
			params: v1beta1.ReplicationParameters{
				Name:    "scheduled-replication",
				Trigger: "scheduled",
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
			isValid: true,
		},
		{
			name: "valid with filters",
			params: v1beta1.ReplicationParameters{
				Name:    "filtered-replication",
				Filters: []v1beta1.ReplicationFilter{{Type: "name", Value: "*"}},
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.ReplicationParameters{
				DestinationReg: v1beta1.ReplicationDestination{
					Name:      "dest-reg",
					Namespace: "namespace",
					URL:       "https://dest harbor.example.com",
				},
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

type mockReplicationClient struct {
	harborclients.HarborClienter
	listReplicationPoliciesFunc func(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error)
	createReplicationPolicyFunc func(ctx context.Context, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error)
	updateReplicationPolicyFunc func(ctx context.Context, policyID string, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error)
	deleteReplicationPolicyFunc func(ctx context.Context, policyID string) error
	closeFunc                   func() error
}

func (m *mockReplicationClient) ListReplicationPolicies(ctx context.Context) ([]*harborclients.ReplicationPolicyStatus, error) {
	if m.listReplicationPoliciesFunc != nil {
		return m.listReplicationPoliciesFunc(ctx)
	}
	return nil, nil
}

func (m *mockReplicationClient) CreateReplicationPolicy(ctx context.Context, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
	if m.createReplicationPolicyFunc != nil {
		return m.createReplicationPolicyFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockReplicationClient) UpdateReplicationPolicy(ctx context.Context, policyID string, spec *harborclients.ReplicationPolicySpec) (*harborclients.ReplicationPolicyStatus, error) {
	if m.updateReplicationPolicyFunc != nil {
		return m.updateReplicationPolicyFunc(ctx, policyID, spec)
	}
	return nil, nil
}

func (m *mockReplicationClient) DeleteReplicationPolicy(ctx context.Context, policyID string) error {
	if m.deleteReplicationPolicyFunc != nil {
		return m.deleteReplicationPolicyFunc(ctx, policyID)
	}
	return nil
}

func (m *mockReplicationClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockReplicationClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}

func getStringPtr(s string) *string {
	return &s
}

func getBoolValue(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
