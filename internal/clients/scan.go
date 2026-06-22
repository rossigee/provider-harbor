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
	"time"

	harborartifact "github.com/goharbor/go-client/pkg/sdk/v2.0/client/artifact"
	harborscan "github.com/goharbor/go-client/pkg/sdk/v2.0/client/scan"
	"github.com/pkg/errors"
)

// ScanStatus represents the status of an artifact scan
type ScanStatus struct {
	ID            string
	Status        string
	CriticalCount int64
	HighCount     int64
	MediumCount   int64
	LowCount      int64
	StartTime     time.Time
	EndTime       time.Time
}

// TriggerScan triggers a vulnerability scan on the specified artifact.
// Harbor returns 202 Accepted; the scan runs asynchronously. Use GetScan to
// poll the result via the artifact's scan_overview.
func (c *HarborClient) TriggerScan(ctx context.Context, projectID, repoName, reference string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if repoName == "" {
		return errors.New("repository name is required")
	}
	if reference == "" {
		return errors.New("reference is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Triggering Harbor artifact scan", "projectId", projectID, "repo", repoName, "reference", reference)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return errors.Wrap(err, "cannot resolve project for scan")
	}

	params := harborscan.NewScanArtifactParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithReference(reference)
	if _, err := v2Client.Scan.ScanArtifact(ctx, params); err != nil {
		return errors.Wrap(err, "cannot trigger Harbor artifact scan")
	}
	return nil
}

// ListScans lists scan results for all artifacts in a repository. Each artifact's
// scan status is sourced from its scan_overview (first MIME-type entry).
// Artifacts with no scan data produce a ScanStatus with Status="".
func (c *HarborClient) ListScans(ctx context.Context, projectID, repoName string) ([]*ScanStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Listing Harbor artifact scans", "projectId", projectID, "repo", repoName)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for scan listing")
	}

	withScan := true
	params := harborartifact.NewListArtifactsParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithWithScanOverview(&withScan)
	resp, err := v2Client.Artifact.ListArtifacts(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot list Harbor artifacts for scans")
	}

	out := make([]*ScanStatus, 0, len(resp.Payload))
	for _, a := range resp.Payload {
		if a == nil {
			continue
		}
		scan := &ScanStatus{}
		for _, rep := range a.ScanOverview {
			scan.ID = rep.ReportID
			scan.Status = rep.ScanStatus
			scan.StartTime = time.Time(rep.StartTime)
			scan.EndTime = time.Time(rep.EndTime)
			if rep.Summary != nil {
				scan.CriticalCount = rep.Summary.Summary["Critical"]
				scan.HighCount = rep.Summary.Summary["High"]
				scan.MediumCount = rep.Summary.Summary["Medium"]
				scan.LowCount = rep.Summary.Summary["Low"]
			}
			break
		}
		out = append(out, scan)
	}
	return out, nil
}

// GetScan retrieves the scan result for an artifact by fetching the artifact
// with its scan_overview and extracting the first available NativeReportSummary.
// Returns (nil, nil) when the artifact itself does not exist (404). When no
// scan has been triggered yet, the scan_overview map is empty and the returned
// ScanStatus has Status="" (not-started).
//
// Caveat: Harbor's scan_overview is keyed by MIME type
// (e.g. "application/vnd.security.vulnerability.report; version=1.1"); we pick
// the first entry. Severity-level counts (Critical/High/Medium/Low) are stored
// in NativeReportSummary.Summary as a string->int64 map, not fixed fields.
func (c *HarborClient) GetScan(ctx context.Context, projectID, repoName, reference string) (*ScanStatus, error) {
	if projectID == "" {
		return nil, errors.New("project ID is required")
	}
	if repoName == "" {
		return nil, errors.New("repository name is required")
	}
	if reference == "" {
		return nil, errors.New("reference is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return nil, errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Retrieving Harbor scan", "projectId", projectID, "repo", repoName, "reference", reference)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return nil, errors.Wrap(err, "cannot resolve project for scan")
	}

	withScan := true
	params := harborartifact.NewGetArtifactParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithReference(reference).
		WithWithScanOverview(&withScan)
	resp, err := v2Client.Artifact.GetArtifact(ctx, params)
	if err != nil {
		if isHarborNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "cannot get Harbor artifact for scan status")
	}

	scan := &ScanStatus{}
	// Pick the first scan report from the overview map (keyed by MIME type).
	found := false
	for _, rep := range resp.Payload.ScanOverview {
		found = true
		scan.ID = rep.ReportID
		scan.Status = rep.ScanStatus
		scan.StartTime = time.Time(rep.StartTime)
		scan.EndTime = time.Time(rep.EndTime)
		if rep.Summary != nil {
			scan.CriticalCount = rep.Summary.Summary["Critical"]
			scan.HighCount = rep.Summary.Summary["High"]
			scan.MediumCount = rep.Summary.Summary["Medium"]
			scan.LowCount = rep.Summary.Summary["Low"]
		}
		break
	}
	// The artifact exists but has no scan overview yet → no scan has run. Report
	// not-found so the reconciler triggers a scan (Create -> TriggerScan) rather
	// than treating the un-scanned artifact as an existing, never-ready Scan.
	if !found {
		return nil, nil
	}
	return scan, nil
}

// StopScan stops a running vulnerability scan on an artifact.
// 404 (artifact or scan not found) is treated as success (idempotent).
func (c *HarborClient) StopScan(ctx context.Context, projectID, repoName, reference string) error {
	if projectID == "" {
		return errors.New("project ID is required")
	}
	if repoName == "" {
		return errors.New("repository name is required")
	}
	if reference == "" {
		return errors.New("reference is required")
	}

	v2Client := c.clientSet.V2()
	if v2Client == nil {
		return errors.New("failed to get Harbor v2 client")
	}

	c.logger.Info("Stopping Harbor artifact scan", "projectId", projectID, "repo", repoName, "reference", reference)

	projectName, err := c.resolveProjectName(ctx, projectID)
	if err != nil {
		return errors.Wrap(err, "cannot resolve project for scan")
	}

	params := harborscan.NewStopScanArtifactParams().WithContext(ctx).
		WithProjectName(projectName).
		WithRepositoryName(repoName).
		WithReference(reference)
	if _, err := v2Client.Scan.StopScanArtifact(ctx, params); err != nil {
		// A Scan is an action, not a deletable resource. Stop is only meaningful
		// while a scan is running; Harbor returns 422 (UnprocessableEntity) when
		// there's nothing to stop (e.g. the scan already completed) and 404 if the
		// artifact is gone. Both mean "nothing to do" — treat as success so the
		// managed resource can be deleted.
		if isHarborNotFound(err) || isHarborCode(err, http.StatusUnprocessableEntity) {
			return nil
		}
		return errors.Wrap(err, "cannot stop Harbor artifact scan")
	}
	return nil
}
