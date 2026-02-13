package rubix

type OIDCProvider struct {
	ProviderName string `json:"providerName"`
	ClientID     string `json:"clientID"`
	ClientSecret string `json:"clientSecret"`
	ClientKeys   string `json:"clientKeys"`
	IssuerURL    string `json:"issuerURL"`
}

func (o OIDCProvider) Configured() bool {
	return o.ClientID != "" && o.IssuerURL != ""
}
