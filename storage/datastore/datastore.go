package datastore

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/kubex/rubix-storage/rubix"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

const ProviderKey = "datastore"
const kindWorkspace = "RxWorkspace"
const kindMembership = "RxMembership"
const kindMemberPermission = "RxMemberPermission"

var ErrNotFound = errors.New("datastore: not found")
var ErrReadFailure = errors.New("datastore: read failure")

type Provider struct {
	client    dataStoreClient
	ProjectID string `json:"projectId"`
}

func FromJson(data []byte) (*Provider, error) {
	p := &Provider{}
	if err := json.Unmarshal(data, &p); err == nil {
		return p, p.Init()
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
	Uuid                  string `datastore:"-"`
	Alias                 string
	Name                  string `datastore:",noindex"`
	Domain                string `datastore:",noindex"`
	InstalledApplications []byte `datastore:",noindex"`
}

func (ws workspaceStore) dsID() *datastore.Key {
	return datastore.NameKey(kindWorkspace, ws.Uuid, nil)
}

type workspaceMembership struct {
	WorkspaceUUID string `datastore:"-"`
	IdentityID    string
	Role          rubix.MembershipRole
}

func (wm workspaceMembership) dsID() *datastore.Key {
	return datastore.NameKey(kindMembership, wm.IdentityID, workspaceStore{Uuid: wm.WorkspaceUUID}.dsID())
}

type workspaceMemberPermission struct {
	WorkspaceUUID string `datastore:"-"`
	IdentityID    string
	Vendor        string
	App           string
	Key           string
	Statement     []byte `datastore:",noindex"`
}

func (wmp workspaceMemberPermission) dsID() *datastore.Key {
	return datastore.NameKey(kindMemberPermission, wmp.Vendor+"~"+wmp.App+"~"+wmp.Key, workspaceStore{Uuid: wmp.WorkspaceUUID}.dsID())
}

type dataStoreClient interface {
	io.Closer
	Get(ctx context.Context, key *datastore.Key, dst interface{}) (err error)
	Put(ctx context.Context, key *datastore.Key, src interface{}) (*datastore.Key, error)
	GetAll(ctx context.Context, q *datastore.Query, dst interface{}) (keys []*datastore.Key, err error)
}
