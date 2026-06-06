/*
Copyright 2024 Crossplane Harbor Provider.
*/

package repository

import (
	"context"
	"testing"
)

func TestConnectNotRepository(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Connect with nil should return %s error", errNotRepository)
	}
}

func TestObserveNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Observe with nil should return %s error", errNotRepository)
	}
}

func TestUpdateNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Update with nil should return %s error", errNotRepository)
	}
}

func TestDeleteNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Delete with nil should return %s error", errNotRepository)
	}
}

func TestCreateNotRepository(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRepository {
		t.Errorf("Create with nil should return %s error", errNotRepository)
	}
}
