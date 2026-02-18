package rubix

type OIDCProvider struct {
	Uuid         string `json:"uuid"`
	Workspace    string `json:"workspace"`
	ProviderName string `json:"providerName"`
	DisplayName  string `json:"displayName"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	ClientKeys   string `json:"clientKeys"`
	IssuerURL    string `json:"issuerURL"`
}

func (o OIDCProvider) Configured() bool {
	return o.ClientID != "" && o.IssuerURL != ""
}

type MutateOIDCProviderPayload struct {
	ProviderName *string
	DisplayName  *string
	ClientID     *string
	ClientSecret *string
	ClientKeys   *string
	IssuerURL    *string
}

type MutateOIDCProviderOption func(*MutateOIDCProviderPayload)

func WithOIDCProviderName(name string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ProviderName = &name }
}

func WithOIDCDisplayName(name string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.DisplayName = &name }
}

func WithOIDCClientID(clientID string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientID = &clientID }
}

func WithOIDCClientSecret(secret string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientSecret = &secret }
}

func WithOIDCClientKeys(keys string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.ClientKeys = &keys }
}

func WithOIDCIssuerURL(url string) MutateOIDCProviderOption {
	return func(p *MutateOIDCProviderPayload) { p.IssuerURL = &url }
}
