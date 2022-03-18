package jsonfile

import (
	"encoding/json"
	"os"
	"strings"
)

const ProviderKey = "jsonfile"

type Provider struct {
	dataDirectory string
}

func FromJson(data []byte) (*Provider, error) {
	cfg := struct {
		DataDirectory string `json:"dataDirectory"`
	}{}

	if err := json.Unmarshal(data, &cfg); err == nil {
		return New(cfg.DataDirectory), nil
	} else {
		return nil, err
	}
}

func New(dataDirectory string) *Provider {
	return &Provider{dataDirectory: dataDirectory}
}

func (p Provider) filePath(dataType, filename string) string {
	return strings.TrimRight(p.dataDirectory, "/") + "/" + dataType + "." + filename + ".json"
}

func (p Provider) fileData(filePath string) []byte {
	bytes, _ := os.ReadFile(filePath)
	return bytes
}
