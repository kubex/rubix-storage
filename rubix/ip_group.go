package rubix

type IPGroup struct {
	Workspace   string   `json:"workspace"`
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Source      string   `json:"source"`     // "manual" or "external"
	Entries     []string `json:"entries"`     // IPs and CIDRs
	ExternalURL string   `json:"externalUrl"`
	JSONPath    string   `json:"jsonPath"`
	LastSynced  string   `json:"lastSynced"` // RFC3339
	EntryCount  int      `json:"entryCount"`
}

type MutateIPGroupPayload struct {
	Title       *string
	Description *string
	Source      *string
	Entries     *[]string
	ExternalURL *string
	JSONPath    *string
	LastSynced  *string
	EntryCount  *int
}

type MutateIPGroupOption func(*MutateIPGroupPayload)

func WithIPGroupName(name string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.Title = &name }
}

func WithIPGroupDescription(desc string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.Description = &desc }
}

func WithIPGroupSource(source string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.Source = &source }
}

func WithIPGroupEntries(entries []string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) {
		p.Entries = &entries
		count := len(entries)
		p.EntryCount = &count
	}
}

func WithIPGroupExternalURL(url string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.ExternalURL = &url }
}

func WithIPGroupJSONPath(path string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.JSONPath = &path }
}

func WithIPGroupLastSynced(ts string) MutateIPGroupOption {
	return func(p *MutateIPGroupPayload) { p.LastSynced = &ts }
}
