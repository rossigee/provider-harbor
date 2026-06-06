/*
Copyright 2024 Crossplane Harbor Provider.
*/

package artifact

import (
	"context"
	"testing"
)

func TestConnectNotArtifact(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotArtifact {
		t.Errorf("Connect with nil should return %s error", errNotArtifact)
	}
}

func TestObserveNotArtifact(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotArtifact {
		t.Errorf("Observe with nil should return %s error", errNotArtifact)
	}
}

func TestUpdateNotArtifact(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err != nil {
		t.Errorf("Update with nil should return nil error, got %v", err)
	}
}

func TestDeleteNotArtifact(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotArtifact {
		t.Errorf("Delete with nil should return %s error", errNotArtifact)
	}
}

func TestCreateNotArtifact(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotArtifact {
		t.Errorf("Create with nil should return %s error", errNotArtifact)
	}
}

