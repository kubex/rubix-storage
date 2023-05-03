package mysql

import (
	"log"
	"testing"
)

func TestDataStore(t *testing.T) {
	prov, err := FromJson([]byte(`{"primaryDsn":"root@tcp(mysql.dev.local-host.xyz:3306)", "database":"rubix"}`))
	if err != nil {
		t.Error(err)
		return
	}

	prov.Connect()
	defer prov.Close()

	log.Println(prov.GetUserWorkspaceUUIDs("abc"))
}
