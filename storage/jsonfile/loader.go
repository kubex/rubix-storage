package jsonfile

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"

	"github.com/kubex/definitions-go/app"
	"github.com/kubex/rubix-storage/rubix"
)

func (p Provider) GetUserWorkspaceAliases(userId string) ([]string, error) {
	var ids []string
	err := json.Unmarshal(p.fileData(p.filePath("user", userId+".workspaces")), &ids)
	return ids, err
}

func (p Provider) GetWorkspaceUserIDs(workspaceUuid string) ([]string, error) {
	var ids []string
	err := json.Unmarshal(p.fileData(p.filePath("workspace", workspaceUuid+".users")), &ids)
	return ids, err
}

func (p Provider) RetrieveWorkspace(workspaceAlias string) (*rubix.Workspace, error) {
	jsonPath := p.filePath("workspace", workspaceAlias)
	f, err := os.Open(jsonPath)
	var bytes []byte
	if err == nil {
		bytes, err = ioutil.ReadAll(f)
	}
	if err != nil {
		return nil, errors.New("unable to load workspace.json @ " + jsonPath)
	}

	if ws, err := rubix.WorkspaceFromJson(bytes); err != nil {
		return nil, err
	} else if ws.Alias != workspaceAlias && ws.Uuid != workspaceAlias {
		return nil, errors.New("invalid workspace data")
	} else {
		return ws, nil
	}
}

func (p Provider) GetAuthData(lookup rubix.Lookup) (map[string]string, error) {
	data := map[string]string{}
	err := json.Unmarshal(p.fileData(p.filePath("auth", lookup.String())), &data)
	return data, err
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
