# Configuration (system authentication and OIDC)

The `Configuration` resource manages Harbor's system-wide authentication
settings: the primary auth mode and the OpenID Connect (OAuth2 SSO) settings
used when the mode is `oidc_auth`. It maps onto Harbor's
`PUT /api/v2.0/configurations` and `GET /api/v2.0/configurations` endpoints.

API group: `configuration.harbor.m.crossplane.io/v1beta1`, namespaced.

## One Configuration per Harbor instance

Harbor has a single set of system settings per instance. Each `Configuration`
manages the instance selected by its `providerConfigRef`, so create exactly one
per Harbor and point each at that instance's ProviderConfig. Two
`Configuration` resources for the same instance will overwrite each other.

## Example

```yaml
apiVersion: configuration.harbor.m.crossplane.io/v1beta1
kind: Configuration
metadata:
  name: harbor-oidc
  namespace: harbor-config
  annotations:
    kubernetes.io/description: "OIDC SSO for the golder Harbor"
spec:
  forProvider:
    authMode: oidc_auth
    oidc:
      name: Keycloak
      endpoint: https://sso.example.com/auth/realms/Example
      clientId: harbor
      clientSecretRef:
        name: harbor-oidc-client
        namespace: harbor-config
        key: clientSecret
      scope: "openid profile email"
      userClaim: preferred_username
      groupsClaim: groups
      adminGroup: harbor-admins
      verifyCert: true
      autoOnboard: true
  providerConfigRef:
    name: harbor-golder
```

See [examples/configuration/configuration.yaml](../examples/configuration/configuration.yaml).

## Fields

| Field | Harbor setting | Notes |
|-------|----------------|-------|
| `authMode` | `auth_mode` | `db_auth`, `ldap_auth`, `http_auth`, `oidc_auth` or `uaa_auth` |
| `oidc.name` | `oidc_name` | Label on the login page |
| `oidc.endpoint` | `oidc_endpoint` | OIDC issuer URL |
| `oidc.clientId` | `oidc_client_id` | |
| `oidc.clientSecretRef` | `oidc_client_secret` | Secret reference, see below |
| `oidc.scope` | `oidc_scope` | Space-separated scopes |
| `oidc.userClaim` | `oidc_user_claim` | Claim used as the Harbor username |
| `oidc.groupsClaim` | `oidc_groups_claim` | Claim listing the user's groups |
| `oidc.adminGroup` | `oidc_admin_group` | Members become Harbor system admins |
| `oidc.groupFilter` | `oidc_group_filter` | Regex limiting imported groups |
| `oidc.verifyCert` | `oidc_verify_cert` | Defaults to `true` |
| `oidc.autoOnboard` | `oidc_auto_onboard` | Create users on first OIDC login |
| `oidc.logout` | `oidc_logout` | End the OIDC session on logout |

## Behaviour

**Partial updates.** Harbor applies only the fields present in the request.
Fields you leave out of the spec keep their current value in Harbor, and
Observe only compares fields the spec sets.

**Clearing a value.** Set a field to an empty string to clear it, for example
`adminGroup: ""`. An empty string is sent to Harbor; an omitted field is not.

**Client secret.** Harbor never returns `oidc_client_secret`. The controller
stores a SHA-256 hash of the value it last wrote in
`status.atProvider.oidcClientSecretHash`. Observe compares that hash with the
referenced secret, so rotating the secret in Kubernetes triggers an update.
Changing the secret in Harbor by hand is not detected.

**Deletion.** Deleting the `Configuration` leaves Harbor's settings unchanged.
Removing the resource must not switch the login mode out from under users.

**Switching modes.** Setting `authMode: oidc_auth` before the OIDC fields are
correct can lock users out of Harbor. Apply the OIDC settings first and check
that SSO login works before switching the mode.

## Status

- `status.atProvider.authMode` is the mode Harbor reports.
- `status.atProvider.oidcClientSecretHash` is the hash of the last secret
  written. It is not the secret.

## Verification

Harbor's behaviour for omitted and empty fields was checked against the Harbor
v2.14.4 source: `UpdateConfigurations` marshals the request with `omitempty`
and passes the resulting map to `UpdateUserConfigs`, so only present keys are
written. Harbor's `Configurations` response model has no
`oidc_client_secret` field, which is why the hash is used.
