/*
Copyright 2024 Crossplane Harbor Provider.
*/

package project

import (
	"context"
	"testing"
)

func TestConnectNotProject(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Connect with nil should return %s error", errNotProject)
	}
}

func TestObserveNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Observe with nil should return %s error", errNotProject)
	}
}

func TestUpdateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Update with nil should return %s error", errNotProject)
	}
}

func TestDeleteNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Delete with nil should return %s error", errNotProject)
	}
}

func TestCreateNotProject(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotProject {
		t.Errorf("Create with nil should return %s error", errNotProject)
	}
}
