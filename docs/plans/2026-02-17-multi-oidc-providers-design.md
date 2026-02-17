# Multi-OIDC Providers per Workspace

## Problem

Workspaces currently support a single OIDC provider stored as a JSON column on the workspaces table. We need to support multiple OIDC providers per workspace.

## Decision

Split OIDC providers into a dedicated `workspace_oidc_providers` table with UUID primary keys. Clean break from the old embedded JSON column.

## Database Table

```sql
CREATE TABLE workspace_oidc_providers (
    uuid           varchar(64) NOT NULL,
    workspace      varchar(64) NOT NULL,
    providerName   varchar(120) NOT NULL,
    clientID       varchar(255) NOT NULL,
    clientSecret   varchar(255) NULL,
    clientKeys     text NULL,
    issuerURL      varchar(255) NOT NULL,
    PRIMARY KEY (uuid)
);
CREATE INDEX oidc_workspace ON workspace_oidc_providers(workspace);
```

Primary key is `uuid` alone (auto-generated), with a workspace index for lookups.

## Model

`OIDCProvider` struct gains `Uuid` and `Workspace` fields. `Configured()` helper remains.

`Workspace` struct changes `OIDCProvider OIDCProvider` to `OIDCProviders []OIDCProvider`. Providers are not eagerly loaded with workspace queries; fetched separately via `GetOIDCProviders(workspace)`.

## Provider Interface

Remove: `SetWorkspaceOIDCProvider(workspaceUuid string, provider OIDCProvider) error`

Add:
- `GetOIDCProviders(workspace string) ([]OIDCProvider, error)`
- `GetOIDCProvider(workspace, uuid string) (*OIDCProvider, error)`
- `CreateOIDCProvider(workspace string, provider OIDCProvider) error`
- `MutateOIDCProvider(workspace, uuid string, options ...MutateOIDCProviderOption) error`
- `DeleteOIDCProvider(workspace, uuid string) error`

Mutation follows the existing option-function pattern (like `MutateRole`, `MutateBrand`).

## Scope

- Email domain whitelist remains workspace-level (unchanged)
- No new display/ordering fields on the provider
- No backward-compatibility shim for old column
