package rubix

type MutateUserOption func(*MutateUserPayload)

type MutateUserPayload struct {
	RolesToAdd    []string
	RolesToRemove []string
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
