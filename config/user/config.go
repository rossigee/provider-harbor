package user

// ExternalName handling for Harbor users.
//
// In Crossplane, the external-name annotation tracks the identifier of the external resource.
// For Harbor users, we use the username as the external identifier because:
//
// 1. Usernames are user-friendly and stable identifiers
// 2. Harbor API operations primarily work with usernames
// 3. Numeric user IDs are internal Harbor implementation details
//
// The User controller (internal/controller/user/user_controller.go) handles
// the external-name mapping by:
//
// 1. Setting external-name annotation = username when creating users
// 2. Using the username from external-name annotation for Observe/Update/Delete operations
// 3. This ensures consistent identification of Harbor users across Kubernetes and Harbor
//
// Example external-name logic (implemented in user_controller.go):
//   - Create: Set external-name annotation to cr.Spec.ForProvider.Username
//   - Observe: Use external-name annotation to query user in Harbor
//   - Update: Use external-name annotation to identify user for updates
//   - Delete: Use external-name annotation to identify user for deletion
