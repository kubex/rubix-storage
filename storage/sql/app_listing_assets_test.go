package sql

import (
	"bytes"
	"errors"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestAppListingAssets(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	png := []byte("\x89PNG\r\n\x1a\nfake")
	for _, name := range []string{"one.png", "two.png"} {
		if err := p.StoreAppListingAsset(rubix.AppListingAsset{VendorID: "acme", AppID: "crm", Name: name, ContentType: "image/png", Data: png}); err != nil {
			t.Fatal(err)
		}
	}
	got, err := p.GetAppListingAsset("acme", "crm", "one.png")
	if err != nil || got.ContentType != "image/png" || !bytes.Equal(got.Data, png) {
		t.Fatalf("asset: %+v %v", got, err)
	}
	if err := p.PruneAppListingAssets("acme", "crm", []string{"two.png"}); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetAppListingAsset("acme", "crm", "one.png"); !errors.Is(err, rubix.ErrNoResultFound) {
		t.Fatal("pruned asset still stored", err)
	}
	if _, err := p.GetAppListingAsset("acme", "crm", "two.png"); err != nil {
		t.Fatal("kept asset pruned", err)
	}

	for name, bad := range map[string]rubix.AppListingAsset{
		"svg":       {VendorID: "acme", AppID: "crm", Name: "x.svg", ContentType: "image/svg+xml", Data: png},
		"path":      {VendorID: "acme", AppID: "crm", Name: "../x.png", ContentType: "image/png", Data: png},
		"too large": {VendorID: "acme", AppID: "crm", Name: "x.png", ContentType: "image/png", Data: make([]byte, rubix.AppListingMaxAssetBytes+1)},
	} {
		if err := p.StoreAppListingAsset(bad); !errors.Is(err, rubix.ErrAppListing) {
			t.Errorf("%s accepted: %v", name, err)
		}
	}

	listing := rubix.AppListing{VendorID: "acme", AppID: "crm", Screenshots: []rubix.AppListingScreenshot{{URL: "asset:two.png"}}}
	if err := p.StoreAppListing(listing); err != nil {
		t.Fatal("asset screenshot rejected:", err)
	}
}
