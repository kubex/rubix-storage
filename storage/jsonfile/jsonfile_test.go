package jsonfile

import (
	"strings"
	"testing"
)

func TestRetrieveWorkspace(t *testing.T) {

	noWorkspaceError := "unable to load workspace"

	inst := Provider{}

	if wspace, err := inst.RetrieveWorkspace(""); wspace != nil || err == nil || !strings.Contains(err.Error(), noWorkspaceError) {
		t.Errorf("Retrieving workspaces with an invalid request, received %s", err)
	}

	if wspace, err := inst.RetrieveWorkspace("testing"); wspace != nil || err == nil || !strings.Contains(err.Error(), "unable to load workspace") {
		t.Errorf("Expected, unable to load workspace, received: %s", err)
	}

	inst.dataDirectory = "xxx"
	wspace, err := inst.RetrieveWorkspace("pass")
	if err == nil || wspace != nil {
		t.Errorf("Expected to fail loading workspace from an invalid root path")
	}
	unableToLoad := "unable to load workspace.json"
	if _, err := inst.RetrieveWorkspace("pass"); err == nil || !strings.Contains(err.Error(), unableToLoad) {
		t.Errorf("expected to fail workspace load, received %s", err)
	}

	inst.dataDirectory = "_testdata"

	if _, err := inst.RetrieveWorkspace(""); err == nil || !strings.Contains(err.Error(), unableToLoad) {
		t.Errorf("expected to fail workspace load, received %s", err)
	}

	unableToDecode := "unable to decode workspace json"
	for _, wsId := range []string{"empty", "corrupt"} {
		if _, err := inst.RetrieveWorkspace(wsId); err == nil || !strings.Contains(err.Error(), unableToDecode) {
			t.Errorf("expected to fail workspace decode, received %s", err)
		}
	}

	if _, err := inst.RetrieveWorkspace("baddata"); err == nil || !strings.Contains(err.Error(), "invalid workspace data") {
		t.Errorf("expected to fail workspace read, received %s", err)
	}

	wsDef, err := inst.RetrieveWorkspace("pass")
	if err != nil {
		t.Errorf("Expected valid result, received error %s", err)
	} else if wsDef == nil {
		t.Error("received no error or app definition")
	} else {
		if wsDef.Alias != "pass" || wsDef.Uuid != "fc9eb3db-048b-40d1-8a97-394306d7e948" {
			t.Error("The incorrect workspace was returned")
		}
	}
}
