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

func (p Provider) GetWorkspaceMembers(workspaceUuid string) ([]rubix.WorkspaceMembership, error) {
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

func (p Provider) SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (bool, error) {
	panic("implement me")
}

func (p Provider) GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error) {
	panic("implement me")
}

func (p Provider) ClearUserStatusLogout(workspaceUuid, userUuid string) error {
	panic("implement me")
}

func (p Provider) ClearUserStatusID(workspaceUuid, userUuid, statusID string) error {
	panic("implement me")
}

func (p Provider) GetRole(workspace, role string) (*rubix.Role, error) {
	panic("implement me")
}

func (p Provider) GetRoles(workspace string) ([]rubix.Role, error) {
	panic("implement me")
}

func (p Provider) CreateRole(workspace, role, title, description string, permissions, users []string) error {
	panic("implement me")
}

func (p Provider) MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error {
	panic("implement me")
}

func (p Provider) SetUserType(workspace, user string, accountType rubix.UserType) error {
	panic("implement me")
}

func (p Provider) SetUserState(workspace, user string, accountType rubix.UserRowState) error {
	panic("implement me")
}

func (p Provider) RemoveUserFromWorkspace(workspace, user string) error {
	panic("implement me")
}
