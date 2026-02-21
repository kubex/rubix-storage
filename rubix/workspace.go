package rubix

import (
	"encoding/json"
	"errors"

	"github.com/kubex/definitions-go/app"
)

type Workspace struct {
	Uuid                  string            `json:"uuid"`
	Alias                 string            `json:"alias"`
	Domain                string            `json:"domain"`
	Name                  string            `json:"name"`
	Icon                  string            `json:"icon"`
	InstalledApplications []app.ScopedKey   `json:"installedApplications"`
	SystemVendors         []string          `json:"systemVendors"`
	DefaultApp            app.GlobalAppID   `json:"defaultApp"`
	FooterParts           map[string]string `json:"footerParts"`
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
