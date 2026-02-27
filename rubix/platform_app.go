package rubix

// PlatformApplication represents an installed application at the platform level.
type PlatformApplication struct {
	VendorID           string   `json:"vendorID"`
	AppID              string   `json:"appID"`
	ReleaseChannel     string   `json:"releaseChannel"`
	SignatureKey       string   `json:"signatureKey"`
	Endpoint           string   `json:"endpoint"`
	SimpleApp          bool     `json:"simpleApp"`
	Framed             bool     `json:"framed"`
	AllowScripts       bool     `json:"allowScripts"`
	CookiePassthrough  []string `json:"cookiePassthrough"`
	GloballyAvailable  bool     `json:"globallyAvailable"`
	AllowedWorkspaces  []string `json:"allowedWorkspaces"`
	WorkspaceAvailable bool     `json:"workspaceAvailable"`
}

// PlatformVendor represents a vendor registered at the platform level.
type PlatformVendor struct {
	VendorID    string `json:"vendorID"`
	Name        string `json:"name"`
	Description string `json:"description"`
	LogoURL     string `json:"logoURL"`
	Icon        string `json:"icon"`
	Discovery   string `json:"discovery"`
}
