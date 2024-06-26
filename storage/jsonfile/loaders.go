package jsonfile

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) GetWorkspaceUUIDByAlias(alias string) (string, error) {
	if files, err := ioutil.ReadDir(p.dataDirectory); err == nil {
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") && strings.HasPrefix(file.Name(), "workspace.") {
				workspace := rubix.Workspace{}
				_ = json.Unmarshal(p.fileData(p.dataDirectory+"/"+file.Name()), &workspace)
				if workspace.Alias == alias {
					return workspace.Uuid, nil
				}
			}
		}
	}
	return "", nil
}

func (p Provider) GetUserWorkspaceUUIDs(userId string) ([]string, error) {
	var ids []string
	err := json.Unmarshal(p.fileData(p.filePath("user", userId+".workspaces")), &ids)
	return ids, err
}

func (p Provider) GetWorkspaceMembers(workspaceUuid string, userIDs ...string) ([]rubix.Membership, error) {

	var ids []string
	err := json.Unmarshal(p.fileData(p.filePath("workspace", workspaceUuid+".users")), &ids)

	members := make([]rubix.Membership, len(ids))
	for i, id := range ids {
		members[i] = rubix.Membership{
			Workspace: workspaceUuid,
			UserID:    id,
		}
	}

	return members, err
}

func (p Provider) RetrieveWorkspace(workspaceAlias string) (*rubix.Workspace, error) {
	jsonPath := p.filePath("workspace", workspaceAlias)
	f, err := os.Open(jsonPath)
	if err != nil {
		return nil, errors.New("unable to load workspace.json @ " + jsonPath)
	}
	bytes, err := io.ReadAll(f)
	if ws, err := rubix.WorkspaceFromJson(bytes); err != nil {
		return nil, err
	} else if ws.Alias != workspaceAlias && ws.Uuid != workspaceAlias {
		return nil, errors.New("invalid workspace data")
	} else {
		return ws, nil
	}
}

func (p *Provider) GetAuthData(workspaceUuid, userUuid string, appIDs ...app.GlobalAppID) ([]rubix.DataResult, error) {
	var err error
	var result []rubix.DataResult
	for _, aid := range appIDs {
		lookup := rubix.NewLookup(workspaceUuid, userUuid, aid)
		data := map[string]string{}
		err = json.Unmarshal(p.fileData(p.filePath("auth", lookup.String())), &data)
		for k, v := range data {
			result = append(result, rubix.DataResult{
				VendorID: lookup.AppID.VendorID,
				AppID:    lookup.AppID.AppID,
				Key:      k,
				Value:    v,
			})
		}
	}
	return result, err
}

func (p Provider) GetPermissionStatements(lookup rubix.Lookup, permissions ...app.ScopedKey) ([]app.PermissionStatement, error) {
	var result []app.PermissionStatement
	for _, statement := range p.userPermissionStatements(lookup) {
		for _, pk := range permissions {
			if pk.Key == statement.Permission.Key {
				result = append(result, statement)
			}
		}
	}
	return result, nil
}

func (p Provider) UserHasPermission(lookup rubix.Lookup, permissions ...app.ScopedKey) (bool, error) {
	statements := p.userPermissionStatements(lookup)
	for _, perm := range permissions {
		matchedPerm := false
		for _, statement := range statements {
			if perm.Key == statement.Permission.Key && statement.Effect == app.PermissionEffectAllow {
				matchedPerm = true
				break
			}
		}
		if !matchedPerm {
			return false, nil
		}
	}

	return true, nil
}

func (p Provider) userPermissionStatements(lookup rubix.Lookup) []app.PermissionStatement {
	statements := []app.PermissionStatement{}
	filePath := p.filePath("permissions", lookup.String())
	bytes, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return statements
	}
	err = json.Unmarshal(bytes, &statements)
	return statements
}

func (p *Provider) SetMembershipType(workspace, user string, accountType rubix.MembershipType) error {
	panic("implement me")
}

func (p *Provider) SetMembershipState(workspace, user string, accountType rubix.MembershipState) error {
	panic("implement me")
}

func (p *Provider) RemoveUserFromWorkspace(workspace, user string) error {
	panic("implement me")
}

func (p Provider) SetUserStatus(workspaceUuid, userUuid string, status rubix.UserStatus) (bool, error) {
	panic("implement me")
}

func (p Provider) GetUserStatus(workspaceUuid, userUuid string) (rubix.UserStatus, error) {
	panic("implement me")
}

func (p *Provider) AddUserToWorkspace(workspaceID, userID string, as rubix.MembershipType, partnerId string) error {
	panic("implement me")
}

func (p *Provider) CreateUser(userID, name, email string) error {
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

func (p Provider) GetUserRoles(workspace, user string) ([]rubix.UserRole, error) {
	panic("implement me")
}

func (p *Provider) DeleteRole(workspace, role string) error {
	panic("implement me")
}

func (p Provider) CreateRole(workspace, trole, itle, description string, permissions, users []string) error {
	panic("implement me")
}

func (p Provider) MutateRole(workspace, role string, options ...rubix.MutateRoleOption) error {
	panic("implement me")
}

func (p *Provider) MutateUser(workspace, user string, options ...rubix.MutateUserOption) error {
	panic("implement me")
}
