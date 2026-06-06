/*
Copyright 2024 Crossplane Harbor Provider.
*/

package webhook

import (
	"context"
	"testing"
)

func TestConnectNotWebhook(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotWebhook {
		t.Errorf("Connect with nil should return %s error", errNotWebhook)
	}
}

func TestObserveNotWebhook(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotWebhook {
		t.Errorf("Observe with nil should return %s error", errNotWebhook)
	}
}

func TestUpdateNotWebhook(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotWebhook {
		t.Errorf("Update with nil should return %s error", errNotWebhook)
	}
}

func TestDeleteNotWebhook(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotWebhook {
		t.Errorf("Delete with nil should return %s error", errNotWebhook)
	}
}

func TestCreateNotWebhook(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotWebhook {
		t.Errorf("Create with nil should return %s error", errNotWebhook)
	}
}
