package main

import (
	"github.com/kubex/rubix-storage/storage/sql"
	"log"
)

func main() {
	prov := sql.Provider{
		SqlLite: true,
		PrimaryDSN: "libsql://" + "cosmic-bumblebee-bajbnet" + ".turso.io?" +
			"authToken=eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3MjA3MDkyMDYsImlkIjoiNDJjZDNiOGYtZmRhOC00MDNkLTk0MWItYzBiMjhmOTA5YzRhIn0.LlrChE3nCSFMEO9TxBXM-arMhFNZc345vWXz0IplpmdbDwnsZoEkjoPR9kZjzeEhuN5m0gkA6CeXxo8RRHbuDA",
	}

	/*prov.PrimaryDSN = "libsql://" + "rubix-bajb" + ".turso.io?authToken=" +
	"eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3MjA3ODU2NjQsImlkIjoiYjUzOTI0MTAtMmI4Yy00NGY0LThjNzUtZDRkYTMzN2Y3NzY2In0.BdN_khNO6a-BATOIMHl-MqlV8tCvk1-bNELwhROeuEAfdAhoYo99NTKEL_uD0LMv3UoBlrfU8eb71qN24_QYBg"
	*/

	prov.PrimaryDSN = "libsql://rubix-bajb.turso.io?authToken=eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJhIjoicnciLCJpYXQiOjE3MjA3ODU2NjQsImlkIjoiYjUzOTI0MTAtMmI4Yy00NGY0LThjNzUtZDRkYTMzN2Y3NzY2In0.BdN_khNO6a-BATOIMHl-MqlV8tCvk1-bNELwhROeuEAfdAhoYo99NTKEL_uD0LMv3UoBlrfU8eb71qN24_QYBg"

	log.Println(prov.Initialize())
	prov.Sync()
	log.Println(prov.GetUserStatus("kyle-fortifi-1", "bcb06253-26d7-4f90-98fc-84609e9c653a"))

	log.Println(prov.RetrieveWorkspaceByDomain("chargehive-sandbox.chargehive.cloud"))
	log.Println(prov.GetWorkspaceUUIDByAlias("chargehive-sandbox"))

	uuids, err := prov.GetUserWorkspaceUUIDs("bcb06253-26d7-4f90-98fc-84609e9c653a")
	if err != nil {
		log.Println(err)
	} else {
		ws, wsErr := prov.RetrieveWorkspaces(uuids...)
		if wsErr != nil {
			log.Println(wsErr)
		} else {
			for _, w := range ws {
				log.Println(w)
			}
		}
	}
}
