package rubix

import (
	"errors"
	"fmt"
)

// VendorAppOrder is the order a vendor prefers its apps in on its marketplace
// page, most important first. Apps it leaves out follow, by name.
type VendorAppOrder struct {
	VendorID  string
	AppIDs    []string
	UpdatedBy string
}

const VendorAppOrderMax = 200

var ErrVendorAppOrder = errors.New("invalid app order")

// Validate reports a missing vendor, too many apps, or a blank or repeated app.
func (o VendorAppOrder) Validate() error {
	if o.VendorID == "" {
		return fmt.Errorf("%w: vendor is required", ErrVendorAppOrder)
	}
	if len(o.AppIDs) > VendorAppOrderMax {
		return fmt.Errorf("%w: more than %d apps", ErrVendorAppOrder, VendorAppOrderMax)
	}
	seen := map[string]bool{}
	for _, id := range o.AppIDs {
		if id == "" || len(id) > 64 {
			return fmt.Errorf("%w: app IDs are 1 to 64 characters", ErrVendorAppOrder)
		}
		if seen[id] {
			return fmt.Errorf("%w: %s is listed twice", ErrVendorAppOrder, id)
		}
		seen[id] = true
	}
	return nil
}
