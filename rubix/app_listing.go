package rubix

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/openbyte-os/sdk-go/app"
)

// AppListing is an app's marketplace presentation, written by its vendor in
// the developer centre and read by the marketplace. One listing per app.
type AppListing struct {
	VendorID string `json:"vendorID"`
	AppID    string `json:"appID"`

	// Tagline is the one-line pitch shown on cards and under the app name.
	Tagline string `json:"tagline,omitempty"`
	// Overview is the long description; paragraphs are separated by blank lines.
	Overview    string                 `json:"overview,omitempty"`
	Highlights  []AppListingHighlight  `json:"highlights,omitempty"`
	Screenshots []AppListingScreenshot `json:"screenshots,omitempty"`
	// Scopes are the data access the app asks a workspace to approve at install.
	Scopes []AppListingScope `json:"scopes,omitempty"`
	// Languages are display names, e.g. "English".
	Languages []string `json:"languages,omitempty"`
	DocsURL   string   `json:"docsURL,omitempty"`
	// Brand styles the marketplace card cover and app page banner.
	Brand AppListingBrand `json:"brand,omitzero"`

	// ManagedByApp listings are synced from the app's /_kubex/listing each time
	// a new version is seen running; SyncedVersion is the version last read.
	ManagedByApp  bool   `json:"managedByApp,omitempty"`
	SyncedVersion string `json:"syncedVersion,omitempty"`

	UpdatedBy string    `json:"updatedBy,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
}

// AppListingAsset is an image uploaded for a listing, referenced from a
// screenshot URL as "asset:<Name>".
type AppListingAsset struct {
	VendorID    string
	AppID       string
	Name        string
	ContentType string
	Data        []byte
	UpdatedAt   time.Time
}

const (
	AppListingAssetPrefix   = "asset:"
	AppListingMaxAssetBytes = 5 << 20
)

// AppListingAssetTypes are the image types a listing asset may hold.
var AppListingAssetTypes = map[string]bool{"image/png": true, "image/jpeg": true, "image/webp": true, "image/gif": true}

// AssetName returns the stored asset a screenshot URL refers to, if any.
func (s AppListingScreenshot) AssetName() (string, bool) {
	name, ok := strings.CutPrefix(s.URL, AppListingAssetPrefix)
	return name, ok && ValidAppListingAssetName(name)
}

// ValidAppListingAssetName is 1-100 letters, digits, dots, dashes or underscores, not starting with a dot.
func ValidAppListingAssetName(name string) bool {
	if name == "" || len(name) > 100 || name[0] == '.' {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '.' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func (a AppListingAsset) Validate() error {
	switch {
	case a.VendorID == "" || a.AppID == "":
		return fmt.Errorf("%w: vendor and app are required", ErrAppListing)
	case !ValidAppListingAssetName(a.Name):
		return fmt.Errorf("%w: invalid image name %q", ErrAppListing, a.Name)
	case !AppListingAssetTypes[a.ContentType]:
		return fmt.Errorf("%w: %s must be a PNG, JPEG, WebP or GIF image", ErrAppListing, a.Name)
	case len(a.Data) == 0 || len(a.Data) > AppListingMaxAssetBytes:
		return fmt.Errorf("%w: %s must be up to %d MB", ErrAppListing, a.Name, AppListingMaxAssetBytes>>20)
	}
	return nil
}

// AppListingBrand is the app's marketplace look. Color is #RRGGBB, Pattern is
// one of app.ListingPatterns, and Cover and Icon are https image URLs or
// "asset:<Name>". Icon replaces the app's icon on marketplace tiles only, and
// Hero is the large banner shown when the app is promoted on the store front.
type AppListingBrand struct {
	Color   string `json:"color,omitempty"`
	Pattern string `json:"pattern,omitempty"`
	Cover   string `json:"cover,omitempty"`
	Icon    string `json:"icon,omitempty"`
	Hero    string `json:"hero,omitempty"`
}

// CoverAssetName returns the stored asset the cover refers to, if any.
func (b AppListingBrand) CoverAssetName() (string, bool) { return assetRef(b.Cover) }

// IconAssetName returns the stored asset the icon refers to, if any.
func (b AppListingBrand) IconAssetName() (string, bool) { return assetRef(b.Icon) }

// HeroAssetName returns the stored asset the hero image refers to, if any.
func (b AppListingBrand) HeroAssetName() (string, bool) { return assetRef(b.Hero) }

func assetRef(url string) (string, bool) {
	name, ok := strings.CutPrefix(url, AppListingAssetPrefix)
	return name, ok && ValidAppListingAssetName(name)
}

// ValidBrandColor is a six-digit hex colour, e.g. "#7557f5".
func ValidBrandColor(c string) bool {
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for _, r := range c[1:] {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}

// ValidBrandPattern is one of app.ListingPatterns.
func ValidBrandPattern(p string) bool {
	for _, known := range app.ListingPatterns {
		if string(known) == p {
			return true
		}
	}
	return false
}

type AppListingHighlight struct {
	Icon  string `json:"icon,omitempty"`
	Title string `json:"title"`
	Text  string `json:"text,omitempty"`
}

// AppListingScreenshot URL is the full-size https image.
type AppListingScreenshot struct {
	URL     string `json:"url"`
	Caption string `json:"caption,omitempty"`
}

type AppListingScope struct {
	Key         string `json:"key"`
	Description string `json:"description"`
	Sensitive   bool   `json:"sensitive,omitempty"`
}

// Listing limits, enforced on store.
const (
	AppListingMaxTagline     = 140
	AppListingMaxOverview    = 4000
	AppListingMaxHighlights  = 6
	AppListingMaxScreenshots = 8
	AppListingMaxScopes      = 30
	AppListingMaxLanguages   = 20
	appListingMaxShort       = 120
	appListingMaxText        = 300
	appListingMaxURL         = 2048
)

var ErrAppListing = errors.New("invalid app listing")

// OverviewParagraphs splits the overview on blank lines.
func (l AppListing) OverviewParagraphs() []string {
	var out []string
	for _, p := range strings.Split(strings.ReplaceAll(l.Overview, "\r\n", "\n"), "\n\n") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Validate reports the first field that breaks the listing limits.
func (l AppListing) Validate() error {
	fail := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrAppListing, fmt.Sprintf(format, args...))
	}
	switch {
	case l.VendorID == "" || l.AppID == "":
		return fail("vendor and app are required")
	case len(l.Tagline) > AppListingMaxTagline:
		return fail("tagline is longer than %d characters", AppListingMaxTagline)
	case len(l.Overview) > AppListingMaxOverview:
		return fail("overview is longer than %d characters", AppListingMaxOverview)
	case len(l.Highlights) > AppListingMaxHighlights:
		return fail("more than %d highlights", AppListingMaxHighlights)
	case len(l.Screenshots) > AppListingMaxScreenshots:
		return fail("more than %d screenshots", AppListingMaxScreenshots)
	case len(l.Scopes) > AppListingMaxScopes:
		return fail("more than %d data access entries", AppListingMaxScopes)
	case len(l.Languages) > AppListingMaxLanguages:
		return fail("more than %d languages", AppListingMaxLanguages)
	}
	for _, h := range l.Highlights {
		if strings.TrimSpace(h.Title) == "" || len(h.Title) > appListingMaxShort || len(h.Text) > appListingMaxText || len(h.Icon) > 64 {
			return fail("each highlight needs a title of up to %d characters and text of up to %d", appListingMaxShort, appListingMaxText)
		}
	}
	for _, s := range l.Screenshots {
		if _, isAsset := s.AssetName(); !isAsset && !secureURL(s.URL) {
			return fail("screenshot %q must be an https URL or an uploaded image", s.URL)
		}
		if len(s.Caption) > appListingMaxShort {
			return fail("screenshot captions are up to %d characters", appListingMaxShort)
		}
	}
	for _, s := range l.Scopes {
		if strings.TrimSpace(s.Key) == "" || strings.TrimSpace(s.Description) == "" || len(s.Key) > appListingMaxShort || len(s.Description) > appListingMaxText {
			return fail("each data access entry needs a key and a description")
		}
	}
	for _, lang := range l.Languages {
		if strings.TrimSpace(lang) == "" || len(lang) > 64 {
			return fail("languages are names of up to 64 characters")
		}
	}
	if b := l.Brand; b.Color != "" && !ValidBrandColor(b.Color) {
		return fail("brand colour %q must be a hex colour like #7557f5", b.Color)
	}
	if b := l.Brand; b.Pattern != "" && !ValidBrandPattern(b.Pattern) {
		return fail("brand pattern %q is not one of %v", b.Pattern, app.ListingPatterns)
	}
	for kind, url := range map[string]string{"cover": l.Brand.Cover, "icon": l.Brand.Icon, "hero": l.Brand.Hero} {
		if _, isAsset := assetRef(url); url != "" && !isAsset && !secureURL(url) {
			return fail("%s %q must be an https URL or an uploaded image", kind, url)
		}
	}
	if l.DocsURL != "" && !secureURL(l.DocsURL) {
		return fail("documentation link must be an https URL")
	}
	return nil
}

func secureURL(raw string) bool {
	if len(raw) > appListingMaxURL {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
