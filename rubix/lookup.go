package rubix

import "github.com/kubex/definitions-go/app"

type Lookup struct {
	WorkspaceUUID string
	UserUUID      string
	AppID         app.GlobalAppID
}

func NewLookup(WorkspaceUUID, UserUUID string, AppID app.GlobalAppID) Lookup {
	return Lookup{WorkspaceUUID: WorkspaceUUID, UserUUID: UserUUID, AppID: AppID}
}

func (l Lookup) String() string {
	return l.WorkspaceUUID + "---" + l.UserUUID + "---" + l.AppID.VendorID + "---" + l.AppID.AppID
}
