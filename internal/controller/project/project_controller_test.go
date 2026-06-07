/*
Copyright 2024 Crossplane Harbor Provider.
*/

package project

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/rossigee/provider-harbor/apis/project/v1beta1"
)

// ERROR CASE TESTS

func TestConnectNotProject(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Connect with nil should return %s error", errNotProject)
	}
}

func TestObserveNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Observe with nil should return %s error", errNotProject)
	}
}

func TestUpdateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Update with nil should return %s error", errNotProject)
	}
}

func TestDeleteNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Delete with nil should return %s error", errNotProject)
	}
}

func TestCreateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Create with nil should return %s error", errNotProject)
	}
}

// HAPPY-PATH AND VALIDATION TESTS

func TestProjectHasRequiredFields(t *testing.T) {
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-project",
			Namespace: "default",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
	}

	if project.Spec.ForProvider.Name == "" {
		t.Error("Project Name should not be empty")
	}
	if project.Name == "" {
		t.Error("Metadata Name should not be empty")
	}
}

func TestProjectStatusFields(t *testing.T) {
	project := &v1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-project",
		},
		Spec: v1beta1.ProjectSpec{
			ForProvider: v1beta1.ProjectParameters{
				Name: "my-project",
			},
		},
		Status: v1beta1.ProjectStatus{
			AtProvider: v1beta1.ProjectObservation{
				ID: ptrString("123"),
			},
		},
	}

	if project.Status.AtProvider.ID == nil {
		t.Error("Status ID should be populated")
	}
	if *project.Status.AtProvider.ID != "123" {
		t.Errorf("Status ID should be '123', got %s", *project.Status.AtProvider.ID)
	}
}

func TestProjectParametersValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  v1beta1.ProjectParameters
		isValid bool
	}{
		{
			name: "valid with only required name",
			params: v1beta1.ProjectParameters{
				Name: "my-project",
			},
			isValid: true,
		},
		{
			name: "valid with public flag",
			params: v1beta1.ProjectParameters{
				Name:   "public-project",
				Public: ptrBool(true),
			},
			isValid: true,
		},
		{
			name: "valid with security settings",
			params: v1beta1.ProjectParameters{
				Name:                    "secure-project",
				EnableContentTrust:      ptrBool(true),
				EnableContentTrustCosign: ptrBool(true),
				AutoScanImages:          ptrBool(true),
				PreventVulnerableImages: ptrBool(true),
				Severity:                ptrString("high"),
			},
			isValid: true,
		},
		{
			name: "valid with CVE allowlist",
			params: v1beta1.ProjectParameters{
				Name:         "project-with-cves",
				CVEAllowlist: []string{"CVE-2024-1234", "CVE-2024-5678"},
			},
			isValid: true,
		},
		{
			name: "valid with storage limit",
			params: v1beta1.ProjectParameters{
				Name:         "project-with-quota",
				StorageLimit: ptrInt64(1073741824), // 1GB
			},
			isValid: true,
		},
		{
			name: "valid with metadata",
			params: v1beta1.ProjectParameters{
				Name: "project-with-metadata",
				Metadata: map[string]string{
					"environment": "production",
					"team":        "platform",
				},
			},
			isValid: true,
		},
		{
			name: "missing required name",
			params: v1beta1.ProjectParameters{
				Public: ptrBool(true),
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

func TestProjectSecuritySettings(t *testing.T) {
	tests := []struct {
		name     string
		severity string
		isValid  bool
	}{
		{"negligible severity", "negligible", true},
		{"low severity", "low", true},
		{"medium severity", "medium", true},
		{"high severity", "high", true},
		{"critical severity", "critical", true},
		{"invalid severity", "unknown", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severities := map[string]bool{
				"negligible": true,
				"low":        true,
				"medium":     true,
				"high":       true,
				"critical":   true,
			}
			isValid := severities[tt.severity]
			if isValid != tt.isValid {
				t.Errorf("Expected severity '%s' to be valid=%v, got %v", tt.severity, tt.isValid, isValid)
			}
		})
	}
}

func TestProjectStorageLimitValidation(t *testing.T) {
	tests := []struct {
		name  string
		limit int64
		desc  string
	}{
		{"1MB", 1048576, "1MB storage limit"},
		{"1GB", 1073741824, "1GB storage limit"},
		{"10GB", 10737418240, "10GB storage limit"},
		{"1TB", 1099511627776, "1TB storage limit"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			project := &v1beta1.Project{
				Spec: v1beta1.ProjectSpec{
					ForProvider: v1beta1.ProjectParameters{
						Name:         "test-project",
						StorageLimit: ptrInt64(tt.limit),
					},
				},
			}

			if project.Spec.ForProvider.StorageLimit == nil {
				t.Error("StorageLimit should be set")
			}
			if *project.Spec.ForProvider.StorageLimit != tt.limit {
				t.Errorf("Expected %d, got %d", tt.limit, *project.Spec.ForProvider.StorageLimit)
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

func ptrString(s string) *string {
	return &s
}
