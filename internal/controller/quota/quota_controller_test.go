/*
Copyright 2025 Crossplane Harbor Provider.
*/

package quota

import (
	"context"
	"testing"

	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	quotav1beta1 "github.com/rossigee/provider-harbor/apis/quota/v1beta1"
)

func TestResolveProjectRefEmptyName(t *testing.T) {
	ext := &external{
		kube:    nil,
		service: nil,
	}

	_, err := ext.resolveProjectRef(context.Background(), xpv1.Reference{Name: ""})

	if err == nil {
		t.Errorf("expected error for empty project reference name")
	}
	if !contains(err.Error(), "project reference is required") {
		t.Errorf("expected 'project reference is required' error, got %q", err.Error())
	}
}

func TestCreateReadOnlyError(t *testing.T) {
	ext := &external{
		kube:    nil,
		service: nil,
	}

	_, err := ext.Create(context.Background(), &quotav1beta1.Quota{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

func TestUpdateReadOnlyError(t *testing.T) {
	ext := &external{
		kube:    nil,
		service: nil,
	}

	_, err := ext.Update(context.Background(), &quotav1beta1.Quota{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

func TestDeleteReadOnlyError(t *testing.T) {
	ext := &external{
		kube:    nil,
		service: nil,
	}

	_, err := ext.Delete(context.Background(), &quotav1beta1.Quota{})
	if err == nil {
		t.Errorf("expected error for read-only resource")
	}
	if !contains(err.Error(), "read-only") {
		t.Errorf("expected read-only error, got %q", err.Error())
	}
}

// Helper functions
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
