package sql

import (
	"errors"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestAppListings(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	if _, err := p.GetAppListing("acme", "crm"); !errors.Is(err, rubix.ErrNoResultFound) {
		t.Fatalf("missing listing: %v", err)
	}

	listing := rubix.AppListing{
		VendorID:    "acme",
		AppID:       "crm",
		Tagline:     "Every customer in one place",
		Overview:    "First paragraph.\n\nSecond paragraph.",
		Highlights:  []rubix.AppListingHighlight{{Icon: "users@lu", Title: "Shared timeline", Text: "Everyone sees the same history."}},
		Screenshots: []rubix.AppListingScreenshot{{URL: "https://cdn.example.com/crm/1.png", Caption: "Timeline"}},
		Scopes:      []rubix.AppListingScope{{Key: "customers.write", Description: "Update customer records", Sensitive: true}},
		Languages:   []string{"English"},
		DocsURL:     "https://docs.example.com/crm",
		UpdatedBy:   "user-1",
	}
	if err := p.StoreAppListing(listing); err != nil {
		t.Fatal(err)
	}
	listing.Tagline = "Every customer, one timeline"
	if err := p.StoreAppListing(listing); err != nil {
		t.Fatal("update:", err)
	}

	got, err := p.GetAppListing("acme", "crm")
	if err != nil {
		t.Fatal(err)
	}
	if got.Tagline != listing.Tagline || got.UpdatedBy != "user-1" || got.UpdatedAt.IsZero() ||
		len(got.Screenshots) != 1 || !got.Scopes[0].Sensitive || len(got.OverviewParagraphs()) != 2 {
		t.Fatalf("round trip: %+v", got)
	}

	all, err := p.GetAppListings()
	if err != nil || len(all) != 1 || all[0].AppID != "crm" {
		t.Fatalf("list: %+v %v", all, err)
	}

	for name, bad := range map[string]rubix.AppListing{
		"http screenshot": {VendorID: "acme", AppID: "crm", Screenshots: []rubix.AppListingScreenshot{{URL: "http://cdn.example.com/1.png"}}},
		"untitled":        {VendorID: "acme", AppID: "crm", Highlights: []rubix.AppListingHighlight{{Text: "No title"}}},
		"no app":          {VendorID: "acme"},
	} {
		if err := p.StoreAppListing(bad); !errors.Is(err, rubix.ErrAppListing) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}
}

func TestAppInstallCounts(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	for _, ws := range []string{"ws-1", "ws-2"} {
		if err := p.SetWorkspaceApplication(ws, "acme", "crm", ""); err != nil {
			t.Fatal(err)
		}
	}
	if err := p.SetWorkspaceApplication("ws-1", "acme", "billing", ""); err != nil {
		t.Fatal(err)
	}
	counts, err := p.GetAppInstallCounts()
	if err != nil || counts["acme/crm"] != 2 || counts["acme/billing"] != 1 {
		t.Fatalf("counts: %v %v", counts, err)
	}
}
