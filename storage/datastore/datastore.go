package datastore

import "encoding/json"

const ProviderKey = "datastore"

type Provider struct {
}

func FromJson(data []byte) (*Provider, error) {
	cfg := struct{}{}

	if err := json.Unmarshal(data, &cfg); err == nil {
		return &Provider{}, nil
	} else {
		return nil, err
	}
}
