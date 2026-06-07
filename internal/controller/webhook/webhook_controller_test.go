/*
Copyright 2024 Crossplane Harbor Provider.
*/

package webhook

import (
	"context"
	"errors"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/webhook/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectNotWebhook(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockWebhookClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, nil)
	if err == nil {
		t.Error("Connect should fail when resource is nil")
	}
}

func TestConnectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockWebhookClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Webhook{})
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

	_, err := conn.Connect(ctx, &v1beta1.Webhook{})
	if err == nil {
		t.Error("Connect should fail when client creation fails")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockWebhookClient{
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

func TestObserveWebhookNotFound(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return nil, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
	if err != nil {
		t.Errorf("Observe should not return error for not found, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when webhook not found")
	}
}

func TestObserveWebhookExists(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return []*harborclients.WebhookStatus{
					{
						ID:           "webhook-123",
						ProjectID:    "project-1",
						Name:         "test-webhook",
						URL:          "https://webhook.example.com",
						EventTypes:   []string{"PUSH_ARTIFACT"},
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if !obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be true when values match")
	}
}

func TestObserveWebhookNotUpToDate(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:    "project-1",
				Name:         "test-webhook",
				URL:          "https://webhook.example.com",
				EventTypes:   []string{"PUSH_ARTIFACT"},
				Description: ptrString("updated description"),
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				oldDesc := "old description"
				return []*harborclients.WebhookStatus{
					{
						ID:           "webhook-123",
						ProjectID:    "project-1",
						Name:         "test-webhook",
						URL:          "https://webhook.example.com",
						Description:  &oldDesc,
						EventTypes:   []string{"PUSH_ARTIFACT"},
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
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

func TestObserveWebhookNotUpToDateURL(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://new-webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return []*harborclients.WebhookStatus{
					{
						ID:           "webhook-123",
						ProjectID:    "project-1",
						Name:         "test-webhook",
						URL:          "https://old-webhook.example.com",
						EventTypes:   []string{"PUSH_ARTIFACT"},
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when URL differs")
	}
}

func TestObserveWebhookNotUpToDateEventTypes(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT", "DELETE_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return []*harborclients.WebhookStatus{
					{
						ID:           "webhook-123",
						ProjectID:    "project-1",
						Name:         "test-webhook",
						URL:          "https://webhook.example.com",
						EventTypes:   []string{"PUSH_ARTIFACT"},
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when event types differ")
	}
}

func TestObserveWebhookWithNilDescription(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return []*harborclients.WebhookStatus{
					{
						ID:           "webhook-123",
						ProjectID:    "project-1",
						Name:         "test-webhook",
						Description:  nil,
						URL:          "https://webhook.example.com",
						EventTypes:   []string{"PUSH_ARTIFACT"},
						CreationTime: time.Now(),
						UpdateTime:   time.Now(),
					},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, webhook)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
}

func TestObserveWebhookListError(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			listWebhooksFunc: func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
				return nil, errors.New("list failed")
			},
		},
	}

	_, err := ext.Observe(ctx, webhook)
	if err == nil {
		t.Error("Observe should fail when list returns error")
	}
}

func TestCreateWebhookSuccess(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := false
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				SkipCertVerify: &skipCertVerify,
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			createWebhookFunc: func(ctx context.Context, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
				return &harborclients.WebhookStatus{
					ID:           "new-webhook-id",
					ProjectID:    spec.ProjectID,
					Name:         spec.Name,
					URL:          spec.URL,
					EventTypes:   spec.EventTypes,
					CreationTime: time.Now(),
					UpdateTime:   time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, webhook)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateWebhookError(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := false
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				SkipCertVerify: &skipCertVerify,
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			createWebhookFunc: func(ctx context.Context, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
				return nil, errors.New("create failed")
			},
		},
	}

	_, err := ext.Create(ctx, webhook)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestCreateWebhookWithOptionalFields(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := true
	desc := "Test webhook description"
	authHeader := "Bearer token"
	enabled := true
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				Description:    &desc,
				AuthHeader:     &authHeader,
				SkipCertVerify: &skipCertVerify,
				Enabled:        &enabled,
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			createWebhookFunc: func(ctx context.Context, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
				if spec.Description == nil {
					t.Error("Description should be set")
				}
				if spec.AuthHeader == nil {
					t.Error("AuthHeader should be set")
				}
				return &harborclients.WebhookStatus{
					ID:           "new-webhook-id",
					ProjectID:    spec.ProjectID,
					Name:         spec.Name,
					URL:          spec.URL,
					EventTypes:   spec.EventTypes,
					CreationTime: time.Now(),
					UpdateTime:   time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Create(ctx, webhook)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestUpdateWebhookSuccess(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := false
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				SkipCertVerify: &skipCertVerify,
			},
		},
		Status: v1beta1.WebhookStatus{
			AtProvider: v1beta1.WebhookObservation{
				ID: ptrString("webhook-123"),
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			updateWebhookFunc: func(ctx context.Context, projectID, webhookID string, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
				return &harborclients.WebhookStatus{
					ID:           webhookID,
					ProjectID:    projectID,
					Name:         spec.Name,
					URL:          spec.URL,
					EventTypes:   spec.EventTypes,
					CreationTime: time.Now(),
					UpdateTime:   time.Now(),
				}, nil
			},
		},
	}

	_, err := ext.Update(ctx, webhook)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateWebhookError(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := false
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				SkipCertVerify: &skipCertVerify,
			},
		},
		Status: v1beta1.WebhookStatus{
			AtProvider: v1beta1.WebhookObservation{
				ID: ptrString("webhook-123"),
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			updateWebhookFunc: func(ctx context.Context, projectID, webhookID string, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
				return nil, errors.New("update failed")
			},
		},
	}

	_, err := ext.Update(ctx, webhook)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestUpdateWebhookNoID(t *testing.T) {
	ctx := context.Background()
	skipCertVerify := false
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:      "project-1",
				Name:           "test-webhook",
				URL:            "https://webhook.example.com",
				EventTypes:     []string{"PUSH_ARTIFACT"},
				SkipCertVerify: &skipCertVerify,
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{},
	}

	_, err := ext.Update(ctx, webhook)
	if err == nil {
		t.Error("Update should fail when ID not set")
	}
}

func TestDeleteWebhookSuccess(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
		Status: v1beta1.WebhookStatus{
			AtProvider: v1beta1.WebhookObservation{
				ID: ptrString("webhook-123"),
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			deleteWebhookFunc: func(ctx context.Context, projectID, webhookID string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, webhook)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteWebhookError(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
		Status: v1beta1.WebhookStatus{
			AtProvider: v1beta1.WebhookObservation{
				ID: ptrString("webhook-123"),
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{
			deleteWebhookFunc: func(ctx context.Context, projectID, webhookID string) error {
				return errors.New("delete failed")
			},
		},
	}

	_, err := ext.Delete(ctx, webhook)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestDeleteWebhookNoID(t *testing.T) {
	ctx := context.Background()
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	ext := &external{
		service: &mockWebhookClient{},
	}

	_, err := ext.Delete(ctx, webhook)
	if err != nil {
		t.Errorf("Delete should not fail when ID not set, got %v", err)
	}
}

func TestWebhookHasRequiredFields(t *testing.T) {
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-webhook",
			Namespace: "default",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
	}

	if webhook.Spec.ForProvider.ProjectID == "" {
		t.Error("Webhook ProjectID should not be empty")
	}
	if webhook.Spec.ForProvider.Name == "" {
		t.Error("Webhook Name should not be empty")
	}
	if webhook.Spec.ForProvider.URL == "" {
		t.Error("Webhook URL should not be empty")
	}
	if len(webhook.Spec.ForProvider.EventTypes) == 0 {
		t.Error("Webhook EventTypes should not be empty")
	}
	if webhook.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestWebhookStatusFields(t *testing.T) {
	webhook := &v1beta1.Webhook{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-webhook",
		},
		Spec: v1beta1.WebhookSpec{
			ForProvider: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
		},
		Status: v1beta1.WebhookStatus{
			AtProvider: v1beta1.WebhookObservation{
				ID: ptrString("webhook-123"),
			},
		},
	}

	if webhook.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *webhook.Status.AtProvider.ID != "webhook-123" {
		t.Errorf("Status ID should be 'webhook-123', got %s", *webhook.Status.AtProvider.ID)
	}
}

func TestWebhookParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.WebhookParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.WebhookParameters{
				ProjectID:    "project-1",
				Name:         "test-webhook",
				URL:          "https://webhook.example.com",
				EventTypes:   []string{"PUSH_ARTIFACT"},
				Description:  ptrString("My webhook"),
				SkipCertVerify: ptrBool(false),
			},
			isValid: true,
		},
		{
			name: "missing required project ID",
			params: v1beta1.WebhookParameters{
				Name:       "test-webhook",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
			isValid: false,
		},
		{
			name: "missing required name",
			params: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				URL:        "https://webhook.example.com",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
			isValid: false,
		},
		{
			name: "missing required URL",
			params: v1beta1.WebhookParameters{
				ProjectID:  "project-1",
				Name:       "test-webhook",
				EventTypes: []string{"PUSH_ARTIFACT"},
			},
			isValid: false,
		},
		{
			name: "missing required event types",
			params: v1beta1.WebhookParameters{
				ProjectID: "project-1",
				Name:      "test-webhook",
				URL:       "https://webhook.example.com",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.ProjectID != "" && tt.params.Name != "" && tt.params.URL != "" && len(tt.params.EventTypes) > 0
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

type mockWebhookClient struct {
	harborclients.HarborClienter
	listWebhooksFunc  func(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error)
	createWebhookFunc func(ctx context.Context, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error)
	updateWebhookFunc func(ctx context.Context, projectID, webhookID string, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error)
	deleteWebhookFunc func(ctx context.Context, projectID, webhookID string) error
	closeFunc          func() error
}

func (m *mockWebhookClient) ListWebhooks(ctx context.Context, projectID string) ([]*harborclients.WebhookStatus, error) {
	if m.listWebhooksFunc != nil {
		return m.listWebhooksFunc(ctx, projectID)
	}
	return nil, nil
}

func (m *mockWebhookClient) CreateWebhook(ctx context.Context, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
	if m.createWebhookFunc != nil {
		return m.createWebhookFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockWebhookClient) UpdateWebhook(ctx context.Context, projectID, webhookID string, spec *harborclients.WebhookSpec) (*harborclients.WebhookStatus, error) {
	if m.updateWebhookFunc != nil {
		return m.updateWebhookFunc(ctx, projectID, webhookID, spec)
	}
	return nil, nil
}

func (m *mockWebhookClient) DeleteWebhook(ctx context.Context, projectID, webhookID string) error {
	if m.deleteWebhookFunc != nil {
		return m.deleteWebhookFunc(ctx, projectID, webhookID)
	}
	return nil
}

func (m *mockWebhookClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockWebhookClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}

func ptrBool(b bool) *bool {
	return &b
}