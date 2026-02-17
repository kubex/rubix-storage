# Multi-OIDC Providers per Workspace — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Support multiple OIDC providers per workspace by splitting OIDC config into a dedicated table with full CRUD operations.

**Architecture:** New `workspace_oidc_providers` table with UUID primary key. OIDCProvider struct gains identity fields. Provider interface replaces `SetWorkspaceOIDCProvider` with Get/Create/Mutate/Delete methods following existing patterns (like brands, teams).

**Tech Stack:** Go, database/sql, SQLite/MySQL, existing migration system

---

### Task 1: Add migration for new table and drop old column

**Files:**
- Modify: `storage/sql/schema.go:186-188` (add new migrations after existing OIDC ones)

**Step 1: Add the new table migration and index**

In `storage/sql/schema.go`, append these migrations at the end of the `migrations()` function (before `return queries`):

```go
queries = append(queries, migQuery("CREATE TABLE `workspace_oidc_providers` ("+
    "`uuid`           varchar(64)  NOT NULL,"+
    "`workspace`      varchar(64)  NOT NULL,"+
    "`providerName`   varchar(120) NOT NULL,"+
    "`clientID`       varchar(255) NOT NULL,"+
    "`clientSecret`   varchar(255) NULL,"+
    "`clientKeys`     text NULL,"+
    "`issuerURL`      varchar(255) NOT NULL,"+
    "PRIMARY KEY (`uuid`)"+
    ");"))
queries = append(queries, migQuery("CREATE INDEX `oidc_workspace` ON `workspace_oidc_providers`(`workspace`);"))
```

**Step 2: Run tests to verify migration applies cleanly**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go test ./storage/sql/ -run TestIntegration_SQLite_EndToEnd -v`
Expected: PASS (existing tests still work, new table created during Initialize)

**Step 3: Commit**

```bash
git add storage/sql/schema.go
git commit -m "feat: add workspace_oidc_providers table migration"
```

---

### Task 2: Update OIDCProvider model and add mutation types

**Files:**
- Modify: `rubix/oidc.go`

**Step 1: Update the OIDCProvider struct and add mutation types**

Replace the entire contents of `rubix/oidc.go` with:

```go
package rubix

type OIDCProvider struct {
	Uuid         string `json:"uuid"`
	Workspace    string `json:"workspace"`
	ProviderName string `json:"providerName"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	ClientKeys   string `json:"clientKeys"`
	IssuerURL    string `json:"issuerURL"`
}

func (o OIDCProvider) Configured() bool {
	return o.ClientID != "" && o.IssuerURL != ""
}

type MutateOIDCProviderPayload struct {
	ProviderName *string
	ClientID     *string
	ClientSecret *string
	ClientKeys   *string
	IssuerURL    *string
}

type MutateOIDCProviderOption func(*MutateOIDCProviderPayload)

func WithOIDCProviderName(name string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ProviderName = &name }
}

func WithOIDCClientID(clientID string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientID = &clientID }
}

func WithOIDCClientSecret(secret string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientSecret = &secret }
}

func WithOIDCClientKeys(keys string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientKeys = &keys }
}

func WithOIDCIssuerURL(url string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.IssuerURL = &url }
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go build ./rubix/`
Expected: Compilation error — `Workspace` struct still references old `OIDCProvider` field type. That's fine, we fix it in the next task.

**Step 3: Commit**

```bash
git add rubix/oidc.go
git commit -m "feat: add UUID/workspace fields and mutation types to OIDCProvider"
```

---

### Task 3: Update Workspace struct — remove embedded OIDC

**Files:**
- Modify: `rubix/workspace.go` (remove OIDCProvider field, add OIDCProviders slice)

**Step 1: Update Workspace struct**

In `rubix/workspace.go`, replace the `OIDCProvider` field:

Change:
```go
OIDCProvider          OIDCProvider      `json:"oidcProvider"`
```

To:
```go
OIDCProviders         []OIDCProvider    `json:"oidcProviders"`
```

**Step 2: Verify it compiles (will fail on SQL layer)**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go vet ./...`
Expected: Errors in `storage/sql/loaders.go` referencing the old field. We fix those next.

**Step 3: Commit**

```bash
git add rubix/workspace.go
git commit -m "feat: replace single OIDCProvider with OIDCProviders slice on Workspace"
```

---

### Task 4: Update Provider interface

**Files:**
- Modify: `storage/provider.go`

**Step 1: Replace SetWorkspaceOIDCProvider with new OIDC methods**

In `storage/provider.go`, replace line 19:

```go
SetWorkspaceOIDCProvider(workspaceUuid string, provider rubix.OIDCProvider) error
```

With:

```go
GetOIDCProviders(workspace string) ([]rubix.OIDCProvider, error)
GetOIDCProvider(workspace, uuid string) (*rubix.OIDCProvider, error)
CreateOIDCProvider(workspace string, provider rubix.OIDCProvider) error
MutateOIDCProvider(workspace, uuid string, options ...rubix.MutateOIDCProviderOption) error
DeleteOIDCProvider(workspace, uuid string) error
```

**Step 2: Verify interface compiles (SQL provider won't satisfy yet)**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go build ./storage/`
Expected: Compilation errors — SQL provider doesn't implement the new methods yet. That's expected.

**Step 3: Commit**

```bash
git add storage/provider.go
git commit -m "feat: replace SetWorkspaceOIDCProvider with multi-provider CRUD interface"
```

---

### Task 5: Remove old OIDC column from workspace SQL queries

**Files:**
- Modify: `storage/sql/loaders.go:189-249`

**Step 1: Remove oidcProvider from workspace SELECT and scan**

In `storage/sql/loaders.go`, in the `retrieveWorkspacesByQuery` function:

1. Remove `oidcProvider` from the SELECT columns (line 191): change the query from:
   ```
   "SELECT uuid, alias, domain, name, icon, installedApplications,defaultApp,systemVendors,footerParts,accessCondition,oidcProvider,emailDomainWhitelist FROM workspaces WHERE "
   ```
   to:
   ```
   "SELECT uuid, alias, domain, name, icon, installedApplications,defaultApp,systemVendors,footerParts,accessCondition,emailDomainWhitelist FROM workspaces WHERE "
   ```

2. Remove the `oidcProviderJson` variable declaration (line 202):
   Delete: `oidcProviderJson := sql.NullString{}`

3. Remove `&oidcProviderJson` from the Scan call (line 207):
   Change: `scanErr := rows.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &icon, &installedApplicationsJson, &defaultApp, &sysVendors, &footerPartsJson, &accessConditionJson, &oidcProviderJson, &emailDomainWhitelistJson)`
   To: `scanErr := rows.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &icon, &installedApplicationsJson, &defaultApp, &sysVendors, &footerPartsJson, &accessConditionJson, &emailDomainWhitelistJson)`

4. Remove the oidcProvider unmarshal line (line 217):
   Delete: `json.Unmarshal([]byte(oidcProviderJson.String), &located.OIDCProvider)`

**Step 2: Remove the old SetWorkspaceOIDCProvider function**

Delete the entire `SetWorkspaceOIDCProvider` function (lines 238-249).

**Step 3: Verify it compiles**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go vet ./...`
Expected: Errors — Provider interface not satisfied. Fixed in next task.

**Step 4: Commit**

```bash
git add storage/sql/loaders.go
git commit -m "feat: remove oidcProvider column from workspace queries"
```

---

### Task 6: Implement OIDC provider CRUD in SQL provider

**Files:**
- Modify: `storage/sql/loaders.go` (add new functions at the end, before the Settings section)

**Step 1: Add the CRUD implementations**

Add the following functions to `storage/sql/loaders.go` (before the `GetSettings` function):

```go
// --- OIDC Providers ---
func (p *Provider) GetOIDCProviders(workspace string) ([]rubix.OIDCProvider, error) {
	rows, err := p.primaryConnection.Query(
		"SELECT uuid, workspace, providerName, clientID, clientSecret, clientKeys, issuerURL FROM workspace_oidc_providers WHERE workspace = ?",
		workspace,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []rubix.OIDCProvider
	for rows.Next() {
		var it rubix.OIDCProvider
		clientSecret := sql.NullString{}
		clientKeys := sql.NullString{}
		if err := rows.Scan(&it.Uuid, &it.Workspace, &it.ProviderName, &it.ClientID, &clientSecret, &clientKeys, &it.IssuerURL); err != nil {
			return nil, err
		}
		it.ClientSecret = clientSecret.String
		it.ClientKeys = clientKeys.String
		items = append(items, it)
	}
	return items, nil
}

func (p *Provider) GetOIDCProvider(workspace, uuid string) (*rubix.OIDCProvider, error) {
	row := p.primaryConnection.QueryRow(
		"SELECT uuid, workspace, providerName, clientID, clientSecret, clientKeys, issuerURL FROM workspace_oidc_providers WHERE workspace = ? AND uuid = ?",
		workspace, uuid,
	)
	var it rubix.OIDCProvider
	clientSecret := sql.NullString{}
	clientKeys := sql.NullString{}
	if err := row.Scan(&it.Uuid, &it.Workspace, &it.ProviderName, &it.ClientID, &clientSecret, &clientKeys, &it.IssuerURL); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, rubix.ErrNoResultFound
		}
		return nil, err
	}
	it.ClientSecret = clientSecret.String
	it.ClientKeys = clientKeys.String
	return &it, nil
}

func (p *Provider) CreateOIDCProvider(workspace string, provider rubix.OIDCProvider) error {
	_, err := p.primaryConnection.Exec(
		"INSERT INTO workspace_oidc_providers (uuid, workspace, providerName, clientID, clientSecret, clientKeys, issuerURL) VALUES (?, ?, ?, ?, ?, ?, ?)",
		provider.Uuid, workspace, provider.ProviderName, provider.ClientID, provider.ClientSecret, provider.ClientKeys, provider.IssuerURL,
	)
	if p.isDuplicateConflict(err) {
		return rubix.ErrDuplicate
	}
	if err != nil {
		return err
	}
	p.update()
	return nil
}

func (p *Provider) MutateOIDCProvider(workspace, uuid string, options ...rubix.MutateOIDCProviderOption) error {
	if len(options) == 0 {
		return nil
	}
	defer p.update()
	payload := rubix.MutateOIDCProviderPayload{}
	for _, opt := range options {
		opt(&payload)
	}
	var fields []string
	var vals []any
	if payload.ProviderName != nil {
		fields = append(fields, "providerName = ?")
		vals = append(vals, *payload.ProviderName)
	}
	if payload.ClientID != nil {
		fields = append(fields, "clientID = ?")
		vals = append(vals, *payload.ClientID)
	}
	if payload.ClientSecret != nil {
		fields = append(fields, "clientSecret = ?")
		vals = append(vals, *payload.ClientSecret)
	}
	if payload.ClientKeys != nil {
		fields = append(fields, "clientKeys = ?")
		vals = append(vals, *payload.ClientKeys)
	}
	if payload.IssuerURL != nil {
		fields = append(fields, "issuerURL = ?")
		vals = append(vals, *payload.IssuerURL)
	}
	if len(fields) == 0 {
		return nil
	}
	vals = append(vals, workspace, uuid)
	q := fmt.Sprintf("UPDATE workspace_oidc_providers SET %s WHERE workspace = ? AND uuid = ?", strings.Join(fields, ", "))
	res, err := p.primaryConnection.Exec(q, vals...)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return rubix.ErrNoResultFound
	}
	return nil
}

func (p *Provider) DeleteOIDCProvider(workspace, uuid string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM workspace_oidc_providers WHERE workspace = ? AND uuid = ?", workspace, uuid)
	if err != nil {
		return err
	}
	p.update()
	return nil
}
```

**Step 2: Verify it compiles**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go vet ./...`
Expected: PASS — all interface methods satisfied

**Step 3: Run existing tests**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go test ./storage/sql/ -run TestIntegration_SQLite_EndToEnd -v`
Expected: PASS

**Step 4: Commit**

```bash
git add storage/sql/loaders.go
git commit -m "feat: implement OIDC provider CRUD operations in SQL provider"
```

---

### Task 7: Add integration tests for OIDC provider CRUD

**Files:**
- Modify: `storage/sql/integration_test.go` (add test section at end of TestIntegration_SQLite_EndToEnd)

**Step 1: Add OIDC provider test cases**

Add the following block at the end of `TestIntegration_SQLite_EndToEnd`, before the closing `}`:

```go
// OIDC Providers
oidc1 := rubix.OIDCProvider{
    Uuid:         "oidc-1",
    ProviderName: "Okta",
    ClientID:     "client-abc",
    ClientSecret: "secret-xyz",
    IssuerURL:    "https://okta.example.com",
}
if err := p.CreateOIDCProvider(ws, oidc1); err != nil {
    t.Fatalf("CreateOIDCProvider: %v", err)
}
oidc2 := rubix.OIDCProvider{
    Uuid:         "oidc-2",
    ProviderName: "Auth0",
    ClientID:     "client-def",
    IssuerURL:    "https://auth0.example.com",
}
if err := p.CreateOIDCProvider(ws, oidc2); err != nil {
    t.Fatalf("CreateOIDCProvider 2: %v", err)
}

// Duplicate should return ErrDuplicate
if err := p.CreateOIDCProvider(ws, oidc1); err != rubix.ErrDuplicate {
    t.Fatalf("expected ErrDuplicate, got %v", err)
}

// GetOIDCProviders
providers, err := p.GetOIDCProviders(ws)
if err != nil || len(providers) != 2 {
    t.Fatalf("GetOIDCProviders: len=%d err=%v", len(providers), err)
}

// GetOIDCProvider
got, err := p.GetOIDCProvider(ws, "oidc-1")
if err != nil {
    t.Fatalf("GetOIDCProvider: %v", err)
}
if got.ProviderName != "Okta" || got.ClientID != "client-abc" || got.ClientSecret != "secret-xyz" || got.IssuerURL != "https://okta.example.com" {
    t.Fatalf("GetOIDCProvider mismatch: %+v", got)
}

// GetOIDCProvider not found
if _, err := p.GetOIDCProvider(ws, "nonexistent"); err != rubix.ErrNoResultFound {
    t.Fatalf("expected ErrNoResultFound, got %v", err)
}

// MutateOIDCProvider
if err := p.MutateOIDCProvider(ws, "oidc-1",
    rubix.WithOIDCProviderName("Okta SSO"),
    rubix.WithOIDCClientSecret("new-secret"),
); err != nil {
    t.Fatalf("MutateOIDCProvider: %v", err)
}
got, err = p.GetOIDCProvider(ws, "oidc-1")
if err != nil {
    t.Fatalf("GetOIDCProvider after mutate: %v", err)
}
if got.ProviderName != "Okta SSO" || got.ClientSecret != "new-secret" {
    t.Fatalf("MutateOIDCProvider mismatch: %+v", got)
}

// MutateOIDCProvider not found
if err := p.MutateOIDCProvider(ws, "nonexistent", rubix.WithOIDCProviderName("X")); err != rubix.ErrNoResultFound {
    t.Fatalf("expected ErrNoResultFound on mutate, got %v", err)
}

// DeleteOIDCProvider
if err := p.DeleteOIDCProvider(ws, "oidc-2"); err != nil {
    t.Fatalf("DeleteOIDCProvider: %v", err)
}
providers, err = p.GetOIDCProviders(ws)
if err != nil || len(providers) != 1 {
    t.Fatalf("GetOIDCProviders after delete: len=%d err=%v", len(providers), err)
}
if providers[0].Uuid != "oidc-1" {
    t.Fatalf("expected oidc-1 to remain, got %s", providers[0].Uuid)
}
```

**Step 2: Run the test**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go test ./storage/sql/ -run TestIntegration_SQLite_EndToEnd -v`
Expected: PASS

**Step 3: Commit**

```bash
git add storage/sql/integration_test.go
git commit -m "test: add integration tests for OIDC provider CRUD"
```

---

### Task 8: Final verification

**Step 1: Run all tests**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go test ./... -v -count=1`
Expected: All tests PASS

**Step 2: Run vet and build**

Run: `cd /Users/brooke.bryan/code/kubex/rubix-storage && go vet ./... && go build ./...`
Expected: No errors

**Step 3: Commit any remaining cleanup**

If any changes needed, commit them. Otherwise this task is a verification-only step.
