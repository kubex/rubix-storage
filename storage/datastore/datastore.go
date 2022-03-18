package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"cloud.google.com/go/datastore"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const ProviderKey = "datastore"
const kindWorkspace = "RxWorkspace"

var ErrNotFound = errors.New("datastore: not found")
var ErrReadFailure = errors.New("datastore: read failure")

type Provider struct {
	client    dataStoreClient
	ProjectID string `json:"projectId"`
}

func FromJson(data []byte) (*Provider, error) {
	cfg := struct{}{}

	if err := json.Unmarshal(data, &cfg); err == nil {
		p := &Provider{}
		p.Init()
		return p, nil
	} else {
		return nil, err
	}
}

func (p *Provider) Init() error {
	var err error
	p.client, err = datastore.NewClient(context.Background(), p.ProjectID,
		option.WithGRPCDialOption(grpc.WithReturnConnectionError()),
		option.WithGRPCDialOption(grpc.WithTimeout(time.Second*5)),
		option.WithGRPCDialOption(grpc.WithDisableRetry()))
	return err
}

type workspaceStore struct {
	Uuid                  string
	Alias                 string
	Name                  string
	Domain                string
	InstalledApplications []byte
}

type dataStoreClient interface {
	io.Closer
	Get(ctx context.Context, key *datastore.Key, dst interface{}) (err error)
	Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error)
	GetAll(ctx context.Context, q *datastore.Query, dst interface{}) (keys []*datastore.Key, err error)
}
