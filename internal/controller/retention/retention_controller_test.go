/*
Copyright 2024 Crossplane Harbor Provider.
*/

package retention

import (
	"context"
	"testing"
)

func TestConnectNotRetention(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Connect with nil should return %s error", errNotRetention)
	}
}

func TestObserveNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Observe with nil should return %s error", errNotRetention)
	}
}

func TestUpdateNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Update with nil should return %s error", errNotRetention)
	}
}

func TestDeleteNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Delete with nil should return %s error", errNotRetention)
	}
}

func TestCreateNotRetention(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRetention {
		t.Errorf("Create with nil should return %s error", errNotRetention)
	}
}
