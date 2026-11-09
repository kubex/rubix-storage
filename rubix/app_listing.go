package rubix

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
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
