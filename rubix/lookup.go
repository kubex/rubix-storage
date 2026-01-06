package rubix

import (
	"net"
	"time"

	"github.com/kubex/definitions-go/app"
)

type Lookup struct {
	WorkspaceUUID   string
	UserUUID        string
	AppID           app.GlobalAppID
	GeoLocation     string
	IpAddress       net.IP
	MFA             bool
	VerifiedAccount bool
	SessionIssued   time.Time
}

type DataResult struct {
	VendorID string
	AppID    string
	Key      string
	Value    string
}

type Setting struct {
	Workspace string
	Vendor    string
	App       string
	Key       string
	Value     string
}

func NewLookup(WorkspaceUUID, UserUUID string, AppID app.GlobalAppID, geoLocation string, ipAddress string, mfa bool, verifiedAccount bool, sessionIssued time.Time) Lookup {
	return Lookup{
		WorkspaceUUID:   WorkspaceUUID,
		UserUUID:        UserUUID,
		AppID:           AppID,
		GeoLocation:     geoLocation,
		IpAddress:       net.ParseIP(ipAddress),
		MFA:             mfa,
		VerifiedAccount: verifiedAccount,
		SessionIssued:   sessionIssued,
	}
}

func (l Lookup) String() string {
	return l.WorkspaceUUID + "---" + l.UserUUID + "---" + l.AppID.VendorID + "---" + l.AppID.AppID
}
