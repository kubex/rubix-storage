package sql

import (
	"errors"
	"slices"
	"testing"

	"github.com/kubex/rubix-storage/rubix"
)

func TestVendorAppOrder(t *testing.T) {
	p := newTestProvider(t)
	defer p.Close()

	if err := p.StoreVendorAppOrder(rubix.VendorAppOrder{VendorID: "acme", AppIDs: []string{"crm", "billing"}}); err != nil {
		t.Fatal(err)
	}
	if err := p.StoreVendorAppOrder(rubix.VendorAppOrder{VendorID: "acme", AppIDs: []string{"billing", "crm", "notes"}}); err != nil {
		t.Fatal(err)
	}
	orders, err := p.GetVendorAppOrders()
	if err != nil || !slices.Equal(orders["acme"], []string{"billing", "crm", "notes"}) {
		t.Fatalf("orders = %v, %v", orders, err)
	}

	if err := p.StoreVendorAppOrder(rubix.VendorAppOrder{VendorID: "acme"}); err != nil {
		t.Fatal(err)
	}
	if orders, _ := p.GetVendorAppOrders(); len(orders) != 0 {
		t.Fatalf("an empty order is removed, got %v", orders)
	}

	for _, bad := range []rubix.VendorAppOrder{{AppIDs: []string{"crm"}}, {VendorID: "acme", AppIDs: []string{"crm", "crm"}}, {VendorID: "acme", AppIDs: []string{""}}} {
		if err := p.StoreVendorAppOrder(bad); !errors.Is(err, rubix.ErrVendorAppOrder) {
			t.Errorf("%+v accepted: %v", bad, err)
		}
	}
}
