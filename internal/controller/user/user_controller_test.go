/*
Copyright 2024 Crossplane Harbor Provider.
*/

package user

import (
	"context"
	"testing"
)

func TestConnectNotUser(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Connect with nil should return %s error", errNotUser)
	}
}

func TestObserveNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Observe with nil should return %s error", errNotUser)
	}
}

func TestUpdateNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Update with nil should return %s error", errNotUser)
	}
}

func TestDeleteNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Delete with nil should return %s error", errNotUser)
	}
}

func TestCreateNotUser(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotUser {
		t.Errorf("Create with nil should return %s error", errNotUser)
	}
}
