package rubix

import (
	"errors"
	"net/netip"
	"strings"
	"time"

	"github.com/openbyte-os/sdk-go/app"
)

type GatewaySourceScope string

const (
	GatewaySourceGlobal GatewaySourceScope = "global"
	GatewaySourceVendor GatewaySourceScope = "vendor"
	GatewaySourceApp    GatewaySourceScope = "app"
)

var ErrGatewaySourceIP = errors.New("invalid gateway source IP entry")
var ErrGatewaySourceIPNotFound = errors.New("gateway source IP entry not found")

type GatewaySourceIP struct {
	ID          string             `json:"id"`
	Scope       GatewaySourceScope `json:"scope"`
	VendorID    string             `json:"vendorID,omitempty"`
	AppID       string             `json:"appID,omitempty"`
	Network     string             `json:"network"`
	Description string             `json:"description"`
	CreatedBy   string             `json:"createdBy"`
	CreatedAt   time.Time          `json:"createdAt"`
}

func ValidGatewaySourceScope(scope GatewaySourceScope, vendor, application string) bool {
	switch scope {
	case GatewaySourceGlobal:
		return vendor == "" && application == ""
	case GatewaySourceVendor:
		return vendor != "" && application == "" && app.NewID(vendor, "example").Validate(false) == nil
	case GatewaySourceApp:
		return vendor != "" && application != "" && app.NewID(vendor, application).Validate(false) == nil
	}
	return false
}

// NormalizeIPNetwork accepts literal addresses and CIDRs, not hostnames, ports,
// zones, ranges, or lists. Stored entries are canonical masked prefixes.
func NormalizeIPNetwork(value string) (string, error) {
	value = strings.TrimSpace(value)
	if addr, err := netip.ParseAddr(value); err == nil {
		if addr.Zone() != "" {
			return "", ErrGatewaySourceIP
		}
		addr = addr.Unmap()
		return netip.PrefixFrom(addr, addr.BitLen()).String(), nil
	}
	prefix, err := netip.ParsePrefix(value)
	if err != nil || prefix.Addr().Zone() != "" {
		return "", ErrGatewaySourceIP
	}
	if prefix.Addr().Is4In6() {
		if prefix.Bits() < 96 {
			return "", ErrGatewaySourceIP
		}
		prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
	}
	return prefix.Masked().String(), nil
}
