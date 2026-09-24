/*
Copyright 2025 Crossplane Harbor Provider.
*/

package systeminfo

import (
	"context"
	"testing"

	sdkmodels "github.com/goharbor/go-client/pkg/sdk/v2.0/models"
	"github.com/pkg/errors"
	systeminfoav1beta1 "github.com/rossigee/provider-harbor/apis/systeminfo/v1beta1"
	harborclients "github.com/rossigee/provider-harbor/internal/clients"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSystemInfoObserve(t *testing.T) {
	tests := []struct {
		name                 string
		systemInfo           *systeminfoav1beta1.SystemInfo
		harborInfo           *sdkmodels.GeneralInfo
		harborError          error
		expectError          bool
		expectErr            string
		expectResourceExists bool
		expectedVersion      string
	}{
		{
			name: "observe system info successfully",
			systemInfo: &systeminfoav1beta1.SystemInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name: "harbor-system",
				},
				Spec:   systeminfoav1beta1.SystemInfoSpec{},
				Status: systeminfoav1beta1.SystemInfoStatus{},
			},
			harborInfo: &sdkmodels.GeneralInfo{
				HarborVersion: ptrSystemInfoString("v2.8.0"),
			},
			expectResourceExists: true,
			expectedVersion:      "v2.8.0",
		},
		{
			name: "system info fetch error",
			systemInfo: &systeminfoav1beta1.SystemInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name: "harbor-system",
				},
				Spec:   systeminfoav1beta1.SystemInfoSpec{},
				Status: systeminfoav1beta1.SystemInfoStatus{},
			},
			harborError: errors.New("connection error"),
			expectError: true,
			expectErr:   "failed to get system info",
		},
		{
			name: "system info with empty version",
			systemInfo: &systeminfoav1beta1.SystemInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name: "harbor-system",
				},
				Spec:   systeminfoav1beta1.SystemInfoSpec{},
				Status: systeminfoav1beta1.SystemInfoStatus{},
			},
			harborInfo: &sdkmodels.GeneralInfo{
				HarborVersion: ptrSystemInfoString(""),
			},
			expectResourceExists: true,
			expectedVersion:      "",
		},
		{
			name: "system info with nil version",
			systemInfo: &systeminfoav1beta1.SystemInfo{
				ObjectMeta: metav1.ObjectMeta{
					Name: "harbor-system",
				},
				Spec:   systeminfoav1beta1.SystemInfoSpec{},
				Status: systeminfoav1beta1.SystemInfoStatus{},
			},
			harborInfo: &sdkmodels.GeneralInfo{
				HarborVersion: nil,
			},
			expectResourceExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &harborclients.MockHarborClient{
				GetSystemInfoFunc: func(ctx context.Context) (*sdkmodels.GeneralInfo, error) {
					if tt.harborError != nil {
						return nil, tt.harborError
					}
					return tt.harborInfo, nil
				},
			}

			ext := &external{
				service: mockClient,
			}

			obs, err := ext.Observe(context.Background(), tt.systemInfo)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				if tt.expectErr != "" && !contains(err.Error(), tt.expectErr) {
					t.Errorf("expected error containing %q, got %q", tt.expectErr, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if obs.ResourceExists != tt.expectResourceExists {
				t.Errorf("expected ResourceExists %v, got %v", tt.expectResourceExists, obs.ResourceExists)
			}

			if tt.expectedVersion != "" && (tt.systemInfo.Status.AtProvider.HarborVersion == nil || *tt.systemInfo.Status.AtProvider.HarborVersion != tt.expectedVersion) {
				t.Errorf("expected version %q, got %v", tt.expectedVersion, tt.systemInfo.Status.AtProvider.HarborVersion)
			}
		})
	}
}

func TestSystemInfoCreateReadOnlyError(t *testing.T) {
	ext := &external{
		service: nil,
	}

	_, err := ext.Create(context.Background(), &systeminfoav1beta1.SystemInfo{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

func TestSystemInfoUpdateReadOnlyError(t *testing.T) {
	ext := &external{
		service: nil,
	}

	_, err := ext.Update(context.Background(), &systeminfoav1beta1.SystemInfo{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

func TestSystemInfoDeleteReadOnlyError(t *testing.T) {
	ext := &external{
		service: nil,
	}

	_, err := ext.Delete(context.Background(), &systeminfoav1beta1.SystemInfo{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

func TestSystemInfoDisconnect(t *testing.T) {
	ext := &external{
		service: nil,
	}

	err := ext.Disconnect(context.Background())
	if err != nil {
		t.Errorf("unexpected error on disconnect: %v", err)
	}
}

// Helper functions
func ptrSystemInfoString(s string) *string {
	return &s
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > len(substr) && len(s) > 0 && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
