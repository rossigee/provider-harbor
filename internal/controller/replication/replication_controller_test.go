/*
Copyright 2024 Crossplane Harbor Provider.
*/

package replication

import (
	"context"
	"testing"
)

func TestConnectNotReplication(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Connect with nil should return %s error", errNotReplication)
	}
}

func TestObserveNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Observe with nil should return %s error", errNotReplication)
	}
}

func TestUpdateNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Update with nil should return %s error", errNotReplication)
	}
}

func TestDeleteNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Delete with nil should return %s error", errNotReplication)
	}
}

func TestCreateNotReplication(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotReplication {
		t.Errorf("Create with nil should return %s error", errNotReplication)
	}
}
