package sql

import "github.com/kubex/rubix-storage/rubix"

// RetrieveAllWorkspaces returns workspace installations for platform background jobs.
func (p *Provider) RetrieveAllWorkspaces() (map[string]*rubix.Workspace, error) {
	return p.retrieveWorkspacesByQuery("1 = 1")
}
