package rubix

type ServiceProviderState string

const (
	ServiceProviderStateActive   ServiceProviderState = "active"
	ServiceProviderStatePaused   ServiceProviderState = "paused"
	ServiceProviderStateArchived ServiceProviderState = "archived"
)

type ServiceProvider struct {
	Workspace       string
	ServiceID       string
	ServiceProvider string
	Name            string
	Description     string
	Labels          []string
	State           ServiceProviderState
	UserAccess      bool
	Token           string
}

type MutateServiceProviderPayload struct {
	Name        *string
	Description *string
	Labels      *[]string
	State       *ServiceProviderState
	UserAccess  *bool
}

type MutateServiceProviderOption func(*MutateServiceProviderPayload)

func WithServiceProviderName(name string) MutateServiceProviderOption {
	return func(p *MutateServiceProviderPayload) { p.Name = &name }
}

func WithServiceProviderDescription(description string) MutateServiceProviderOption {
	return func(p *MutateServiceProviderPayload) { p.Description = &description }
}

func WithServiceProviderLabels(labels []string) MutateServiceProviderOption {
	return func(p *MutateServiceProviderPayload) { p.Labels = &labels }
}

func WithServiceProviderState(state ServiceProviderState) MutateServiceProviderOption {
	return func(p *MutateServiceProviderPayload) { p.State = &state }
}

func WithServiceProviderUserAccess(userAccess bool) MutateServiceProviderOption {
	return func(p *MutateServiceProviderPayload) { p.UserAccess = &userAccess }
}
