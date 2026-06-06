/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scanner

import (
	"context"
	"testing"
)

func TestConnectNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotScannerRegistration {
		t.Errorf("Connect with nil should return %s error", errNotScannerRegistration)
	}
}

func TestObserveNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotScannerRegistration {
		t.Errorf("Observe with nil should return %s error", errNotScannerRegistration)
	}
}

func TestUpdateNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotScannerRegistration {
		t.Errorf("Update with nil should return %s error", errNotScannerRegistration)
	}
}

func TestDeleteNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotScannerRegistration {
		t.Errorf("Delete with nil should return %s error", errNotScannerRegistration)
	}
}

func TestCreateNotScannerRegistration(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotScannerRegistration {
		t.Errorf("Create with nil should return %s error", errNotScannerRegistration)
	}
}
