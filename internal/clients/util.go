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
	"net/http"
	"strconv"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/pkg/errors"
	"k8s.io/utils/ptr"
)

// isHarborCode reports whether err represents a given HTTP status from Harbor.
// Harbor's swagger is inconsistent: some operations carry a typed response (which
// implements IsCode), while others surface a generic *runtime.APIError. This
// detects both.
func isHarborCode(err error, code int) bool {
	if err == nil {
		return false
	}
	var apiErr *runtime.APIError
	if errors.As(err, &apiErr) && apiErr.Code == code {
		return true
	}
	var coder interface{ IsCode(int) bool }
	if errors.As(err, &coder) {
		return coder.IsCode(code)
	}
	return false
}

// isHarborNotFound maps a Harbor 404 to the (nil, nil) not-found contract.
func isHarborNotFound(err error) bool {
	return isHarborCode(err, http.StatusNotFound)
}

// idFromLocation extracts the trailing numeric ID from a Harbor Location header
// (e.g. "/api/v2.0/replication/policies/42" -> 42).
func idFromLocation(location string) (int64, error) {
	parts := strings.Split(strings.TrimRight(location, "/"), "/")
	if len(parts) == 0 {
		return 0, errors.New("empty location")
	}
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

// projectRef returns the project_name_or_id path value plus the X-Is-Resource-Name
// header pointer: Harbor needs the header set when a project is addressed by name
// rather than numeric ID. A nil header means "addressed by numeric ID".
func projectRef(projectID string) (string, *bool) {
	if _, err := strconv.ParseInt(projectID, 10, 64); err == nil {
		return projectID, nil
	}
	return projectID, ptr.To(true)
}

// resolveProjectID returns the numeric Harbor project id for a project reference
// that may be either a numeric id or a project name.
func (c *HarborClient) resolveProjectID(ctx context.Context, ref string) (int64, error) {
	if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
		return id, nil
	}
	st, err := c.GetProject(ctx, ref)
	if err != nil {
		return 0, err
	}
	if st == nil {
		return 0, errors.Errorf("project %q not found", ref)
	}
	id, err := strconv.ParseInt(st.ID, 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "project %q has non-numeric id %q", ref, st.ID)
	}
	return id, nil
}

// resolveProjectName returns the Harbor project NAME for a project reference that
// may be either a numeric id or a name. Harbor endpoints addressed by the bare
// project_name path segment (repositories, artifacts, scans) and the robot
// permission namespace need the name, not the id; callers pass the contractual
// numeric projectId and this resolves it via GET /projects/{id}. A non-numeric
// ref is treated as already being a name and returned verbatim (backward compat
// for name-based callers).
func (c *HarborClient) resolveProjectName(ctx context.Context, ref string) (string, error) {
	if _, err := strconv.ParseInt(ref, 10, 64); err != nil {
		return ref, nil
	}
	st, err := c.GetProject(ctx, ref)
	if err != nil {
		return "", err
	}
	if st == nil {
		return "", errors.Errorf("project %q not found", ref)
	}
	return st.Name, nil
}
