package cassandra

import (
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	panic("implement me")
}

func (p Provider) GetUserWorkspaceUUIDs(userId string) ([]string, error) {
	panic("implement me")
}

func (p Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	panic("implement me")
}

func (p Provider) RetrieveWorkspace(workspaceAlias string) (*rubix.Workspace, error) {
	panic("implement me")
}

func (p *Provider) GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error) {
	panic("implement me")
}

func (p Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	panic("implement me")
}

func (p Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	panic("implement me")
}
