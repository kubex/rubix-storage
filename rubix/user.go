package rubix

type MutateUserOption func(*MutateUserPayload)

type MutateUserPayload struct {
	Name          *string
	Email         *string
	RolesToAdd    []string
	RolesToRemove []string
}

func WithUserName(name string) MutateUserOption {
	return func(p *MutateUserPayload) { p.Name = &name }
}

func WithUserEmail(email string) MutateUserOption {
	return func(p *MutateUserPayload) { p.Email = &email }
}

func WithRolesToAdd(roles ...string) MutateUserOption {
	return func(p *MutateUserPayload) {
		p.RolesToAdd = append(p.RolesToAdd, roles...)
	}
}

func WithRolesToRemove(roles ...string) MutateUserOption {
	return func(p *MutateUserPayload) {
		p.RolesToRemove = append(p.RolesToRemove, roles...)
	}
}
