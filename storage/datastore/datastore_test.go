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
	log.Println(p.GetUserWorkspaceUUIDs("anonymous"))
	log.Println(p.GetWorkspaceMembers("random-workspace", ""))
	uuid := "rumble-uuid"
	alias := "rumble"
	w := &rubix.Workspace{
		Uuid:   uuid,
		Alias:  alias,
		Domain: alias + ".cubex-local.com",
		Name:   alias + " WorkSpace",
	}
	p.StoreWorkspace(w)

	p.AddMembership(uuid, "anonymous", rubix.MembershipTypeOwner)

}
