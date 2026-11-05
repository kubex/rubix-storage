# rubix-storage
Data storage for console access

## Gateway credentials

`gateway_credentials` stores global app identity credentials separately from workspace
settings and inbound signature keys. `storage.Provider` exposes create, list,
verify, rotate, and revoke methods. Raw `rxa_...` tokens are returned only by
create/rotate; database rows contain their SHA-256 hashes. Metadata and verification
results never expose a secret or digest.

Rotation atomically inserts a replacement and retires the previous credential,
with an optional overlap of up to 24 hours. The original expiry is preserved;
concurrent/repeated rotation of an already replaced credential is rejected.
Revocation and expiry are enforced on each verification. Workspace permissions
remain the responsibility of Rubix after app identity is verified.

## Gateway source IP rules

`gateway_source_ips` stores the global, vendor, and app source allowlists with
creator metadata. Scope tuples are validated, addresses and CIDRs are normalized,
and deletion always includes the owning scope/vendor/app. The effective query
returns only the union of the global rules, the caller's vendor rules, and the
caller's app rules; the HTTP layer authenticates the app and resolves trusted
network/proxy identity before applying them. No rule grants app or user identity.
