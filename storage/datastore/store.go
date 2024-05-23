package datastore

import (
	"context"
	"encoding/json"

	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) StoreWorkspace(w *rubix.Workspace) error {
	ws := &workspaceStore{
		Uuid:   w.Uuid,
		Alias:  w.Alias,
		Name:   w.Name,
		Domain: w.Domain,
	}
	ws.InstalledApplications, _ = json.Marshal(w.InstalledApplications)

	_, err := p.client.Put(context.Background(), ws.dsID(), ws)
	return err
}

func (p Provider) AddMembership(workspaceUUID, identityID string, Role rubix.MembershipType) error {
	mem := &workspaceMembership{
		WorkspaceUUID: workspaceUUID,
		IdentityID:    identityID,
		Role:          Role,
	}
	_, err := p.client.Put(context.Background(), mem.dsID(), mem)
	return err
}
