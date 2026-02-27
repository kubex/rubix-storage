package rubix

import (
	"encoding/json"
	"errors"

	"github.com/kubex/definitions-go/app"
)

type MetricTicker struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// MetricTickers is an ordered list of metric ticker name/URL pairs.
// Supports unmarshaling from both the legacy map format {"name":"url"}
// and the current array format [{"name":"...","url":"..."}].
type MetricTickers []MetricTicker

func (mt MetricTickers) ToMap() map[string]string {
	m := make(map[string]string, len(mt))
	for _, t := range mt {
		m[t.Name] = t.URL
	}
	return m
}

func (mt *MetricTickers) UnmarshalJSON(data []byte) error {
	// Try array format first (current)
	var tickers []MetricTicker
	if err := json.Unmarshal(data, &tickers); err == nil {
		*mt = tickers
		return nil
	}
	// Fall back to legacy map format
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	for name, url := range m {
		*mt = append(*mt, MetricTicker{Name: name, URL: url})
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
	MetricTickers         MetricTickers     `json:"metricTickers"`
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
