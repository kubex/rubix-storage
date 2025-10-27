package rubix

import "github.com/kubex/definitions-go/app"

type Role struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Users       []string         // Not on roles table
	Permissions []RolePermission // Not on roles table
	Constraints []UserRoleConstraint
}

type UserRole struct {
	Workspace string
	User      string
	Role      string
}

type UserRoleConstraint struct {
	Type     UserRoleConstraintType     `json:"type"`
	Operator UserRoleConstraintOperator `json:"operator"`
	Value    interface{}                `json:"value"`
}

type UserRoleConstraintType string

const (
	UserRoleConstraintTypeLocation  UserRoleConstraintType = "location"
	UserRoleConstraintTypeIpAddress UserRoleConstraintType = "ipAddress"
)

type UserRoleConstraintOperator string

const (
	UserRoleConstraintOperatorInList UserRoleConstraintOperator = "inList"
)

type RolePermission struct {
	Workspace   string                     `json:"workspace"`
	Role        string                     `json:"role"`
	Permission  string                     `json:"permission"`
	Resource    string                     `json:"resource"`
	Allow       bool                       `json:"allow"`
	Meta        string                     `json:"meta"`
	Constraints []RolePermissionConstraint `json:"constraints"`
}

type RolePermissionConstraint struct {
	Field    string                           `json:"field"`
	Type     app.PermissionConstraintType     `json:"type"`
	Operator app.PermissionConstraintOperator `json:"operator"`
	Value    interface{}                      `json:"value"`
}

type MutateRolePayload struct {
	Title       *string
	Description *string
	UsersToAdd  []string
	UsersToRem  []string
	PermsToAdd  map[string][]RolePermissionConstraint
	PermsToRem  []string
	Constraints *[]UserRoleConstraint
}

type MutateRoleOption func(*MutateRolePayload)

func WithName(title string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.Title = &title
	}
}

func WithDescription(description string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.Description = &description
	}
}

func WithConstraints(constraints []UserRoleConstraint) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.Constraints = &constraints
	}
}

func WithUsersToAdd(users ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.UsersToAdd = append(p.UsersToAdd, users...)
	}
}

func WithUsersToRemove(users ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.UsersToRem = append(p.UsersToRem, users...)
	}
}

func WithPermsToAdd(perms map[string][]RolePermissionConstraint) MutateRoleOption {
	return func(p *MutateRolePayload) {
		if p.PermsToAdd == nil {
			p.PermsToAdd = make(map[string][]RolePermissionConstraint)
		}
		for k, v := range perms {
			p.PermsToAdd[k] = v
		}
	}
}

func WithPermsToRemove(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToRem = append(p.PermsToRem, perms...)
	}
}
