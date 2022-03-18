package cassandra

import (
	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) GetUserWorkspaceAliases(userId string) ([]string, error) {
	panic("implement me")
}

func (p Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	panic("implement me")
}

func (p Provider) RetrieveWorkspace(workspaceAlias string) (*rubix.Workspace, error) {
	panic("implement me")
}

func (p Provider) GetAuthData(lookup rubix.Lookup) (map[string]string, error) {
	panic("implement me")
}

func (p Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	panic("implement me")
}

func (p Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	panic("implement me")
}
