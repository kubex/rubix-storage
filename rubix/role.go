package rubix

type Role struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Users       []string         // Not on roles table
	Permissions []RolePermission // Not on roles table
}

type UserRole struct {
	Workspace string
	User      string
	Role      string
}

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
	Type     RolePermissionConstraintType     `json:"type"`
	Operator RolePermissionConstraintOperator `json:"operator"`
	Value    interface{}                      `json:"value"`
}

type RolePermissionConstraintType string

const (
	TypeValue    RolePermissionConstraintType = "value"
	TypeLocation RolePermissionConstraintType = "location"
)

type RolePermissionConstraintOperator string

const (
	OperatorLessThan           RolePermissionConstraintOperator = "lessThan"
	OperatorGreaterThan        RolePermissionConstraintOperator = "greaterThan"
	OperatorEqual              RolePermissionConstraintOperator = "equal"
	OperatorNotEqual           RolePermissionConstraintOperator = "notEqual"
	OperatorLessThanOrEqual    RolePermissionConstraintOperator = "lessThanOrEqual"
	OperatorGreaterThanOrEqual RolePermissionConstraintOperator = "greaterThanOrEqual"
)

type MutateRolePayload struct {
	Title       *string
	Description *string
	UsersToAdd  []string
	UsersToRem  []string
	PermsToAdd  map[string][]RolePermissionConstraint
	PermsToRem  []string
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
