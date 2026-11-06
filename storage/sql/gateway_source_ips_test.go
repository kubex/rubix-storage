package sql

import (
	"errors"
	"github.com/kubex/rubix-storage/rubix"
	"testing"
)

func TestGatewaySourceIPScopes(t *testing.T) {
	for _, vendor := range []string{"acme", "kx"} {
		t.Run(vendor, func(t *testing.T) { testGatewaySourceIPScopes(t, vendor) })
	}
}

func testGatewaySourceIPScopes(t *testing.T, vendor string) {
	p := newTestProvider(t)
	defer p.Close()
	entries := []rubix.GatewaySourceIP{
		{Scope: rubix.GatewaySourceGlobal, Network: "2001:db8::1"},
		{Scope: rubix.GatewaySourceVendor, VendorID: vendor, Network: "203.0.113.20/24"},
		{Scope: rubix.GatewaySourceApp, VendorID: vendor, AppID: "orders", Network: "198.51.100.5"},
		{Scope: rubix.GatewaySourceApp, VendorID: vendor, AppID: "billing", Network: "198.51.100.6"},
		{Scope: rubix.GatewaySourceVendor, VendorID: "other", Network: "192.0.2.1"},
	}
	for i, e := range entries {
		e.CreatedBy = "admin"
		var err error
		entries[i], err = p.AddGatewaySourceIP(e)
		if err != nil {
			t.Fatal(err)
		}
	}
	effective, err := p.GetEffectiveGatewaySourceIPs(vendor, "orders")
	if err != nil || len(effective) != 3 {
		t.Fatalf("scope union: %+v %v", effective, err)
	}
	for _, e := range effective {
		if e.VendorID == "other" || e.AppID == "billing" {
			t.Fatal("another app's source IP leaked")
		}
	}
	appEntry := entries[2]
	if err := p.DeleteGatewaySourceIP(rubix.GatewaySourceApp, vendor, "billing", appEntry.ID); !errors.Is(err, rubix.ErrGatewaySourceIPNotFound) {
		t.Fatal("cross-app deletion accepted", err)
	}
	if err := p.DeleteGatewaySourceIP(rubix.GatewaySourceApp, vendor, "orders", entries[0].ID); !errors.Is(err, rubix.ErrGatewaySourceIPNotFound) {
		t.Fatal("global deletion through app scope accepted", err)
	}
	if err := p.DeleteGatewaySourceIP(appEntry.Scope, appEntry.VendorID, appEntry.AppID, appEntry.ID); err != nil {
		t.Fatal(err)
	}
	effective, err = p.GetEffectiveGatewaySourceIPs(vendor, "orders")
	if err != nil || len(effective) != 2 {
		t.Fatal("deletion not reflected", err)
	}
	invalid := rubix.GatewaySourceIP{Scope: rubix.GatewaySourceGlobal, VendorID: vendor, Network: "203.0.113.1", CreatedBy: "admin"}
	if _, err := p.AddGatewaySourceIP(invalid); !errors.Is(err, rubix.ErrGatewaySourceIP) {
		t.Fatal("invalid scope tuple accepted", err)
	}
}
