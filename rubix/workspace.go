package rubix

import (
	"encoding/json"
	"errors"

	"github.com/kubex/definitions-go/app"
)

type FooterPart struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// FooterParts is an ordered list of footer name/URL pairs.
// Supports unmarshaling from both the legacy map format {"name":"url"}
// and the current array format [{"name":"...","url":"..."}].
type FooterParts []FooterPart

func (fp FooterParts) ToMap() map[string]string {
	m := make(map[string]string, len(fp))
	for _, p := range fp {
		m[p.Name] = p.URL
	}
	return m
}

func (fp *FooterParts) UnmarshalJSON(data []byte) error {
	// Try array format first (current)
	var parts []FooterPart
	if err := json.Unmarshal(data, &parts); err == nil {
		*fp = parts
		return nil
	}
	// Fall back to legacy map format
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for name, url := range m {
		*fp = append(*fp, FooterPart{Name: name, URL: url})
	}
	return nil
}

type Workspace struct {
	Uuid                  string            `json:"uuid"`
	Alias                 string            `json:"alias"`
	Domain                string            `json:"domain"`
	Name                  string            `json:"name"`
	Icon                  string            `json:"icon"`
	InstalledApplications []app.ScopedKey   `json:"installedApplications"`
	SystemVendors         []string          `json:"systemVendors"`
	DefaultApp            app.GlobalAppID   `json:"defaultApp"`
	FooterParts           FooterParts       `json:"footerParts"`
	AccessCondition       Condition         `json:"accessCondition"`
	OIDCProviders         []OIDCProvider    `json:"oidcProviders"`
	EmailDomainWhitelist  []string          `json:"emailDomainWhitelist"`
	EmailDomainApproval   map[string]string `json:"emailDomainApproval"`
	MemberApprovalMode    string            `json:"memberApprovalMode"`
}

func WorkspaceFromJson(jsonBytes []byte) (*Workspace, error) {
	w := &Workspace{}
	if err := json.Unmarshal(jsonBytes, w); err != nil {
		return nil, errors.New("unable to decode workspace json")
	}
	return w, nil
}
