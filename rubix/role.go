package rubix

type Role struct {
	Workspace   string
	Role        string
	Name        string
	Description string
	Users       []string
	Permissions []string
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

func WithTitle(title string) MutateRoleOption {
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

func WithUsersToRem(users ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.UsersToRem = append(p.UsersToRem, users...)
	}
}

func WithPermsToAdd(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToAdd = append(p.PermsToAdd, perms...)
	}
}

func WithPermsToRem(perms ...string) MutateRoleOption {
	return func(p *MutateRolePayload) {
		p.PermsToRem = append(p.PermsToRem, perms...)
	}
}
