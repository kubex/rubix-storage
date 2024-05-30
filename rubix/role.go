package rubix

type Role struct {
	Workspace   string
	Role        string
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
	Workspace  string `json:"workspace"`
	Role       string `json:"role"`
	Permission string `json:"permission"`
	Resource   string `json:"resource"`
	Allow      bool   `json:"allow"`
	Meta       string `json:"meta"`
}

type MutateRolePayload struct {
	Title       *string
	Description *string
	UsersToAdd  []string
	UsersToRem  []string
	PermsToAdd  []string
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

func WithPermsToAdd(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToAdd = append(p.PermsToAdd, perms...)
	}
}

func WithPermsToRemove(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToRem = append(p.PermsToRem, perms...)
	}
}
