package cassandra

import (
	"encoding/json"
)

const ProviderKey = "cassandra"

type Provider struct {
	Hosts    []string
	Keyspace string
}

func FromJson(data []byte) (*Provider, error) {
	cfg := struct {
		Hosts    []string `json:"hosts"`
		Keyspace string   `json:"keyspace"`
	}{}

	if err := json.Unmarshal(data, &cfg); err == nil {
		return &Provider{}, nil
	} else {
		return nil, err
	}
}
