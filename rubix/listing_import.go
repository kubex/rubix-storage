package rubix

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/openbyte-os/sdk-go/app"
)

// ListingImport is an SDK listing converted for storage: the listing, the
// screenshot images it carried, and its release notes.
type ListingImport struct {
	Listing  AppListing
	Assets   []AppListingAsset
	Releases []AppRelease
}

// ImportListing converts an app.Listing (from listing.json or an app's
// /_kubex/listing). Screenshots given as https URLs or existing "asset:" names
// are kept; any other URL is a file path read with fetch and stored as an image.
func ImportListing(vendorID, appID string, in app.Listing, fetch func(path string) ([]byte, error)) (ListingImport, error) {
	out := ListingImport{Listing: AppListing{
		VendorID: vendorID, AppID: appID,
		Tagline: strings.TrimSpace(in.Tagline), Overview: strings.TrimSpace(in.Overview),
		Languages: in.Languages, DocsURL: strings.TrimSpace(in.DocsURL),
	}}
	for _, h := range in.Highlights {
		out.Listing.Highlights = append(out.Listing.Highlights, AppListingHighlight{Icon: h.Icon, Title: h.Title, Text: h.Text})
	}
	for _, s := range in.Scopes {
		out.Listing.Scopes = append(out.Listing.Scopes, AppListingScope{Key: s.Key, Description: s.Description, Sensitive: s.Sensitive})
	}
	if len(in.Screenshots) > AppListingMaxScreenshots {
		return out, fmt.Errorf("%w: more than %d screenshots", ErrAppListing, AppListingMaxScreenshots)
	}
	for i, s := range in.Screenshots {
		shot := AppListingScreenshot{URL: strings.TrimSpace(s.URL), Caption: s.Caption}
		if _, isAsset := shot.AssetName(); !isAsset && !secureURL(shot.URL) {
			if fetch == nil {
				return out, fmt.Errorf("%w: screenshot %q must be an https URL", ErrAppListing, shot.URL)
			}
			data, err := fetch(shot.URL)
			if err != nil {
				return out, fmt.Errorf("%w: screenshot %q: %v", ErrAppListing, shot.URL, err)
			}
			asset := AppListingAsset{VendorID: vendorID, AppID: appID, Data: data, ContentType: http.DetectContentType(data)}
			asset.Name = "screenshot-" + strconv.Itoa(i+1) + assetExtension(asset.ContentType, shot.URL)
			if err := asset.Validate(); err != nil {
				return out, err
			}
			out.Assets = append(out.Assets, asset)
			shot.URL = AppListingAssetPrefix + asset.Name
		}
		out.Listing.Screenshots = append(out.Listing.Screenshots, shot)
	}
	if err := out.Listing.Validate(); err != nil {
		return out, err
	}
	for _, r := range in.Releases {
		release := AppRelease{VendorID: vendorID, AppID: appID, Version: strings.TrimSpace(r.Version)}
		for _, n := range r.Notes {
			if n = strings.TrimSpace(n); n != "" {
				release.Notes = append(release.Notes, n)
			}
		}
		if err := release.Validate(); err != nil {
			return out, err
		}
		out.Releases = append(out.Releases, release)
	}
	return out, nil
}

// ExportListing converts a stored listing and its release notes to the SDK
// format. assetURL maps an uploaded image name to the URL written out.
func ExportListing(l AppListing, releases []AppRelease, assetURL func(name string) string) app.Listing {
	out := app.Listing{Tagline: l.Tagline, Overview: l.Overview, Languages: l.Languages, DocsURL: l.DocsURL}
	for _, h := range l.Highlights {
		out.Highlights = append(out.Highlights, app.ListingHighlight{Icon: h.Icon, Title: h.Title, Text: h.Text})
	}
	for _, s := range l.Screenshots {
		url := s.URL
		if name, ok := s.AssetName(); ok && assetURL != nil {
			url = assetURL(name)
		}
		out.Screenshots = append(out.Screenshots, app.ListingScreenshot{URL: url, Caption: s.Caption})
	}
	for _, s := range l.Scopes {
		out.Scopes = append(out.Scopes, app.ListingScope{Key: s.Key, Description: s.Description, Sensitive: s.Sensitive})
	}
	for _, r := range releases {
		out.Releases = append(out.Releases, app.ListingRelease{Version: r.Version, Notes: r.Notes})
	}
	return out
}

// ReferencedAssets lists the uploaded images a listing's screenshots use.
func (l AppListing) ReferencedAssets() []string {
	var names []string
	for _, s := range l.Screenshots {
		if name, ok := s.AssetName(); ok {
			names = append(names, name)
		}
	}
	return names
}

func assetExtension(contentType, source string) string {
	switch contentType {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	}
	return path.Ext(source)
}

// ListingStore is the storage an import is applied to; storage.Provider satisfies it.
type ListingStore interface {
	StoreAppListingAsset(asset AppListingAsset) error
	StoreAppListing(listing AppListing) error
	PruneAppListingAssets(vendorID, appID string, keep []string) error
	StoreAppRelease(release AppRelease) error
}

// Apply stores the import: its images, the listing, and its release notes.
// Images the listing no longer uses are removed.
func (imp ListingImport) Apply(s ListingStore) error {
	for _, asset := range imp.Assets {
		if err := s.StoreAppListingAsset(asset); err != nil {
			return err
		}
	}
	if err := s.StoreAppListing(imp.Listing); err != nil {
		return err
	}
	if err := s.PruneAppListingAssets(imp.Listing.VendorID, imp.Listing.AppID, imp.Listing.ReferencedAssets()); err != nil {
		return err
	}
	for _, release := range imp.Releases {
		if err := s.StoreAppRelease(release); err != nil {
			return err
		}
	}
	return nil
}
