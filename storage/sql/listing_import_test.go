package sql

import (
	"errors"
	"os"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
	"github.com/openbyte-os/sdk-go/app"
)

func TestListingImportRoundTrip(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	files := map[string][]byte{"screenshots/timeline.png": png}
	fetch := func(path string) ([]byte, error) {
		if data, ok := files[path]; ok {
			return data, nil
		}
		return nil, os.ErrNotExist
	}

	in := app.Listing{
		Tagline:     "Every customer in one place",
		Highlights:  []app.ListingHighlight{{Title: "Shared timeline"}},
		Screenshots: []app.ListingScreenshot{{URL: "https://cdn.example.com/1.png", Caption: "Hosted"}, {URL: "screenshots/timeline.png", Caption: "Packaged"}},
		Releases:    []app.ListingRelease{{Version: "2.0.0", Notes: []string{"New timeline", " "}}},
	}
	imp, err := rubix.ImportListing("acme", "crm", in, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if len(imp.Assets) != 1 || imp.Listing.Screenshots[1].URL != "asset:screenshot-2.png" || len(imp.Releases[0].Notes) != 1 {
		t.Fatalf("import: %+v", imp)
	}
	if err := imp.Apply(p); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetAppListingAsset("acme", "crm", "screenshot-2.png"); err != nil {
		t.Fatal("asset not stored", err)
	}
	releases, _ := p.GetAppReleases("acme", "crm")
	if len(releases) != 1 || releases[0].Version != "2.0.0" {
		t.Fatalf("releases: %+v", releases)
	}

	// Dropping the packaged screenshot removes its image.
	in.Screenshots = in.Screenshots[:1]
	imp, err = rubix.ImportListing("acme", "crm", in, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if err := imp.Apply(p); err != nil {
		t.Fatal(err)
	}
	if _, err := p.GetAppListingAsset("acme", "crm", "screenshot-2.png"); !errors.Is(err, rubix.ErrNoResultFound) {
		t.Fatal("unused asset kept", err)
	}

	stored, _ := p.GetAppListing("acme", "crm")
	stored.Screenshots = append(stored.Screenshots, rubix.AppListingScreenshot{URL: "asset:logo.png"})
	out := rubix.ExportListing(*stored, releases, func(name string) string { return "screenshots/" + name })
	if out.Tagline != in.Tagline || out.Screenshots[1].URL != "screenshots/logo.png" || out.Releases[0].Version != "2.0.0" {
		t.Fatalf("export: %+v", out)
	}

	if _, err := rubix.ImportListing("acme", "crm", app.Listing{Screenshots: []app.ListingScreenshot{{URL: "missing.png"}}}, fetch); !errors.Is(err, rubix.ErrAppListing) {
		t.Fatal("missing screenshot file accepted", err)
	}
	files["notes.txt"] = []byte("not an image")
	if _, err := rubix.ImportListing("acme", "crm", app.Listing{Screenshots: []app.ListingScreenshot{{URL: "notes.txt"}}}, fetch); !errors.Is(err, rubix.ErrAppListing) {
		t.Fatal("non-image screenshot accepted", err)
	}
}

func TestListingImportBrand(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	fetch := func(path string) ([]byte, error) {
		if path == "cover.png" || path == "assets/icon.png" || path == "hero.png" {
			return png, nil
		}
		return nil, os.ErrNotExist
	}

	in := app.Listing{Brand: app.ListingBrand{Color: " #7557F5", Pattern: app.PatternWaves, Cover: "cover.png", Icon: "assets/icon.png", Hero: "hero.png"}}
	imp, err := rubix.ImportListing("acme", "crm", in, fetch)
	if err != nil {
		t.Fatal(err)
	}
	want := rubix.AppListingBrand{Color: "#7557f5", Pattern: "waves", Cover: "asset:cover.png", Icon: "asset:icon.png", Hero: "asset:hero.png"}
	if imp.Listing.Brand != want || len(imp.Assets) != 3 {
		t.Fatalf("brand: %+v assets %d", imp.Listing.Brand, len(imp.Assets))
	}
	if err := imp.Apply(p); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"cover.png", "icon.png", "hero.png"} {
		if _, err := p.GetAppListingAsset("acme", "crm", name); err != nil {
			t.Fatal(name+" not kept", err)
		}
	}
	stored, err := p.GetAppListing("acme", "crm")
	if err != nil || stored.Brand != want {
		t.Fatalf("stored brand: %+v %v", stored, err)
	}
	out := rubix.ExportListing(*stored, nil, func(name string) string { return "listing/" + name })
	if out.Brand.Cover != "listing/cover.png" || out.Brand.Icon != "listing/icon.png" || out.Brand.Hero != "listing/hero.png" || out.Brand.Pattern != app.PatternWaves {
		t.Fatalf("export: %+v", out.Brand)
	}

	for _, bad := range []app.ListingBrand{{Color: "purple"}, {Color: "#fff"}, {Pattern: "tartan"}, {Cover: "http://cdn.example.com/c.png"}, {Icon: "javascript:alert(1)"}, {Hero: "data:image/png;base64,AA"}} {
		if _, err := rubix.ImportListing("acme", "crm", app.Listing{Brand: bad}, nil); !errors.Is(err, rubix.ErrAppListing) {
			t.Errorf("%+v accepted: %v", bad, err)
		}
	}
}
