package mysql

import (
	"encoding/json"
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid FROM workspaces WHERE alias = ?", alias)
	located := ""
	err := q.Scan(&located)
	return located, err
}

func (p *Provider) GetUserWorkspaceUUIDs(userId string) ([]string, error) {
	rows, err := p.primaryConnection.Query("SELECT workspace FROM workspace_memberships WHERE user = ?", userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	workspaces := []string{}
	for rows.Next() {
		var workspace string
		if err := rows.Scan(&workspace); err != nil {
			return nil, err
		}
		workspaces = append(workspaces, workspace)
	}
	return workspaces, nil
}

func (p *Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	rows, err := p.primaryConnection.Query("SELECT user FROM workspace_memberships WHERE workspace = ?", workspaceUuid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := []string{}
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (p *Provider) RetrieveWorkspace(workspaceAlias string) (*rubix.Workspace, error) {
	q := p.primaryConnection.QueryRow("SELECT uuid, alias, domain, name, installedApplications FROM workspaces WHERE alias = ?", workspaceAlias)
	located := rubix.Workspace{}
	installedApplicationsJson := ""
	err := q.Scan(&located.Uuid, &located.Alias, &located.Domain, &located.Name, &installedApplicationsJson)
	json.Unmarshal([]byte(installedApplicationsJson), &located.InstalledApplications)
	return &located, err
}

func (p *Provider) GetAuthData(lookups ...rubix.Lookup) (map[string]string, error) {
	panic("implement me")
}

func (p *Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	panic("implement me")
}

func (p *Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	panic("implement me")
}
