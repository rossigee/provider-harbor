/*
Copyright 2024 Crossplane Harbor Provider.
*/

package registry

import (
	"context"
	"testing"
)

func TestConnectNotRegistry(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Connect with nil should return %s error", errNotRegistry)
	}
}

func TestObserveNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Observe with nil should return %s error", errNotRegistry)
	}
}

func TestUpdateNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Update with nil should return %s error", errNotRegistry)
	}
}

func TestDeleteNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Delete with nil should return %s error", errNotRegistry)
	}
}

func TestCreateNotRegistry(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRegistry {
		t.Errorf("Create with nil should return %s error", errNotRegistry)
	}
}
