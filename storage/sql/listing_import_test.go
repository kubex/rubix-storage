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
