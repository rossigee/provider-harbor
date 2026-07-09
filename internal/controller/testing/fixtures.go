/*
Copyright 2024 Crossplane Harbor Provider.
*/

package testing

import (
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	projectv1beta1 "github.com/rossigee/provider-harbor/apis/project/v1beta1"
	robotv1beta1 "github.com/rossigee/provider-harbor/apis/robot/v1beta1"
	"github.com/rossigee/provider-harbor/apis/user/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewTestUser creates a test User resource
func NewTestUser(name string) *v1beta1.User {
	return &v1beta1.User{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1beta1.UserSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Name: "default",
				},
			},
			ForProvider: v1beta1.UserParameters{
				Username: "testuser",
				Email:    "test@example.com",
			},
		},
	}
}

// NewTestProject creates a test Project resource
func NewTestProject(name string) *projectv1beta1.Project {
	return &projectv1beta1.Project{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: projectv1beta1.ProjectSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Name: "default",
				},
			},
			ForProvider: projectv1beta1.ProjectParameters{
				Name: "test-project",
			},
		},
	}
}

// NewTestRobot creates a test Robot resource
func NewTestRobot(name string) *robotv1beta1.Robot {
	return &robotv1beta1.Robot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: robotv1beta1.RobotSpec{
			ManagedResourceSpec: xpv1.ManagedResourceSpec{
				ProviderConfigReference: &xpv1.ProviderConfigReference{
					Name: "default",
				},
			},
			ForProvider: robotv1beta1.RobotParameters{
				Name: "test-robot",
			},
		},
	}
}
