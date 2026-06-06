/*
Copyright 2024 Crossplane Harbor Provider.
*/

package robot

import (
	"context"
	"testing"
)

func TestConnectNotRobot(t *testing.T) {
	ctx := context.Background()
	conn := &connector{}

	_, err := conn.Connect(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Connect with nil should return %s error", errNotRobot)
	}
}

func TestObserveNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Observe(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Observe with nil should return %s error", errNotRobot)
	}
}

func TestUpdateNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Update(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Update with nil should return %s error", errNotRobot)
	}
}

func TestDeleteNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Delete(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Delete with nil should return %s error", errNotRobot)
	}
}

func TestCreateNotRobot(t *testing.T) {
	ctx := context.Background()
	ext := &external{}

	_, err := ext.Create(ctx, nil)
	if err == nil || err.Error() != errNotRobot {
		t.Errorf("Create with nil should return %s error", errNotRobot)
	}
}

