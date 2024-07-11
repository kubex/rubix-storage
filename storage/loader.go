package storage

import (
	"encoding/json"
	"errors"
	"github.com/kubex/rubix-storage/storage/sql"
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
	case sql.ProviderKey:
		return sql.FromJson(*loader.Configuration)
	}

	return nil, errors.New("unable to load storage provider '" + loader.Provider + "'")
}
