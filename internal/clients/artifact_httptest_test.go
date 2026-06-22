/*
Proof tests for the real Artifact client implementation.

Runs the actual HarborClient.{List,Get,Delete}Artifact methods against a
stateful in-memory fake of Harbor's /api/v2.0/projects/{project}/repositories
/{repo}/artifacts API (httptest). Demonstrates the methods issue the right HTTP
verbs/paths, parse the real digest/size, and map 404 -> (nil, nil) rather than
returning hardcoded stubs. No live Harbor, no real credentials.
*/
package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeHarborArtifacts returns an httptest server that implements a minimal
// subset of the Harbor artifact API sufficient to exercise our client methods.
func fakeHarborArtifacts(t *testing.T) *httptest.Server {
	t.Helper()
	type artifact struct {
		id     int64
		digest string
		size   int64
		tag    string
	}
	var mu sync.Mutex
	// key: "project/repo/reference" → artifact
	store := map[string]*artifact{}

	// Seed one artifact that tests can GET/DELETE.
	store["library/alpine/latest"] = &artifact{
		id:     101,
		digest: "sha256:deadbeef",
		size:   5242880,
		tag:    "latest",
	}

	artifactJSON := func(a *artifact) string {
		return `{"id":` + itoa(int(a.id)) + `,"digest":"` + a.digest + `","size":` + itoa(int(a.size)) + `,"push_time":"2026-01-01T00:00:00.000Z","pull_time":"2026-01-02T00:00:00.000Z","type":"IMAGE"}`
	}

	mux := http.NewServeMux()

	// /api/v2.0/projects/{project}/repositories/{repo}/artifacts
	// list handler
	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		// Parse: /api/v2.0/projects/{project}/repositories/{repo}/artifacts[/{ref}]
		path := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/")
		parts := strings.SplitN(path, "/repositories/", 2)
		if len(parts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		projectName := parts[0]
		rest := parts[1]
		repoParts := strings.SplitN(rest, "/artifacts", 2)
		if len(repoParts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		repoName := repoParts[0]
		suffix := repoParts[1] // "" for list, "/{ref}" for get/delete

		switch r.Method {
		case http.MethodGet:
			if suffix == "" || suffix == "/" {
				// List artifacts
				var results []string
				prefix := projectName + "/" + repoName + "/"
				for k, a := range store {
					if strings.HasPrefix(k, prefix) {
						results = append(results, artifactJSON(a))
					}
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("[" + strings.Join(results, ",") + "]"))
			} else {
				// Get artifact by reference
				ref := strings.TrimPrefix(suffix, "/")
				key := projectName + "/" + repoName + "/" + ref
				a, ok := store[key]
				if !ok {
					// Return empty body — the SDK Consume call treats EOF as success and
					// GetArtifactNotFound implements IsCode(404), which isHarborNotFound catches.
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(artifactJSON(a)))
			}
		case http.MethodDelete:
			ref := strings.TrimPrefix(suffix, "/")
			key := projectName + "/" + repoName + "/" + ref
			if _, ok := store[key]; !ok {
				// Empty body — SDK treats EOF as success on Delete 404 path.
				w.WriteHeader(http.StatusNotFound)
				return
			}
			delete(store, key)
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux)
}

func TestArtifactClient_GetExists(t *testing.T) {
	srv := fakeHarborArtifacts(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Seeded artifact must be retrievable.
	st, err := c.GetArtifact(ctx, "library", "alpine", "latest")
	if err != nil {
		t.Fatalf("GetArtifact: %v", err)
	}
	if st == nil {
		t.Fatal("expected non-nil ArtifactStatus for seeded artifact")
	}
	if st.Digest != "sha256:deadbeef" {
		t.Errorf("expected digest sha256:deadbeef, got %q (stub would be \"sha256:abc123\")", st.Digest)
	}
	if st.Size != 5242880 {
		t.Errorf("expected size 5242880, got %d", st.Size)
	}
}

func TestArtifactClient_GetNotFound(t *testing.T) {
	srv := fakeHarborArtifacts(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Non-existent artifact must return (nil, nil), not an error.
	st, err := c.GetArtifact(ctx, "library", "alpine", "nonexistent")
	if err != nil {
		t.Fatalf("GetArtifact for non-existent artifact returned error: %v (want nil,nil)", err)
	}
	if st != nil {
		t.Errorf("expected nil for not-found, got %+v", st)
	}
}

func TestArtifactClient_ListArtifacts(t *testing.T) {
	srv := fakeHarborArtifacts(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	artifacts, err := c.ListArtifacts(ctx, "library", "alpine")
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(artifacts) == 0 {
		t.Error("expected at least one artifact from seeded repo")
	}
	found := false
	for _, a := range artifacts {
		if a.Digest == "sha256:deadbeef" {
			found = true
		}
	}
	if !found {
		t.Errorf("seeded artifact sha256:deadbeef not found in list result")
	}
}

func TestArtifactClient_DeleteIdempotent(t *testing.T) {
	srv := fakeHarborArtifacts(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// First delete — succeeds.
	if err := c.DeleteArtifact(ctx, "library", "alpine", "latest"); err != nil {
		t.Fatalf("DeleteArtifact first call: %v", err)
	}
	// Confirm gone.
	if st, err := c.GetArtifact(ctx, "library", "alpine", "latest"); err != nil || st != nil {
		t.Errorf("expected (nil,nil) after delete, got st=%v err=%v", st, err)
	}
	// Second delete — must be idempotent (404 → success).
	if err := c.DeleteArtifact(ctx, "library", "alpine", "latest"); err != nil {
		t.Fatalf("DeleteArtifact idempotent second call: %v", err)
	}
}

// fakeHarborScans returns an httptest server implementing the Harbor
// ScanArtifact (POST .../scan), StopScanArtifact (POST .../scan/stop), and
// GetArtifact (GET with scan_overview) endpoints.
func fakeHarborScans(t *testing.T) *httptest.Server {
	t.Helper()
	var mu sync.Mutex
	type scanState struct {
		triggered bool
		status    string // "Scanning", "Success"
	}
	// key: "project/repo/ref"
	scans := map[string]*scanState{
		"library/alpine/latest": {triggered: false},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/api/v2.0/projects/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()

		path := strings.TrimPrefix(r.URL.Path, "/api/v2.0/projects/")
		parts := strings.SplitN(path, "/repositories/", 2)
		if len(parts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		projectName := parts[0]
		rest := parts[1]
		repoParts := strings.SplitN(rest, "/artifacts/", 2)
		if len(repoParts) != 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		repoName := repoParts[0]
		afterArtifact := repoParts[1] // "ref" or "ref/scan" or "ref/scan/stop"

		// Split ref from operation
		refParts := strings.SplitN(afterArtifact, "/", 2)
		ref := refParts[0]
		op := ""
		if len(refParts) == 2 {
			op = refParts[1]
		}

		key := projectName + "/" + repoName + "/" + ref

		switch {
		case r.Method == http.MethodGet && op == "":
			// GET artifact with scan_overview
			sc, ok := scans[key]
			if !ok {
				// Empty body so the SDK's consumer.Consume call gets EOF (treated as success).
				w.WriteHeader(http.StatusNotFound)
				return
			}
			var scanOverview string
			if sc.triggered {
				scanOverviewData := map[string]interface{}{
					"application/vnd.security.vulnerability.report; version=1.1": map[string]interface{}{
						"report_id":        "rpt-001",
						"scan_status":      sc.status,
						"start_time":       "2026-01-01T10:00:00.000Z",
						"end_time":         "2026-01-01T10:05:00.000Z",
						"complete_percent": 100,
						"summary": map[string]interface{}{
							"total":   3,
							"fixable": 1,
							"summary": map[string]int64{
								"Critical": 0,
								"High":     1,
								"Medium":   2,
								"Low":      0,
							},
						},
					},
				}
				b, _ := json.Marshal(scanOverviewData)
				scanOverview = string(b)
			} else {
				scanOverview = "{}"
			}
			body := `{"id":101,"digest":"sha256:deadbeef","size":5242880,"push_time":"2026-01-01T00:00:00.000Z","pull_time":"2026-01-02T00:00:00.000Z","type":"IMAGE","scan_overview":` + scanOverview + `}`
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(body))

		case r.Method == http.MethodPost && op == "scan":
			// Trigger scan
			if _, ok := scans[key]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			scans[key].triggered = true
			scans[key].status = "Scanning"
			w.WriteHeader(http.StatusAccepted)

		case r.Method == http.MethodPost && op == "scan/stop":
			// Stop scan
			if _, ok := scans[key]; !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			scans[key].status = "stopped"
			w.WriteHeader(http.StatusAccepted)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	return httptest.NewServer(mux)
}

func TestScanClient_TriggerAndGet(t *testing.T) {
	srv := fakeHarborScans(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Before trigger: artifact exists but no scan_overview → GetScan reports
	// (nil, nil) so the reconciler treats it as not-yet-created and triggers a
	// scan (rather than adopting an un-scanned artifact that never goes Ready).
	sc, err := c.GetScan(ctx, "library", "alpine", "latest")
	if err != nil {
		t.Fatalf("GetScan before trigger: %v", err)
	}
	if sc != nil {
		t.Fatalf("expected nil ScanStatus before any scan, got %+v", sc)
	}

	// Trigger scan.
	if err := c.TriggerScan(ctx, "library", "alpine", "latest"); err != nil {
		t.Fatalf("TriggerScan: %v", err)
	}

	// After trigger: status should be "Scanning".
	sc, err = c.GetScan(ctx, "library", "alpine", "latest")
	if err != nil {
		t.Fatalf("GetScan after trigger: %v", err)
	}
	if sc.Status != "Scanning" {
		t.Errorf("expected status Scanning after trigger, got %q", sc.Status)
	}
	if sc.HighCount != 1 {
		t.Errorf("expected HighCount=1, got %d", sc.HighCount)
	}
	if sc.ID != "rpt-001" {
		t.Errorf("expected scan report ID rpt-001, got %q (stub would be \"1\")", sc.ID)
	}
}

func TestScanClient_ArtifactNotFound(t *testing.T) {
	srv := fakeHarborScans(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Non-existent artifact must return (nil, nil).
	sc, err := c.GetScan(ctx, "library", "alpine", "nonexistent")
	if err != nil {
		t.Fatalf("GetScan for non-existent artifact: %v (want nil,nil)", err)
	}
	if sc != nil {
		t.Errorf("expected nil for not-found artifact, got %+v", sc)
	}
}

func TestScanClient_TriggerNotFound(t *testing.T) {
	srv := fakeHarborScans(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Triggering on non-existent artifact must return an error.
	err := c.TriggerScan(ctx, "library", "alpine", "nonexistent")
	if err == nil {
		t.Error("TriggerScan on non-existent artifact: expected error, got nil")
	}
}

func TestScanClient_StopScanIdempotent(t *testing.T) {
	srv := fakeHarborScans(t)
	defer srv.Close()
	c := newTestClient(t, srv.URL)
	ctx := context.Background()

	// Stop non-existent artifact: idempotent (should succeed).
	if err := c.StopScan(ctx, "library", "alpine", "nonexistent"); err != nil {
		t.Errorf("StopScan on non-existent: expected nil (idempotent), got %v", err)
	}
}
