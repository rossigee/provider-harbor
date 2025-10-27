//go:build integration

/*
Copyright 2025 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package clients

import (
	"context"
	"testing"
	"time"
)

func TestNewHarborClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *HarborConfig
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: &HarborConfig{
				URL:      "https://harbor.example.com",
				Username: "admin",
				Password: "Harbor12345",
				Insecure: false,
			},
			expectError: false,
		},
		{
			name: "valid config with insecure",
			config: &HarborConfig{
				URL:      "https://harbor-dev.example.com",
				Username: "admin",
				Password: "Harbor12345",
				Insecure: true,
			},
			expectError: false,
		},
		{
			name:        "nil config",
			config:      nil,
			expectError: true,
			errorMsg:    "config is required",
		},
		{
			name: "missing URL",
			config: &HarborConfig{
				Username: "admin",
				Password: "Harbor12345",
			},
			expectError: true,
			errorMsg:    "harbor URL is required",
		},
		{
			name: "missing username",
			config: &HarborConfig{
				URL:      "https://harbor.example.com",
				Password: "Harbor12345",
			},
			expectError: true,
			errorMsg:    "username is required",
		},
		{
			name: "missing password",
			config: &HarborConfig{
				URL:      "https://harbor.example.com",
				Username: "admin",
			},
			expectError: true,
			errorMsg:    "password is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewHarborClient(tt.config)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
					return
				}
				if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
				if client != nil {
					t.Error("expected nil client when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
					return
				}
				if client == nil {
					t.Error("expected valid client but got nil")
					return
				}

				// Test client properties
				if client.GetBaseURL() != tt.config.URL {
					t.Errorf("expected URL %s, got %s", tt.config.URL, client.GetBaseURL())
				}

				// Test memory footprint info
				footprint := client.GetMemoryFootprint()
				if footprint == "" {
					t.Error("expected memory footprint info")
				} else {
					t.Logf("Memory footprint: %s", footprint)
				}

				// Test cleanup
				err = client.Close()
				if err != nil {
					t.Errorf("failed to close client: %v", err)
				}
			}
		})
	}
}

func TestHarborClientOperations(t *testing.T) {
	config := &HarborConfig{
		URL:      "https://harbor.example.com",
		Username: "admin",
		Password: "Harbor12345",
		Insecure: false,
	}

	client, err := NewHarborClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("error closing client: %v", err)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("TestConnection", func(t *testing.T) {
		err := client.TestConnection(ctx)
		if err != nil {
			t.Errorf("connection test failed: %v", err)
		} else {
			t.Log("Connection test passed")
		}
	})

	t.Run("GetVersion", func(t *testing.T) {
		version, err := client.GetVersion(ctx)
		if err != nil {
			t.Errorf("failed to get version: %v", err)
		} else {
			t.Logf("Harbor version: %s", version)
		}
	})

	t.Run("ListProjects", func(t *testing.T) {
		projects, err := client.ListProjects(ctx)
		if err != nil {
			t.Errorf("failed to list projects: %v", err)
			return
		}

		t.Logf("Found %d projects", len(projects))
		for _, project := range projects {
			t.Logf("Project: Name=%s, Public=%t, Created=%s",
				project.Name, project.Public, project.CreatedAt.Format(time.RFC3339))
		}
	})

	testProjectName := "crossplane-test-project"

	t.Run("CreateProject", func(t *testing.T) {
		spec := &ProjectSpec{
			Name:   testProjectName,
			Public: false,
		}

		status, err := client.CreateProject(ctx, spec)
		if err != nil {
			t.Errorf("failed to create project: %v", err)
			return
		}

		if status.Name != testProjectName {
			t.Errorf("expected project name %s, got %s", testProjectName, status.Name)
		}
		if status.Public != spec.Public {
			t.Errorf("expected public %t, got %t", spec.Public, status.Public)
		}

		t.Logf("Created project: Name=%s, Public=%t, Created=%s",
			status.Name, status.Public, status.CreatedAt.Format(time.RFC3339))
	})

	t.Run("GetProject", func(t *testing.T) {
		status, err := client.GetProject(ctx, testProjectName)
		if err != nil {
			t.Errorf("failed to get project: %v", err)
			return
		}

		if status.Name != testProjectName {
			t.Errorf("expected project name %s, got %s", testProjectName, status.Name)
		}

		t.Logf("Retrieved project: Name=%s, Public=%t, Created=%s",
			status.Name, status.Public, status.CreatedAt.Format(time.RFC3339))
	})

	t.Run("UpdateProject", func(t *testing.T) {
		spec := &ProjectSpec{
			Name:   testProjectName,
			Public: true, // Make it public
		}

		status, err := client.UpdateProject(ctx, testProjectName, spec)
		if err != nil {
			t.Errorf("failed to update project: %v", err)
			return
		}

		if status.Public != true {
			t.Errorf("expected project to be public after update")
		}

		t.Logf("Updated project: Name=%s, Public=%t, Created=%s",
			status.Name, status.Public, status.CreatedAt.Format(time.RFC3339))
	})

	t.Run("DeleteProject", func(t *testing.T) {
		err := client.DeleteProject(ctx, testProjectName)
		if err != nil {
			t.Errorf("failed to delete project: %v", err)
			return
		}

		t.Logf("Successfully deleted project %s", testProjectName)
	})
}

func TestHarborClientValidation(t *testing.T) {
	config := &HarborConfig{
		URL:      "https://harbor.example.com",
		Username: "admin",
		Password: "Harbor12345",
		Insecure: false,
	}

	client, err := NewHarborClient(config)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	defer func() {
		if err := client.Close(); err != nil {
			t.Logf("error closing client: %v", err)
		}
	}()

	ctx := context.Background()

	t.Run("CreateProjectValidation", func(t *testing.T) {
		tests := []struct {
			name        string
			spec        *ProjectSpec
			expectError bool
			errorMsg    string
		}{
			{
				name:        "nil spec",
				spec:        nil,
				expectError: true,
				errorMsg:    "project spec is required",
			},
			{
				name: "empty name",
				spec: &ProjectSpec{
					Name: "",
				},
				expectError: true,
				errorMsg:    "project name is required",
			},
			{
				name: "valid spec",
				spec: &ProjectSpec{
					Name:   "valid-project",
					Public: true,
				},
				expectError: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				_, err := client.CreateProject(ctx, tt.spec)

				if tt.expectError {
					if err == nil {
						t.Error("expected error but got none")
						return
					}
					if tt.errorMsg != "" && err.Error() != tt.errorMsg {
						t.Errorf("expected error message '%s', got '%s'", tt.errorMsg, err.Error())
					}
				} else {
					if err != nil {
						t.Errorf("unexpected error: %v", err)
					}
				}
			})
		}
	})

	t.Run("GetProjectValidation", func(t *testing.T) {
		_, err := client.GetProject(ctx, "")
		if err == nil {
			t.Error("expected error for empty project name")
		}

		expectedMsg := "project name is required"
		if err.Error() != expectedMsg {
			t.Errorf("expected error message '%s', got '%s'", expectedMsg, err.Error())
		}
	})
}
