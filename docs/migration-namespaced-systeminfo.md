# Migrating SystemInfo and Quota to namespaced groups

SystemInfo was cluster-scoped and served from `systeminfo.harbor.crossplane.io`.
Quota was served from `quota.harbor.crossplane.io`. Both are now namespaced and
served from `systeminfo.harbor.m.crossplane.io/v1beta1` and
`quota.harbor.m.crossplane.io/v1beta1`.

Installing this package removes the old CRDs. Deleting a CRD deletes every
custom resource under it, and each resource's deletion policy then runs against
Harbor. Migrate before upgrading.

## Steps

1. Set `spec.deletionPolicy: Orphan` on every existing SystemInfo and Quota
   resource, so the Harbor objects are left in place when the old ones are removed.
2. Record the existing resources (`kubectl get <kind> -o yaml`), then delete the
   old ones once they are orphaned.
3. Upgrade the provider package.
4. Recreate each resource under the new group, with `metadata.namespace` set for
   SystemInfo. Use the external-name annotation to adopt the existing Harbor objects.
