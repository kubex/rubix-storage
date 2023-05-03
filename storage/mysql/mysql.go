package mysql

import (
	"database/sql"
	"encoding/json"
	_ "github.com/go-sql-driver/mysql"
)

const ProviderKey = "mysql"

type Provider struct {
	PrimaryDSN        string   `json:"primaryDsn"` // user:password@tcp(hostname:port)
	ReplicaDSNs       []string `json:"replicaDsns"`
	Database          string   `json:"database"`
	primaryConnection *sql.DB
}

func (p *Provider) Close() error {
	if p.primaryConnection != nil {
		return p.primaryConnection.Close()
	}
	return nil
}

func (p *Provider) Connect() error {
	if p.primaryConnection == nil {

		var err error
		p.primaryConnection, err = sql.Open("mysql", p.PrimaryDSN+"/"+p.Database)

		// Handle any errors that may occur during connection
		if err != nil {
			return err
		}
	}

	// Ping the database to ensure a successful connection
	return p.primaryConnection.Ping()
}

func FromJson(data []byte) (*Provider, error) {
	p := &Provider{}
	if err := json.Unmarshal(data, &p); err == nil {
		return p, nil
	} else {
		return nil, err
	}
}
