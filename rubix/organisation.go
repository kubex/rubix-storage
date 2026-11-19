package rubix

// Brand represents a brand within a workspace
type Brand struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Website     string
	Icon        string
}

type OrganisationStatus string

const (
	OrganisationStatusActive   OrganisationStatus = "active"
	OrganisationStatusArchived OrganisationStatus = "archived"
)

// Department represents a department within a workspace
type Department struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	Icon        string
	Status      OrganisationStatus
}

func (d Department) IsArchived() bool { return d.Status == OrganisationStatusArchived }

// Channel represents a channel within a department
type Channel struct {
	Workspace    string
	ID           string // friendly identifier
	DepartmentID string
	Name         string
	Description  string
	// MaxLevel sets the highest level that can be assigned within this channel
	// 0 means no level restriction is configured
	MaxLevel int
	Icon     string
	Status   OrganisationStatus
}

func (c Channel) IsArchived() bool { return c.Status == OrganisationStatusArchived }

// Distributor represents a distributor within a workspace
type Distributor struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	WebsiteURL  string
	LogoURL     string
	Icon        string
}

// BPO represents a BPO within a workspace
type BPO struct {
	Workspace   string
	ID          string
	Name        string
	Description string
	WebsiteURL  string
	LogoURL     string
	Icon        string
}

// Mutate payloads and options for simple metadata updates

type MutateBrandPayload struct {
	Title       *string
	Description *string
	Website     *string
	Icon        *string
}

type MutateBrandOption func(*MutateBrandPayload)

// WithBrandIcon sets the lucide icon name.
func WithBrandIcon(icon string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Icon = &icon }
}

// WithBrandWebsite sets the brand website.
func WithBrandWebsite(website string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Website = &website }
}

func WithBrandName(title string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Title = &title }
}

func WithBrandDescription(description string) MutateBrandOption {
	return func(p *MutateBrandPayload) { p.Description = &description }
}

type MutateDepartmentPayload struct {
	Title       *string
	Description *string
	Icon        *string
	Status      *OrganisationStatus
}

type MutateDepartmentOption func(*MutateDepartmentPayload)

// WithDepartmentIcon sets the lucide icon name.
func WithDepartmentIcon(icon string) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Icon = &icon }
}

func WithDepartmentName(title string) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Title = &title }
}

func WithDepartmentDescription(description string) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Description = &description }
}

func WithDepartmentStatus(status OrganisationStatus) MutateDepartmentOption {
	return func(p *MutateDepartmentPayload) { p.Status = &status }
}

type MutateDistributorPayload struct {
	Title       *string
	Description *string
	WebsiteURL  *string
	LogoURL     *string
	Icon        *string
}

type MutateDistributorOption func(*MutateDistributorPayload)

// WithDistributorIcon sets the lucide icon name.
func WithDistributorIcon(icon string) MutateDistributorOption {
	return func(p *MutateDistributorPayload) { p.Icon = &icon }
}

func WithDistributorName(title string) MutateDistributorOption {
	return func(p *MutateDistributorPayload) { p.Title = &title }
}

func WithDistributorDescription(description string) MutateDistributorOption {
	return func(p *MutateDistributorPayload) { p.Description = &description }
}

func WithDistributorWebsiteURL(url string) MutateDistributorOption {
	return func(p *MutateDistributorPayload) { p.WebsiteURL = &url }
}

func WithDistributorLogoURL(url string) MutateDistributorOption {
	return func(p *MutateDistributorPayload) { p.LogoURL = &url }
}

type MutateBPOPayload struct {
	Title       *string
	Description *string
	WebsiteURL  *string
	LogoURL     *string
	Icon        *string
}

type MutateBPOOption func(*MutateBPOPayload)

// WithBPOIcon sets the lucide icon name.
func WithBPOIcon(icon string) MutateBPOOption {
	return func(p *MutateBPOPayload) { p.Icon = &icon }
}

func WithBPOName(title string) MutateBPOOption {
	return func(p *MutateBPOPayload) { p.Title = &title }
}

func WithBPODescription(description string) MutateBPOOption {
	return func(p *MutateBPOPayload) { p.Description = &description }
}

func WithBPOWebsiteURL(url string) MutateBPOOption {
	return func(p *MutateBPOPayload) { p.WebsiteURL = &url }
}

func WithBPOLogoURL(url string) MutateBPOOption {
	return func(p *MutateBPOPayload) { p.LogoURL = &url }
}

type MutateChannelPayload struct {
	Title       *string
	Description *string
	MaxLevel    *int
	Icon        *string
	Status      *OrganisationStatus
}

type MutateChannelOption func(*MutateChannelPayload)

// WithChannelIcon sets the lucide icon name.
func WithChannelIcon(icon string) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Icon = &icon }
}

func WithChannelName(title string) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Title = &title }
}

func WithChannelDescription(description string) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Description = &description }
}

// WithChannelMaxLevel sets the maximum level allowed for this channel
// Pass 0 to clear or disable level restrictions
func WithChannelMaxLevel(level int) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.MaxLevel = &level }
}

func WithChannelStatus(status OrganisationStatus) MutateChannelOption {
	return func(p *MutateChannelPayload) { p.Status = &status }
}

// BPO Management - managers, allowed teams, allowed roles

type BPOManager struct {
	Workspace string
	BPO       string
	User      string
}

type BPOTeam struct {
	Workspace string
	BPO       string
	Team      string
}

type BPORole struct {
	Workspace string
	BPO       string
	Role      string
}
