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
	InstalledApplications []app.GlobalAppID `json:"installedApplications"`
}

func WorkspaceFromJson(jsonBytes []byte) (*Workspace, error) {
	w := &Workspace{}
	if err := json.Unmarshal(jsonBytes, w); err != nil {
		return nil, errors.New("unable to decode workspace json")
	}
	return w, nil
}
