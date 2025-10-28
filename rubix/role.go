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
	Constraints []app.PermissionConstraint `json:"constraints"`
}

type MutateRolePayload struct {
	Title                *string
	Description          *string
	UsersToAdd           []string
	UsersToRem           []string
	PermsToAdd           []string
	PermConstraintsToAdd map[string][]app.PermissionConstraint
	PermsToRem           []string
	Constraints          *[]UserRoleConstraint
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

func WithPermsToAdd(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToAdd = append(p.PermsToAdd, perms...)
	}
}

func WithPermConstraintsToAdd(perms map[string][]app.PermissionConstraint) MutateRoleOption {
	return func(p *MutateRolePayload) {
		if p.PermConstraintsToAdd == nil {
			p.PermConstraintsToAdd = make(map[string][]app.PermissionConstraint)
		}
		for k, v := range perms {
			p.PermConstraintsToAdd[k] = v
		}
	}
}

func WithPermsToRemove(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToRem = append(p.PermsToRem, perms...)
	}
}
