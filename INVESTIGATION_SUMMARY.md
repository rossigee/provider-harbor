# E2E Workflow Investigation & Fixes - Summary

**Date:** 2026-08-23  
**Investigation:** GitHub Actions e2e workflow failures for provider-harbor

## Issue Overview

The e2e workflow was failing with multiple root causes:
1. Crossplane CLI download URL changed (breaking change in v2.4.0)
2. Go version mismatch (workflow used 1.26.5, go.mod requires 1.26.6)
3. Artifact resource reconciliation incomplete (status field not set)

## ✅ Fixed Issues (Verified Working)

### 1. Crossplane CLI Migration
**Problem:** v2.4.0 moved CLI from `releases.crossplane.io` to `cli.crossplane.io`
- Old URL: `https://releases.crossplane.io/stable/v2.3.2/bin/linux_amd64/crank`
- New URL: `https://cli.crossplane.io/stable/v2.5.0/bin/linux_amd64/crossplane`
- Binary name changed: `crank` → `crossplane`

**Fix:** Commit `ff80496`
- Updated `.github/workflows/e2e.yaml`
- Changed download URL to new CLI channel
- Updated CLI version to v2.5.0 (latest stable)

**Status:** ✅ WORKING - Verified in workflows #59-61

### 2. Go Version Requirement
**Problem:** Workflow used Go 1.26.5 but go.mod specifies 1.26.6
- Error: `go: go.mod requires go >= 1.26.6 (running go 1.26.5; GOTOOLCHAIN=local)`
- Blocked build + push step

**Fix:** Included in commit `ff80496`
- Updated `GO_VERSION` from 1.26.5 to 1.26.6 in e2e.yaml

**Status:** ✅ WORKING - Verified in all test runs

### 3. Artifact Resource Type Support
**Problem:** ProviderConfig loader didn't recognize Artifact types
- Missing type cast in `NewHarborClientFromProviderConfig`
- Would cause "unsupported managed resource type" error

**Fix:** Commit `2d5c94e`
- Added `artifactv1beta1` import
- Added Artifact type handling to ProviderConfig loader

**Status:** ✅ PARTIAL - Support added, but reconciliation still incomplete

### 4. Harbor API Integration
**Problem:** GetArtifact method was a stub returning hardcoded mock data
- No actual API calls to Harbor
- Prevented proper resource reconciliation

**Fix:** Commit `b4b72bd`
- Implemented real Harbor SDK calls via `sdkartifact` package
- Proper GetArtifactParams setup
- Response payload handling
- Error recovery with logging

**Status:** ✅ IMPLEMENTED - But Artifact reconciliation still fails

## ❌ Unresolved Issue

### Artifact Status Field Not Populated
**Symptom:** After 10+ minutes of waiting, Artifact resource has no status field
```
* status: Required value: field not found in the input object
```

**What works:**
- Artifact resource is created successfully
- Controller is registered and starts
- All other resource types (Project, Member, Registry, etc.) work fine

**What fails:**
- Artifact.status field remains empty
- Ready condition is never set
- Test times out waiting for Ready=True

**Likely Root Causes:**
1. CRD definition issue - status subresource not properly defined
2. Controller reconciliation error - Observe method not completing
3. Managed reconciler framework configuration - Ready condition not being set
4. Harbor client initialization - failing silently in Connect method

**Investigation Done:**
- ✅ Harbor API implementation verified working in principle
- ✅ ProviderConfig loader handles Artifact type
- ✅ Controller is registered in main.go
- ✅ All infrastructure (CLI, Go, build, push) works
- ❓ Provider pod reconciliation logs not accessible (would need kubectl access to kind cluster)

**Would Need to Resolve:**
- Access to provider pod logs: `kubectl logs -n crossplane-system deployment/crossplane-provider-harbor`
- Check controller errors and reconciliation flow
- Verify CRD status subresource definition
- Inspect managed reconciler framework behavior

## Test Results

| Workflow | Status | Issue |
|----------|--------|-------|
| #58 | ❌ FAIL | Original: CLI 404 + Go version |
| #59 | ❌ FAIL | Artifact status assertion (10min timeout) |
| #60 | ❌ FAIL | Artifact status assertion (10min timeout) |
| #61 | ❌ FAIL | Artifact status assertion (10min timeout) |

**All failures on same step:** Artifact status assertion (after Create/Apply succeeds)

## Commits

```
ff80496 - fix(ci): migrate Crossplane CLI download to new separate release channel
2d5c94e - fix: implement Artifact support in Harbor client
b4b72bd - feat: implement Harbor API calls for GetArtifact
```

## CI/CD Pipeline Status

✅ **Infrastructure:** Working correctly
- CLI downloads from correct URL
- Go version matches requirements
- Provider builds and deploys
- All non-Artifact resources reconcile correctly

❌ **Provider Implementation:** Incomplete
- Artifact reconciliation incomplete
- Status field not populated
- Requires deeper Crossplane framework debugging

## Recommendations

### For Immediate Use
- The CLI and Go version fixes are solid and should be merged
- Other resource types work correctly
- CI/CD infrastructure is healthy

### For Artifact Support
1. Debug provider pod logs in kind cluster during e2e test
2. Check CRD status subresource configuration
3. Verify managed.NewReconciler setup in artifact_controller.go
4. Implement full Harbor API integration if API connectivity is issue
5. Add structured logging to reconciliation pipeline

## Files Modified

- `.github/workflows/e2e.yaml` - CLI URL + Go version fixes
- `internal/clients/harbor.go` - Artifact type support + Harbor API calls
- `cmd/provider/main.go` - (already had Artifact controller registered)

