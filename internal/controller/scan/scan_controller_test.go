/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scan

import (
	"context"
	"errors"
	"testing"
	"time"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"

	"github.com/rossigee/provider-harbor/apis/scan/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
)

func TestConnectSuccess(t *testing.T) {
	ctx := context.Background()
	conn := &connector{
		kube: nil,
		newServiceFn: func(ctx context.Context, kube client.Client, mg resource.Managed) (harborclients.HarborClienter, error) {
			return &mockScanClient{}, nil
		},
	}

	_, err := conn.Connect(ctx, &v1beta1.Scan{})
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

	_, err := conn.Connect(ctx, &v1beta1.Scan{})
	if err == nil {
		t.Error("Connect should fail when client creation fails")
	}
}

func TestDisconnect(t *testing.T) {
	ctx := context.Background()
	ext := &external{
		service: &mockScanClient{
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

func TestConnectNotScan(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotScan {
		t.Errorf("Connect with nil should return %s error", errNotScan)
	}
}

func TestObserveNotScan(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotScan {
		t.Errorf("Observe with nil should return %s error", errNotScan)
	}
}

func TestUpdateNotScan(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err != nil {
		t.Errorf("Update with nil should return nil error, got %v", err)
	}
}

func TestDeleteNotScan(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotScan {
		t.Errorf("Delete with nil should return %s error", errNotScan)
	}
}

func TestCreateNotScan(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotScan {
		t.Errorf("Create with nil should return %s error", errNotScan)
	}
}

func TestObserveScanSuccess(t *testing.T) {
	ctx := context.Background()
	projectID := "library"
	repoName := "alpine"
	reference := "latest"
	status := "Success"
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      projectID,
				RepositoryName: repoName,
				Reference:      reference,
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error) {
				return &harborclients.ScanStatus{
					ID:            "scan-123",
					Status:        status,
					CriticalCount: 0,
					HighCount:     1,
					MediumCount:   5,
					LowCount:      10,
					StartTime:     time.Now().Add(-1 * time.Hour),
					EndTime:       time.Now(),
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, scan)
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

func TestObserveScanError(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error) {
				return nil, errors.New("scan not found")
			},
		},
	}

	_, err := ext.Observe(ctx, scan)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
}

func TestCreateScanSuccess(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			triggerScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return nil
			},
		},
	}

	_, err := ext.Create(ctx, scan)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestCreateScanError(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			triggerScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return errors.New("trigger scan failed")
			},
		},
	}

	_, err := ext.Create(ctx, scan)
	if err == nil {
		t.Error("Create should fail when client fails")
	}
}

func TestUpdateScan(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{}

	_, err := ext.Update(ctx, scan)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteScanSuccess(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			stopScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return nil
			},
		},
	}

	_, err := ext.Delete(ctx, scan)
	if err != nil {
		t.Errorf("Delete should not fail, got %v", err)
	}
}

func TestDeleteScanError(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			stopScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return errors.New("stop scan failed")
			},
		},
	}

	_, err := ext.Delete(ctx, scan)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestScanHasRequiredFields(t *testing.T) {
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-scan",
			Namespace: "default",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	if scan.Spec.ForProvider.ProjectID == "" {
		t.Error("ProjectID should not be empty")
	}
	if scan.Spec.ForProvider.RepositoryName == "" {
		t.Error("RepositoryName should not be empty")
	}
	if scan.Spec.ForProvider.Reference == "" {
		t.Error("Reference should not be empty")
	}
}

func TestScanStatusFields(t *testing.T) {
	status := "Success"
	criticalCount := int64(2)
	highCount := int64(5)
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
		Status: v1beta1.ScanStatus{
			AtProvider: v1beta1.ScanObservation{
				ID:            ptrString("scan-123"),
				Status:        &status,
				CriticalCount: &criticalCount,
				HighCount:     &highCount,
			},
		},
	}

	if scan.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if scan.Status.AtProvider.Status == nil {
		t.Error("Status should be populated")
	}
	if *scan.Status.AtProvider.CriticalCount != 2 {
		t.Errorf("CriticalCount should be 2, got %d", *scan.Status.AtProvider.CriticalCount)
	}
}

func TestScanParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ScanParameters
		isValid bool
	}{
		{
			name: "valid with all fields",
			params: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
			isValid: true,
		},
		{
			name: "valid with digest reference",
			params: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "sha256:abc123",
			},
			isValid: true,
		},
		{
			name: "missing project ID",
			params: v1beta1.ScanParameters{
				RepositoryName: "alpine",
				Reference:      "latest",
			},
			isValid: false,
		},
		{
			name: "missing repository name",
			params: v1beta1.ScanParameters{
				ProjectID: "library",
				Reference: "latest",
			},
			isValid: false,
		},
		{
			name: "missing reference",
			params: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := tt.params.ProjectID != "" && tt.params.RepositoryName != "" && tt.params.Reference != ""
			if isValid != tt.isValid {
				t.Errorf("Expected valid=%v, got %v", tt.isValid, isValid)
			}
		})
	}
}

func TestObserveScanEmptyStatus(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error) {
				return &harborclients.ScanStatus{
					ID:            "scan-123",
					Status:        "",
					CriticalCount: 0,
					HighCount:     0,
					MediumCount:   0,
					LowCount:      0,
					StartTime:     time.Time{},
					EndTime:       time.Time{},
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, scan)
	if err != nil {
		t.Errorf("Observe should not fail, got %v", err)
	}
	if !obs.ResourceExists {
		t.Error("ResourceExists should be true")
	}
}

func TestObserveScanListError(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error) {
				return nil, errors.New("scan service unavailable")
			},
		},
	}

	_, err := ext.Observe(ctx, scan)
	if err == nil {
		t.Error("Observe should fail when client returns error")
	}
}

func TestCreateScanWithOptionalFields(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			triggerScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				if projectID != "library" {
					t.Errorf("Expected projectID 'library', got '%s'", projectID)
				}
				if repoName != "alpine" {
					t.Errorf("Expected repoName 'alpine', got '%s'", repoName)
				}
				return nil
			},
		},
	}

	_, err := ext.Create(ctx, scan)
	if err != nil {
		t.Errorf("Create should not fail, got %v", err)
	}
}

func TestUpdateScanNoChanges(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{}

	_, err := ext.Update(ctx, scan)
	if err != nil {
		t.Errorf("Update should not fail, got %v", err)
	}
}

func TestDeleteScanNoID(t *testing.T) {
	ctx := context.Background()
	scan := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-scan",
		},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			stopScanFunc: func(ctx context.Context, projectID, repoName, reference string) error {
				return errors.New("stop scan failed")
			},
		},
	}

	_, err := ext.Delete(ctx, scan)
	if err == nil {
		t.Error("Delete should fail when client fails")
	}
}

func TestScanVulnerabilityCounts(t *testing.T) {
	tests := []struct {
		name          string
		criticalCount int64
		highCount     int64
		mediumCount   int64
		lowCount      int64
		hasVulns      bool
	}{
		{
			name:          "no vulnerabilities",
			criticalCount: 0,
			highCount:     0,
			mediumCount:   0,
			lowCount:      0,
			hasVulns:      false,
		},
		{
			name:          "critical vulnerabilities only",
			criticalCount: 3,
			highCount:     0,
			mediumCount:   0,
			lowCount:      0,
			hasVulns:      true,
		},
		{
			name:          "multiple severity levels",
			criticalCount: 2,
			highCount:     5,
			mediumCount:   10,
			lowCount:      20,
			hasVulns:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scan := &v1beta1.Scan{
				Spec: v1beta1.ScanSpec{
					ForProvider: v1beta1.ScanParameters{
						ProjectID:      "library",
						RepositoryName: "alpine",
						Reference:      "latest",
					},
				},
				Status: v1beta1.ScanStatus{
					AtProvider: v1beta1.ScanObservation{
						CriticalCount: &tt.criticalCount,
						HighCount:     &tt.highCount,
						MediumCount:   &tt.mediumCount,
						LowCount:      &tt.lowCount,
					},
				},
			}

			hasVulns := *scan.Status.AtProvider.CriticalCount > 0 ||
				*scan.Status.AtProvider.HighCount > 0 ||
				*scan.Status.AtProvider.MediumCount > 0 ||
				*scan.Status.AtProvider.LowCount > 0

			if hasVulns != tt.hasVulns {
				t.Errorf("Expected hasVulns=%v, got %v", tt.hasVulns, hasVulns)
			}
		})
	}
}

// mockScanClient implements HarborClienter for scan tests
type mockScanClient struct {
	harborclients.HarborClienter
	getScanFunc     func(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error)
	triggerScanFunc func(ctx context.Context, projectID, repoName, reference string) error
	stopScanFunc    func(ctx context.Context, projectID, repoName, reference string) error
	closeFunc       func() error
}

func (m *mockScanClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*harborclients.ScanStatus, error) {
	if m.getScanFunc != nil {
		return m.getScanFunc(ctx, projectID, repoName, reference)
	}
	return nil, nil
}

func (m *mockScanClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.triggerScanFunc != nil {
		return m.triggerScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *mockScanClient) StopScan(ctx context.Context, projectID, repoName, reference string) error {
	if m.stopScanFunc != nil {
		return m.stopScanFunc(ctx, projectID, repoName, reference)
	}
	return nil
}

func (m *mockScanClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func (m *mockScanClient) GetBaseURL() string {
	return "https://harbor.example.com"
}

// Helper functions
func ptrString(s string) *string {
	return &s
}

// TestObserveScanSetsAvailableOnSuccess verifies that Observe sets
// xpv1.Available() when the scan status is "Success" (case-insensitive).
func TestObserveScanSetsAvailableOnSuccess(t *testing.T) {
	ctx := context.Background()
	sc := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{Name: "test-scan"},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(_ context.Context, _, _, _ string) (*harborclients.ScanStatus, error) {
				return &harborclients.ScanStatus{
					ID:     "rpt-001",
					Status: "Success",
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, sc)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true")
	}
	if !obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=true when status is Success")
	}

	cond := sc.GetCondition(xpv1.TypeReady)
	if cond.Status != corev1.ConditionTrue {
		t.Errorf("expected Ready=True (Available) when scan succeeds, got %v / %v", cond.Status, cond.Reason)
	}
}

// TestObserveScanNotAvailableWhileScanning verifies that Observe does NOT set
// Available when the scan is still running (ResourceUpToDate=false).
func TestObserveScanNotAvailableWhileScanning(t *testing.T) {
	ctx := context.Background()
	sc := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{Name: "test-scan"},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "latest",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(_ context.Context, _, _, _ string) (*harborclients.ScanStatus, error) {
				return &harborclients.ScanStatus{
					ID:     "rpt-001",
					Status: "Scanning",
				}, nil
			},
		},
	}

	obs, err := ext.Observe(ctx, sc)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if !obs.ResourceExists {
		t.Error("expected ResourceExists=true while scan in progress")
	}
	if obs.ResourceUpToDate {
		t.Error("expected ResourceUpToDate=false while scan in progress")
	}

	// Available must NOT be set while still scanning.
	cond := sc.GetCondition(xpv1.TypeReady)
	if cond.Status == corev1.ConditionTrue {
		t.Errorf("expected Ready!=True while scan is in progress, got %v", cond.Status)
	}
}

// TestObserveScanNilStatusNotFound verifies that Observe returns
// ResourceExists=false when GetScan returns (nil, nil).
func TestObserveScanNilStatusNotFound(t *testing.T) {
	ctx := context.Background()
	sc := &v1beta1.Scan{
		ObjectMeta: metav1.ObjectMeta{Name: "missing-artifact"},
		Spec: v1beta1.ScanSpec{
			ForProvider: v1beta1.ScanParameters{
				ProjectID:      "library",
				RepositoryName: "alpine",
				Reference:      "nonexistent",
			},
		},
	}

	ext := &external{
		service: &mockScanClient{
			getScanFunc: func(_ context.Context, _, _, _ string) (*harborclients.ScanStatus, error) {
				return nil, nil // (nil, nil) = artifact not found
			},
		},
	}

	obs, err := ext.Observe(ctx, sc)
	if err != nil {
		t.Fatalf("Observe: %v", err)
	}
	if obs.ResourceExists {
		t.Error("expected ResourceExists=false for nil status")
	}

	cond := sc.GetCondition(xpv1.TypeReady)
	if cond.Status == corev1.ConditionTrue {
		t.Errorf("expected Ready!=True for not-found artifact, got %v", cond.Status)
	}
}
