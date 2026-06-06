/*
Copyright 2024 Crossplane Harbor Provider.
*/

package scan

import (
	"context"
	"testing"
)

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
