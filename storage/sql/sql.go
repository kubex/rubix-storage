package sql

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

const ProviderKey = "sql"

type Provider struct {
	PrimaryDSN        string `json:"primaryDsn"` // user:password@tcp(hostname:port)
	Database          string `json:"database"`
	SqlLite           bool   `json:"sqlLite"`
	primaryConnection *sql.DB
	afterUpdate       []func()
}

func (p *Provider) Close() error {
	var errs []error
	if p.primaryConnection != nil {
		errs = append(errs, p.primaryConnection.Close())
	}
	return errors.Join(errs...)
}

func (p *Provider) Sync() error {
	return nil
}

func (p *Provider) Connect() error {
	if p.primaryConnection == nil {
		var err error
		if p.SqlLite {
			p.primaryConnection, err = sql.Open("libsql", p.PrimaryDSN)
			if err != nil {
				return fmt.Errorf("failed to open db %s", err)
			}
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

func (p *Provider) Initialize() error {
	if err := p.Connect(); err != nil {
		return err
	}

	if err := p.Sync(); err != nil {
		return err
	}

	if p.SqlLite {
		createMigrations := false
		row := p.primaryConnection.QueryRow("SELECT tbl_name FROM sqlite_master WHERE type='table' AND name = 'rubix_migrations';")
		if row != nil {
			if row.Err() != nil && strings.Contains(row.Err().Error(), "no rows") {
				createMigrations = true
			} else if row.Err() != nil {
				return row.Err()
			}
			tblName := ""
			row.Scan(&tblName)
			createMigrations = tblName == ""
		}

		if createMigrations {
			_, err := p.primaryConnection.Exec("create table rubix_migrations (migration varchar(255) not null primary key, applied int not null)")
			if err != nil {
				return err
			}
		}

		processed := make(map[string]bool)
		rows, err := p.primaryConnection.Query("SELECT migration, applied FROM rubix_migrations;")
		if err != nil {
			return err
		}
		for rows.Next() {
			var migKey string
			var applied int
			if scanErr := rows.Scan(&migKey, &applied); scanErr != nil {
				return scanErr
			}
			processed[migKey] = applied == 1
		}

		queries := migrations()
		for _, query := range queries {
			if !processed[query.key] {
				if _, migErr := p.primaryConnection.Exec(query.query); migErr != nil {
					return migErr
				}
				if _, migErr := p.primaryConnection.Exec("INSERT INTO rubix_migrations (migration, applied) VALUES (?, 1);", query.key); migErr != nil {
					return migErr
				}
			}
		}
	}

	return nil
}

func (p *Provider) AfterUpdate(exec func()) error {
	p.afterUpdate = append(p.afterUpdate, exec)
	return nil
}

func (p *Provider) update() {
	for _, exec := range p.afterUpdate {
		exec()
	}
}

func FromJson(data []byte) (*Provider, error) {
	p := &Provider{}
	if err := json.Unmarshal(data, &p); err == nil {
		return p, nil
	} else {
		return nil, err
	}
}
