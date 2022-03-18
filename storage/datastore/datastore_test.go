package datastore

import (
	"log"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestDataStore(t *testing.T) {
	p := Provider{ProjectID: "test-project"}
	p.Init()

	log.Println(p.GetWorkspaceUUIDByAlias("alias"))

	w := &rubix.Workspace{
		Uuid:   "random-workspace",
		Alias:  "alias",
		Domain: "alias.cubex-local.com",
		Name:   "Alias",
	}
	p.StoreWorkspace(w)

}
