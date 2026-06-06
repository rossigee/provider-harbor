/*
Copyright 2024 Crossplane Harbor Provider.
*/

package member

import (
	"context"
	"testing"
)

func TestConnectNotMember(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Connect with nil should return %s error", errNotMember)
	}
}

func TestObserveNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Observe with nil should return %s error", errNotMember)
	}
}

func TestUpdateNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Update with nil should return %s error", errNotMember)
	}
}

func TestDeleteNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Delete with nil should return %s error", errNotMember)
	}
}

func TestCreateNotMember(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotMember {
		t.Errorf("Create with nil should return %s error", errNotMember)
	}
}

