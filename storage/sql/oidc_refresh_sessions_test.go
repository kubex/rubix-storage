package sql

import (
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestOIDCProviderRefreshSessionsRoundTrip(t *testing.T) {
	p := newTestProvider(t)
	defer func() { _ = p.Close() }()

	ws := "ws-refresh"
	if err := p.CreateWorkspace(ws, "Workspace", "refresh", "refresh.local"); err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	on := rubix.OIDCProvider{Uuid: "oidc-on", ProviderName: "Okta", ClientID: "c", ClientSecret: "s", IssuerURL: "https://okta.example.com", RefreshSessions: true}
	off := rubix.OIDCProvider{Uuid: "oidc-off", ProviderName: "Okta", ClientID: "c", ClientSecret: "s", IssuerURL: "https://okta.example.com"}
	for _, provider := range []rubix.OIDCProvider{on, off} {
		if err := p.CreateOIDCProvider(ws, provider); err != nil {
			t.Fatalf("CreateOIDCProvider %s: %v", provider.Uuid, err)
		}
	}

	providers, err := p.GetOIDCProviders(ws)
	if err != nil {
		t.Fatalf("GetOIDCProviders: %v", err)
	}
	listed := map[string]bool{}
	for _, provider := range providers {
		listed[provider.Uuid] = provider.RefreshSessions
	}
	if len(listed) != 2 || !listed["oidc-on"] || listed["oidc-off"] {
		t.Fatalf("GetOIDCProviders RefreshSessions = %v", listed)
	}

	if err := p.MutateOIDCProvider(ws, "oidc-on", rubix.WithOIDCRefreshSessions(false)); err != nil {
		t.Fatalf("MutateOIDCProvider: %v", err)
	}
	got, err := p.GetOIDCProvider(ws, "oidc-on")
	if err != nil {
		t.Fatalf("GetOIDCProvider: %v", err)
	}
	if got.RefreshSessions {
		t.Fatal("RefreshSessions should be off after mutate")
	}
	if err := p.MutateOIDCProvider(ws, "oidc-on", rubix.WithOIDCRefreshSessions(true)); err != nil {
		t.Fatalf("MutateOIDCProvider: %v", err)
	}
	if err := p.MutateOIDCProvider(ws, "oidc-on", rubix.WithOIDCDisplayName("Other")); err != nil {
		t.Fatalf("MutateOIDCProvider unrelated: %v", err)
	}
	if got, _ := p.GetOIDCProvider(ws, "oidc-on"); !got.RefreshSessions {
		t.Fatal("an unrelated mutate should leave RefreshSessions on")
	}
}
