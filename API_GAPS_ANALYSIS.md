# Harbor Provider API - Gaps Analysis

## Executive Summary

The provider currently implements **18 basic CRUD operations** across 4 resources, but is missing **70+ critical operations** from the Harbor API. The client also doesn't implement parameters defined in the CRDs.

**Overall API Coverage: ~20%**

---

## 1. CRD Parameter Implementation Gap

### Project Resource - Only 2/12 Parameters Implemented

**Currently Supported:**
- ✅ Name
- ✅ Public

**Defined in CRD but NOT Used by Client:**
- ❌ EnableContentTrust
- ❌ EnableContentTrustCosign  
- ❌ AutoScanImages
- ❌ PreventVulnerableImages
- ❌ Severity (high/medium/low/critical)
- ❌ CVEAllowlist
- ❌ RegistryID
- ❌ StorageLimit
- ❌ Metadata

**Impact:** Users can't use 83% of Project features they expect from CRDs

### Project Observation - Zero Fields Implemented

**Defined in CRD but NOT Returned:**
- ❌ ID
- ❌ CreationTime
- ❌ UpdateTime
- ❌ OwnerID / OwnerName
- ❌ RepoCount / ChartCount
- ❌ CurrentStorageUsage

**Impact:** Controllers can't observe actual project state

---

## 2. Missing Resource Types (12)

### CRITICAL - Must Implement for Practical Use
1. **Repository** (4 methods)
   - ListRepositories() - Essential for browsing
   - GetRepository()
   - UpdateRepository()
   - DeleteRepository()

2. **Artifact** (6 methods)
   - ListArtifacts() - List images/charts
   - GetArtifact()
   - DeleteArtifact()
   - GetArtifactVulnerabilities() - Security scanning
   - CopyArtifact()
   - GetArtifactReferrers()

3. **Member/ProjectMember** (5 methods)
   - AddProjectMember() - Grant access
   - GetProjectMembers() - List access
   - UpdateProjectMember() - Change permissions
   - DeleteProjectMember() - Revoke access
   - GetProjectMember()

4. **Scan/Vulnerability** (4 methods)
   - TriggerScan()
   - ListScans()
   - GetScan()
   - StopScan()

### HIGH - Important for Security & Operations
5. **Robot** (5 methods) - Service account management
6. **Webhook** (5 methods) - Event-driven automation
7. **Replication** (8 methods) - Cross-registry sync
8. **Retention** (5 methods) - Image cleanup policies

### MEDIUM - Operational/Administrative
9. **ProjectMetadata** (4 methods)
10. **Quota** (3 methods) - Resource limits
11. **Label** (5 methods) - Resource tagging
12. **UserGroup** (5 methods) - Organizational structure

---

## 3. Method Implementation Status

```
Implemented:  18 methods (20%)
├─ Project:        5/5 ✅
├─ User:           4/4 ✅
├─ Scanner:        5/5 ✅
└─ Registry:       4/4 ✅

Missing:      70+ methods (80%)
├─ Repository:     4 ❌ CRITICAL
├─ Artifact:       7 ❌ CRITICAL
├─ Member:         5 ❌ CRITICAL
├─ Scan:           4 ❌ CRITICAL
├─ Robot:          5 ❌ HIGH
├─ Webhook:        5 ❌ HIGH
├─ Replication:    8 ❌ HIGH
├─ Retention:      5 ❌ MEDIUM
├─ Label:          5 ❌ MEDIUM
├─ ProjectMetadata:4 ❌ MEDIUM
├─ Quota:          3 ❌ MEDIUM
├─ UserGroup:      5 ❌ MEDIUM
└─ Other:         20+ ❌ LOW
```

---

## 4. Feature Gap by Priority

### Phase 1: CRITICAL (Required for MVP)

#### Repository Management
- [ ] ListRepositories(ctx, projectID) - Browse repos in project
- [ ] GetRepository(ctx, projectID, repoName) - Get repo metadata
- [ ] DeleteRepository(ctx, projectID, repoName) - Remove repository

#### Artifact Management
- [ ] ListArtifacts(ctx, projectID, repoName) - List images/charts
- [ ] GetArtifact(ctx, projectID, repoName, reference) - Image details
- [ ] DeleteArtifact(ctx, projectID, repoName, reference) - Remove image
- [ ] GetArtifactVulnerabilities(ctx, projectID, repoName, reference) - Scan results

#### Project Member Management
- [ ] AddProjectMember(ctx, projectID, username, role) - Grant access
- [ ] ListProjectMembers(ctx, projectID) - Who has access
- [ ] UpdateProjectMember(ctx, projectID, username, role) - Change permissions
- [ ] DeleteProjectMember(ctx, projectID, username) - Revoke access

**Why Critical:** Without these, users can't:
- Browse images in projects
- Control who can access projects
- See scan/vulnerability results

### Phase 2: HIGH PRIORITY (Planned Features)

#### Scan/Vulnerability Management
- [ ] TriggerScan(ctx, projectID, repoName, reference) - Run scan
- [ ] ListScans(ctx, projectID, repoName) - Scan history
- [ ] GetScan(ctx, projectID, repoName, scanID) - Scan results
- [ ] CreateScanSchedule(ctx, projectID, schedule) - Automated scanning

#### Robot Accounts (Service Authentication)
- [ ] CreateRobot(ctx, name, description, permissions) - Service account
- [ ] ListRobots(ctx) - List service accounts
- [ ] GetRobot(ctx, robotID) - Account details
- [ ] UpdateRobot(ctx, robotID, permissions) - Modify access
- [ ] DeleteRobot(ctx, robotID) - Revoke service account

**Why High:** Enable:
- Automated CI/CD access to registries
- Service-to-service authentication
- Audit trail for robot access

### Phase 3: MEDIUM PRIORITY (Nice to Have)

#### Replication Policies
- [ ] CreateReplicationPolicy(ctx, policy) - Cross-registry sync
- [ ] ListReplicationPolicies(ctx)
- [ ] UpdateReplicationPolicy(ctx, policyID, policy)
- [ ] TriggerReplication(ctx, policyID) - Manual sync
- [ ] ListReplicationExecutions(ctx, policyID) - Sync history

#### Retention Policies
- [ ] CreateRetentionPolicy(ctx, projectID, rules) - Image cleanup
- [ ] ListRetentionPolicies(ctx, projectID)
- [ ] UpdateRetentionPolicy(ctx, projectID, policyID, rules)
- [ ] DeleteRetentionPolicy(ctx, projectID, policyID)

**Why Medium:** Operational convenience but not essential

### Phase 4: LOW PRIORITY (Administrative)

#### WebHooks, Labels, UserGroups, Configuration
- [ ] Webhook management (5 methods)
- [ ] Label management (5 methods)
- [ ] UserGroup management (5 methods)
- [ ] System configuration
- [ ] CVE allowlist management
- [ ] LDAP/OIDC configuration

---

## 5. Data Structure Gaps

### ProjectParameters (CRD vs Client)

```go
// What the CRD defines:
type ProjectParameters struct {
    Name                   string            // ✅ Used
    Public                *bool             // ✅ Used
    EnableContentTrust    *bool             // ❌ Ignored
    EnableContentTrustCosign *bool          // ❌ Ignored
    AutoScanImages        *bool             // ❌ Ignored
    PreventVulnerableImages *bool           // ❌ Ignored
    Severity              *string           // ❌ Ignored
    CVEAllowlist          []string          // ❌ Ignored
    RegistryID            *int64            // ❌ Ignored
    StorageLimit          *int64            // ❌ Ignored
    Metadata              map[string]string // ❌ Ignored
}

// What the client actually uses:
type ProjectSpec struct {
    Name   string
    Public bool
}
```

### ProjectObservation (CRD vs Client)

```go
// What the CRD expects:
type ProjectObservation struct {
    ID                  *string       // ❌ Not returned
    CreationTime        *metav1.Time  // ❌ Not returned
    UpdateTime          *metav1.Time  // ❌ Not returned
    OwnerID             *int64        // ❌ Not returned
    OwnerName           *string       // ❌ Not returned
    RepoCount           *int64        // ❌ Not returned
    ChartCount          *int64        // ❌ Not returned
    CurrentStorageUsage *int64        // ❌ Not returned
}

// What the client returns:
type ProjectStatus struct {
    Name      string
    Public    bool
    CreatedAt time.Time
}
```

---

## 6. Error Handling Gaps

### Missing Validations
- ❌ Project name format validation (alphanumeric, hyphens only)
- ❌ CVE ID format validation (CVE-YYYY-NNNNN)
- ❌ Storage limit range validation (must be > 0)
- ❌ Severity enum validation (must be: negligible, low, medium, high, critical)
- ❌ Registry ID existence check
- ❌ Username/email format validation
- ❌ Role validation for members (admin, developer, guest, master)

### Missing Error Scenarios
- ❌ Duplicate project names
- ❌ Invalid registry ID
- ❌ Insufficient storage quota
- ❌ Invalid member roles
- ❌ CVE allowlist entry conflicts
- ❌ User already a member of project
- ❌ Member with active objects (can't delete)

---

## 7. API Implementation Roadmap

### Immediate (This Sprint)
- [x] Fix logging throughout client
- [x] Add HTTP client configuration
- [x] Improve error handling

### Short Term (Next 2 Sprints)
- [ ] Add ProjectParameters to client (10 missing fields)
- [ ] Return full ProjectObservation
- [ ] Add validation for all inputs
- [ ] Add error handling for common scenarios

### Medium Term (Next Month)
- [ ] Implement Repository CRUD (CRITICAL)
- [ ] Implement Artifact operations (CRITICAL)
- [ ] Implement Member management (CRITICAL)
- [ ] Implement Scan operations (CRITICAL)

### Long Term (Future)
- [ ] Robot account management
- [ ] Replication policies
- [ ] Retention policies
- [ ] Webhook management
- [ ] Additional resources as needed

---

## 8. Impact Assessment

### Current Limitations
Users can ONLY:
- ✅ Create/Read/Update/Delete Projects (basic)
- ✅ Create/Read/Update/Delete Users
- ✅ Create/Read/Update/Delete Scanners
- ✅ Create/Read/Update/Delete Registries

Users CANNOT:
- ❌ Browse images in projects
- ❌ See vulnerability scans
- ❌ Manage project access
- ❌ Set security policies
- ❌ Automate image cleanup
- ❌ Set up cross-registry replication
- ❌ Create service accounts

### For Production Readiness
The provider needs AT LEAST:
1. **Repository + Artifact management** - Can't manage images
2. **Member management** - Can't control access
3. **Scan operations** - Can't enforce security
4. **Robot accounts** - Can't automate CI/CD

---

## Summary

| Aspect | Coverage | Status |
|--------|----------|--------|
| **Core CRUD** | 100% (4/4 resources) | ✅ Complete |
| **CRD Parameters** | 17% (2/12 for Projects) | ❌ Major Gap |
| **Method Count** | 20% (18/90+) | ❌ Major Gap |
| **Critical Features** | 0% | ❌ Missing |
| **Error Handling** | ~30% | ⚠️ Partial |
| **Testing** | ~20% | ❌ Insufficient |

**Verdict:** Provider is **not production-ready**. Needs Phase 2 (Critical) features to be useful.
