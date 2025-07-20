# Terraform Removal Plan for Harbor Provider

## Overview
This document outlines the plan to remove terraform-based code from the Harbor provider and transition to native API implementations.

## Directories to Remove
- `.work/terraform/` - Terraform working directory and all contents
- `.work/goharbor/` - Generated terraform provider files

## Files to Remove/Replace

### Generated Controllers (zz_controller.go files)
All terraform-based controllers that need native replacements:
- `internal/controller/config/configauth/zz_controller.go`
- `internal/controller/config/configsecurity/zz_controller.go`
- `internal/controller/config/configsystem/zz_controller.go`
- `internal/controller/garbagecollection/garbagecollection/zz_controller.go`
- `internal/controller/group/group/zz_controller.go`
- `internal/controller/interrogationservices/interrogationservices/zz_controller.go`
- `internal/controller/label/label/zz_controller.go`
- `internal/controller/preheatinstance/preheatinstance/zz_controller.go`
- `internal/controller/project/immutabletagrule/zz_controller.go`
- `internal/controller/project/membergroup/zz_controller.go`
- `internal/controller/project/memberuser/zz_controller.go`
- `internal/controller/project/project/zz_controller.go`
- `internal/controller/project/retentionpolicy/zz_controller.go`
- `internal/controller/project/webhook/zz_controller.go`
- `internal/controller/purgeauditlog/purgeauditlog/zz_controller.go`
- `internal/controller/registry/registry/zz_controller.go`
- `internal/controller/registry/replication/zz_controller.go`
- `internal/controller/robotaccount/robotaccount/zz_controller.go` - **Already replaced with native/**
- `internal/controller/tasks/task/zz_controller.go`
- `internal/controller/user/user/zz_controller.go`

### Generated API Types (zz_*_types.go files)
API types that may need updates to remove terraform dependencies:
- `apis/config/v1alpha1/zz_configauth_types.go`
- `apis/config/v1alpha1/zz_configsecurity_types.go`
- `apis/config/v1alpha1/zz_configsystem_types.go`
- `apis/garbagecollection/v1alpha1/zz_garbagecollection_types.go`
- `apis/group/v1alpha1/zz_group_types.go`
- `apis/interrogationservices/v1alpha1/zz_interrogationservices_types.go`
- `apis/label/v1alpha1/zz_label_types.go`
- `apis/preheatinstance/v1alpha1/zz_preheatinstance_types.go`
- `apis/project/v1alpha1/zz_immutabletagrule_types.go`
- `apis/project/v1alpha1/zz_membergroup_types.go`
- `apis/project/v1alpha1/zz_memberuser_types.go`
- `apis/project/v1alpha1/zz_project_types.go`
- `apis/project/v1alpha1/zz_retentionpolicy_types.go`
- `apis/project/v1alpha1/zz_webhook_types.go`
- `apis/purgeauditlog/v1alpha1/zz_purgeauditlog_types.go`
- `apis/registry/v1alpha1/zz_registry_types.go`
- `apis/registry/v1alpha1/zz_replication_types.go`
- `apis/robotaccount/v1alpha1/zz_robotaccount_types.go`
- `apis/tasks/v1alpha1/zz_task_types.go`
- `apis/user/v1alpha1/zz_user_types.go`

### Core Files to Update
- `cmd/provider/main.go` - Remove terraform provider initialization
- `internal/clients/harbor.go` - Remove terraform client setup
- `config/external_name.go` - May need updates for native controllers

### Configuration Files
- `config/user/config.go` - Contains terraform references
- `config/retentionpolicy/config.go` - Contains terraform references

## Implementation Order

### Phase 1: Robot Account (COMPLETED)
- ✅ Created native controller at `internal/controller/robotaccount/native/`
- ✅ Implemented full CRUD operations with external-name support
- ✅ Added comprehensive unit tests

### Phase 2: Core Resources (Priority)
1. **User** - Basic user management
2. **Project** - Core project operations
3. **Label** - Global and project labels
4. **Registry** - Registry endpoints

### Phase 3: Project Resources
1. **MemberUser** - Project user memberships
2. **MemberGroup** - Project group memberships
3. **Webhook** - Project webhooks
4. **ImmutableTagRule** - Tag immutability rules
5. **RetentionPolicy** - Retention policies

### Phase 4: System Resources
1. **ConfigAuth** - Authentication configuration
2. **ConfigSecurity** - Security settings
3. **ConfigSystem** - System configuration
4. **GarbageCollection** - GC schedules
5. **PurgeAuditLog** - Audit log cleanup

### Phase 5: Advanced Features
1. **Group** - User groups
2. **Task** - System tasks
3. **Replication** - Registry replication
4. **PreheatInstance** - Image preheating
5. **InterrogationServices** - Vulnerability scanning

## Migration Strategy

1. **Implement native controllers incrementally**
   - Start with most-used resources
   - Maintain backward compatibility during transition
   - Use feature flags if needed

2. **Update controller registration**
   - Modify `cmd/provider/main.go` to register native controllers
   - Remove terraform provider setup

3. **Clean up dependencies**
   - Remove terraform-provider-harbor dependency
   - Remove upjet/terraform dependencies
   - Update go.mod

4. **Update build process**
   - Remove terraform provider generation steps
   - Update Makefile targets
   - Clean up .work directory

## Benefits of Native Implementation
- Full control over external-name handling
- Better error messages and debugging
- Reduced binary size
- Faster startup time
- No terraform state management overhead
- Direct API integration for better performance

## Testing Requirements
- Unit tests for each native controller
- Integration tests with real Harbor instance
- Migration tests for existing resources
- Performance comparison tests