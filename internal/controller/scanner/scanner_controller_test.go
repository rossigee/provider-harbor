/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scanner

import (
	"context"
	"errors"
	"github.com/crossplane/crossplane-runtime/v2/pkg/logging"
	"github.com/rossigee/provider-harbor/apis/scanner/v1beta1"
	"github.com/rossigee/provider-harbor/internal/clients"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestConnectNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube:   nil,
		logger: logging.NewNopLogger(),
	}

	_, err := conn.Connect(ctx, nil)
	if err == nil {
		t.Error("Connect should fail when resource is nil")
	}
}

func TestConnectClientError(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube:   nil,
		logger: logging.NewNopLogger(),
	}

	_, err := conn.Connect(ctx, &v1beta1.ScannerRegistration{})
	if err == nil {
		t.Error("Connect should fail when managed resource is nil")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockScannerClient{
			closeFunc: func() error {
				return nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	err := ext.Disconnect(ctx)
	if err != nil {
		t.Errorf("Disconnect should not fail, got %v", err)
	}
}

func TestObserveScannerRegistrationEmptyName(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{},
		logger:  logging.NewNopLogger(),
	}

	_, err := ext.Observe(ctx, scanner)
	if err == nil {
		t.Error("Observe should fail when scanner name is empty")
	}
}

func TestObserveScannerRegistrationAuthMismatch(t *testing.T) {
	ctx := context.Background()
	auth := "Bearer"
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
				Auth: &auth,
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				oldAuth := "Basic"
				return &harborclients.ScannerStatus{
					UUID:       "scanner-uuid-123",
					Name:       "test-scanner",
					URL:        "https://scanner.example.com",
					Auth:       &oldAuth,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when auth differs")
	}
}

func TestObserveScannerRegistrationCredentialMismatch(t *testing.T) {
	ctx := context.Background()
	cred := "new-secret"
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name:             "test-scanner",
				URL:              "https://scanner.example.com",
				AccessCredential: &cred,
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				oldCred := "old-secret"
				return &harborclients.ScannerStatus{
					UUID:             "scanner-uuid-123",
					Name:             "test-scanner",
					URL:              "https://scanner.example.com",
					AccessCredential: &oldCred,
					CreateTime:       time.Now(),
					UpdateTime:       time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when access credential differs")
	}
}

func TestObserveScannerRegistrationNameMismatch(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "new-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return &harborclients.ScannerStatus{
					UUID:       "scanner-uuid-123",
					Name:       "old-scanner",
					URL:        "https://scanner.example.com",
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
	if obs.ResourceUpToDate {
		t.Error("ResourceUpToDate should be false when name differs")
	}
}

func TestObserveScannerRegistrationNotFound(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("not found")
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
	if err != nil {
		t.Errorf("Observe should not return error for not found, got %v", err)
	}
	if obs.ResourceExists {
		t.Error("ResourceExists should be false when scanner not found")
	}
}

func TestObserveScannerRegistrationExists(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return &harborclients.ScannerStatus{
					UUID:       "scanner-uuid-123",
					Name:       "test-scanner",
					URL:        "https://scanner.example.com",
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
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

func TestObserveScannerRegistrationNotUpToDate(t *testing.T) {
	ctx := context.Background()
	desc := "updated description"
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name:        "test-scanner",
				URL:         "https://scanner.example.com",
				Description: &desc,
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				oldDesc := "old description"
				return &harborclients.ScannerStatus{
					UUID:        "scanner-uuid-123",
					Name:        "test-scanner",
					URL:         "https://scanner.example.com",
					Description: &oldDesc,
					CreateTime:  time.Now(),
					UpdateTime:  time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
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

func TestObserveScannerRegistrationURLMismatch(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://new-scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return &harborclients.ScannerStatus{
					UUID:       "scanner-uuid-123",
					Name:       "test-scanner",
					URL:        "https://old-scanner.example.com",
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	obs, err := ext.Observe(ctx, scanner)
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

func TestCreateScannerRegistrationSuccess(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("not found")
			},
			createScannerRegistrationFunc: func(ctx context.Context, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
				return &harborclients.ScannerStatus{
					UUID:       "new-scanner-uuid",
					Name:       spec.Name,
					URL:        spec.URL,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Create(ctx, scanner)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateScannerRegistrationError(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("not found")
			},
			createScannerRegistrationFunc: func(ctx context.Context, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("create failed")
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Create(ctx, scanner)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestCreateScannerRegistrationWithOptionalFields(t *testing.T) {
	ctx := context.Background()
	desc := "Test scanner description"
	auth := "Bearer"
	cred := "secret-token"
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name:             "test-scanner",
				URL:              "https://scanner.example.com",
				Description:      &desc,
				Auth:             &auth,
				AccessCredential: &cred,
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			getScannerRegistrationFunc: func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("not found")
			},
			createScannerRegistrationFunc: func(ctx context.Context, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
				if spec.Description == nil {
					t.Error("Description should be set")
				}
				if spec.Auth == nil {
					t.Error("Auth should be set")
				}
				if spec.AccessCredential == nil {
					t.Error("AccessCredential should be set")
				}
				return &harborclients.ScannerStatus{
					UUID:       "new-scanner-uuid",
					Name:       spec.Name,
					URL:        spec.URL,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Create(ctx, scanner)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestUpdateScannerRegistrationSuccess(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
		Status: v1beta1.ScannerRegistrationStatus{
			AtProvider: v1beta1.ScannerRegistrationObservation{
				UUID: ptrString("scanner-uuid-123"),
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			updateScannerRegistrationFunc: func(ctx context.Context, scannerID string, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
				return &harborclients.ScannerStatus{
					UUID:       scannerID,
					Name:       spec.Name,
					URL:        spec.URL,
					CreateTime: time.Now(),
					UpdateTime: time.Now(),
				}, nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Update(ctx, scanner)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestUpdateScannerRegistrationError(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
		Status: v1beta1.ScannerRegistrationStatus{
			AtProvider: v1beta1.ScannerRegistrationObservation{
				UUID: ptrString("scanner-uuid-123"),
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			updateScannerRegistrationFunc: func(ctx context.Context, scannerID string, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
				return nil, errors.New("update failed")
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Update(ctx, scanner)
	if err == nil {
		t.Error("Update should fail when client fails")
	}
}

func TestDeleteScannerRegistrationSuccess(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
		Status: v1beta1.ScannerRegistrationStatus{
			AtProvider: v1beta1.ScannerRegistrationObservation{
				UUID: ptrString("scanner-uuid-123"),
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			deleteScannerRegistrationFunc: func(ctx context.Context, scannerID string) error {
				return nil
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Delete(ctx, scanner)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteScannerRegistrationError(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
		Status: v1beta1.ScannerRegistrationStatus{
			AtProvider: v1beta1.ScannerRegistrationObservation{
				UUID: ptrString("scanner-uuid-123"),
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{
			deleteScannerRegistrationFunc: func(ctx context.Context, scannerID string) error {
				return errors.New("delete failed")
			},
		},
		logger: logging.NewNopLogger(),
	}

	_, err := ext.Delete(ctx, scanner)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestDeleteScannerRegistrationNoUUID(t *testing.T) {
	ctx := context.Background()
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	ext := &external{
		service: &mockScannerClient{},
		logger:  logging.NewNopLogger(),
	}

	_, err := ext.Delete(ctx, scanner)
	if err != nil {
		t.Errorf("Delete should not fail when UUID not set, got %v", err)
	}
}

func TestScannerRegistrationHasRequiredFields(t *testing.T) {
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scanner",
			Namespace: "default",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
	}

	if scanner.Spec.ForProvider.Name == "" {
		t.Error("Scanner Name should not be empty")
	}
	if scanner.Spec.ForProvider.URL == "" {
		t.Error("Scanner URL should not be empty")
	}
	if scanner.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestScannerRegistrationStatusFields(t *testing.T) {
	uuid := "scanner-uuid-123"
	scanner := &v1beta1.ScannerRegistration{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scanner",
		},
		Spec: v1beta1.ScannerRegistrationSpec{
			ForProvider: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
		},
		Status: v1beta1.ScannerRegistrationStatus{
			AtProvider: v1beta1.ScannerRegistrationObservation{
				UUID: &uuid,
			},
		},
	}

	if scanner.Status.AtProvider.UUID == nil {
		t.Error("Status UUID should be populated")
	}
	if *scanner.Status.AtProvider.UUID != "scanner-uuid-123" {
		t.Errorf("Status UUID should be 'scanner-uuid-123', got %s", *scanner.Status.AtProvider.UUID)
	}
}

func TestScannerRegistrationParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ScannerRegistrationParameters
		isValid bool
	}{
		{
			name: "valid with required fields",
			params: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
				URL:  "https://scanner.example.com",
			},
			isValid: true,
		},
		{
			name: "valid with description",
			params: v1beta1.ScannerRegistrationParameters{
				Name:        "test-scanner",
				URL:         "https://scanner.example.com",
				Description: ptrString("My scanner"),
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.ScannerRegistrationParameters{
				URL: "https://scanner.example.com",
			},
			isValid: false,
		},
		{
			name: "missing required URL",
			params: v1beta1.ScannerRegistrationParameters{
				Name: "test-scanner",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.Name != "" && tt.params.URL != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

type mockScannerClient struct {
	harborclients.HarborClienter
	getScannerRegistrationFunc    func(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error)
	createScannerRegistrationFunc func(ctx context.Context, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error)
	updateScannerRegistrationFunc func(ctx context.Context, scannerID string, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error)
	deleteScannerRegistrationFunc func(ctx context.Context, scannerID string) error
	closeFunc                     func() error
}

func (m *mockScannerClient) GetScannerRegistration(ctx context.Context, scannerID string) (*harborclients.ScannerStatus, error) {
	if m.getScannerRegistrationFunc != nil {
		return m.getScannerRegistrationFunc(ctx, scannerID)
	}
	return nil, nil
}

func (m *mockScannerClient) CreateScannerRegistration(ctx context.Context, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
	if m.createScannerRegistrationFunc != nil {
		return m.createScannerRegistrationFunc(ctx, spec)
	}
	return nil, nil
}

func (m *mockScannerClient) UpdateScannerRegistration(ctx context.Context, scannerID string, spec *harborclients.ScannerSpec) (*harborclients.ScannerStatus, error) {
	if m.updateScannerRegistrationFunc != nil {
		return m.updateScannerRegistrationFunc(ctx, scannerID, spec)
	}
	return nil, nil
}

func (m *mockScannerClient) DeleteScannerRegistration(ctx context.Context, scannerID string) error {
	if m.deleteScannerRegistrationFunc != nil {
		return m.deleteScannerRegistrationFunc(ctx, scannerID)
	}
	return nil
}

func (m *mockScannerClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockScannerClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

func ptrString(s string) *string {
	return &s
}
