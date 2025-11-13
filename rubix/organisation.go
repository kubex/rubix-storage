package rubix

// Brand represents a brand within a workspace
type Brand struct {
	Workspace   string
	ID          string
	Name        string
	Description string
}

// Department represents a department within a workspace
type Department struct {
	Workspace   string
	ID          string
	Name        string
	Description string
}

// Channel represents a channel within a department
type Channel struct {
	Workspace    string
	ID           string // friendly identifier
	DepartmentID string
	Name         string
	Description  string
}

// Mutate payloads and options for simple metadata updates

type MutateBrandPayload struct {
	Title       *string
	Description *string
}

type MutateBrandOption func(*MutateBrandPayload)

func WithBrandName(title string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Title = &title }
}

func WithBrandDescription(description string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Description = &description }
}

type MutateDepartmentPayload struct {
	Title       *string
	Description *string
}

type MutateDepartmentOption func(*MutateDepartmentPayload)

func WithDepartmentName(title string) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Title = &title }
}

func WithDepartmentDescription(description string) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Description = &description }
}

type MutateChannelPayload struct {
	Title       *string
	Description *string
}

type MutateChannelOption func(*MutateChannelPayload)

func WithChannelName(title string) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Title = &title }
}

func WithChannelDescription(description string) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Description = &description }
}
