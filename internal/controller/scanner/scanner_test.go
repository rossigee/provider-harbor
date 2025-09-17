/*
Copyright 2022 Upbound Inc.
*/

package scanner

import (
	"testing"

	v1alpha1 "github.com/rossigee/provider-harbor/apis/scanner/v1alpha1"
)

func TestScannerRegistrationGroupVersionKind(t *testing.T) {
	expected := "scanner.harbor.crossplane.io/v1alpha1, Kind=ScannerRegistration"
	actual := v1alpha1.ScannerRegistration_GroupVersionKind.String()

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}

func TestScannerRegistrationKind(t *testing.T) {
	expected := "ScannerRegistration"
	actual := v1alpha1.ScannerRegistration_Kind

	if actual != expected {
		t.Errorf("Expected %s, got %s", expected, actual)
	}
}