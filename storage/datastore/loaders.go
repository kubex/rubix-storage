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
	return "", errors.New("no workspace found")
}

func (p Provider) GetUserWorkspaceAliases(userId string) ([]string, error) {
	return []string{}, nil
}

func (p Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	return []string{}, nil
}

func (p Provider) RetrieveWorkspace(workspaceUuid string) (*rubix.Workspace, error) {
	ws := &workspaceStore{}
	if readErr := p.client.Get(context.Background(), datastore.NameKey(kindWorkspace, workspaceUuid, nil), ws); readErr != nil {
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

func (p Provider) GetAuthData(lookup rubix.Lookup) (map[string]string, error) {
	return map[string]string{}, nil
}

func (p Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	return []app.PermissionStatement{}, nil
}

func (p Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	return true, nil
}

func (p Provider) StoreWorkspace(w *rubix.Workspace) error {
	ws := &workspaceStore{
		Uuid:   w.Uuid,
		Alias:  w.Alias,
		Name:   w.Name,
		Domain: w.Domain,
	}
	ws.InstalledApplications, _ = json.Marshal(w.InstalledApplications)

	_, err := p.client.Put(context.Background(), datastore.NameKey(kindWorkspace, ws.Uuid, nil), ws)
	return err
}
