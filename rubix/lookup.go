package rubix

import "github.com/kubex/definitions-go/app"

type Lookup struct {
	WorkspaceUUID string
	UserUUID      string
	AppID         app.GlobalAppID
	GeoLocation   string
}

type DataResult struct {
	VendorID string
	AppID    string
	Key      string
	Value    string
}

func NewLookup(WorkspaceUUID, UserUUID string, AppID app.GlobalAppID, geoLocation string) Lookup {
	return Lookup{WorkspaceUUID: WorkspaceUUID, UserUUID: UserUUID, AppID: AppID, GeoLocation: geoLocation}
}

func (l Lookup) String() string {
	return l.WorkspaceUUID + "---" + l.UserUUID + "---" + l.AppID.VendorID + "---" + l.AppID.AppID
}
