package rubix

type OIDCProvider struct {
	ProviderName      string         `json:"providerName"`
	ClientID          string         `json:"clientID"`
	ClientSecret      string         `json:"clientSecret"`
	ClientKeys        string         `json:"clientKeys"`
	IssuerURL         string         `json:"issuerURL"`
	DefaultMemberType MembershipType `json:"defaultMemberType"`
	AutoActivate      bool           `json:"autoActivate"`
}

func (o OIDCProvider) Configured() bool {
	return o.ClientID != "" && o.IssuerURL != ""
}
