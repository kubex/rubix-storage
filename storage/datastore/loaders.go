package datastore

import (
	"context"
	"encoding/json"
	"errors"

	"cloud.google.com/go/datastore"
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	q := datastore.NewQuery(kindWorkspace).
		Filter("Alias =", alias).
		Limit(1).KeysOnly()

	if keys, err := p.client.GetAll(context.Background(), q, nil); err != nil {
		return "", err
	} else {
		if len(keys) > 0 {
			return keys[0].Name, nil
		}
	}
	return "", nil
}

func (p Provider) GetUserWorkspaceUUIDs(userId string) ([]string, error) {
	q := datastore.NewQuery(kindMembership).
		Filter("IdentityID = ", userId).
		KeysOnly()

	wsuuids := []string{}
	if keys, err := p.client.GetAll(context.Background(), q, nil); err != nil {
		return nil, err
	} else {
		for _, key := range keys {
			if key.Parent != nil {
				wsuuids = append(wsuuids, key.Parent.Name)
			}
		}
	}
	return wsuuids, nil
}

func (p Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	q := datastore.NewQuery(kindMembership).
		Ancestor(workspaceStore{Uuid: workspaceUuid}.dsID()).
		KeysOnly()

	members := []string{}
	if keys, err := p.client.GetAll(context.Background(), q, nil); err != nil {
		return nil, err
	} else {
		for _, key := range keys {
			members = append(members, key.Name)
		}
	}
	return members, nil
}

func (p Provider) RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error) {
	ws := &workspaceStore{Uuid: workspaceUuid}
	if readErr := p.client.Get(context.Background(), ws.dsID(), ws); readErr != nil {
		if errors.Is(readErr, datastore.ErrNoSuchEntity) {
			return nil, ErrNotFound
		}
		return nil, readErr
	}

	workspace := &rubix.Workspace{
		Uuid:   ws.Uuid,
		Alias:  ws.Alias,
		Name:   ws.Name,
		Domain: ws.Domain,
	}

	err := json.Unmarshal(ws.InstalledApplications, &workspace.InstalledApplications)
	return workspace, err
}

func (p *Provider) GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error) {
	return nil, nil
}

func (p Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	return []app.PermissionStatement{}, nil
}

func (p Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	return true, nil
}

func (p Provider) SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (bool, error) {
	panic("implement me")
}
func (p Provider) GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error) {
	panic("implement me")
}
func (p Provider) ClearUserStatusLogout(workspaceUuid, userUuid string) error { panic("implement me") }
func (p Provider) ClearUserStatusID(workspaceUuid, userUuid, statusID string) error {
	panic("implement me")
}
