package mysql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"github.com/tursodatabase/go-libsql"
	"os"
	"path/filepath"
)

const ProviderKey = "mysql"

type Provider struct {
	PrimaryDSN        string   `json:"primaryDsn"` // user:password@tcp(hostname:port)
	ReplicaDSNs       []string `json:"replicaDsns"`
	Database          string   `json:"database"`
	UseTurso          bool     `json:"useTurso"`
	TursoToken        string   `json:"tursoToken"`
	primaryConnection *sql.DB
	tursoDir          string
	tursoConnector    *libsql.Connector
}

func (p *Provider) Close() error {
	var errs []error
	if p.primaryConnection != nil {
		errs = append(errs, p.primaryConnection.Close())
	}
	if p.tursoConnector != nil {
		errs = append(errs, p.tursoConnector.Close())
	}
	if p.tursoDir != "" {
		errs = append(errs, os.RemoveAll(p.tursoDir))
	}
	return errors.Join(errs...)
}

func (p *Provider) Sync() error {
	if p.tursoConnector != nil {
		return p.tursoConnector.Sync()
	}
	return nil
}

func (p *Provider) Connect() error {
	if p.primaryConnection == nil {

		var err error
		if p.UseTurso {
			dbName := "rubix.db"
			primaryUrl := "libsql://" + p.Database + ".turso.io"
			authToken := p.TursoToken

			p.tursoDir, err = os.MkdirTemp("", "libsql-*")
			if err != nil {
				return fmt.Errorf("error creating temporary directory: %s", err)
			}

			dbPath := filepath.Join(p.tursoDir, dbName)
			p.tursoConnector, err = libsql.NewEmbeddedReplicaConnector(dbPath, primaryUrl, libsql.WithAuthToken(authToken))
			if err != nil {
				return err
			}

			p.tursoConnector.Sync()

			p.primaryConnection = sql.OpenDB(p.tursoConnector)
		} else {
			p.primaryConnection, err = sql.Open("mysql", p.PrimaryDSN+"/"+p.Database+"?parseTime=true")
		}

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
