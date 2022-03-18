package storage

import (
	"encoding/json"
	"errors"

	"github.com/kubex/rubix-storage/storage/cassandra"
	"github.com/kubex/rubix-storage/storage/datastore"
	"github.com/kubex/rubix-storage/storage/jsonfile"
)

func Load(jsonBytes []byte) (Provider, error) {

	loader := struct {
		Provider      string
		Configuration *json.RawMessage
	}{}

	err := json.Unmarshal(jsonBytes, &loader)
	if err != nil {
		return nil, err
	}

	switch loader.Provider {
	case jsonfile.ProviderKey:
		return jsonfile.FromJson(*loader.Configuration)
	case cassandra.ProviderKey:
		return cassandra.FromJson(*loader.Configuration)
	case datastore.ProviderKey:
		return datastore.FromJson(*loader.Configuration)
	}

	return nil, errors.New("unable to load provider '" + loader.Provider + "'")
}
