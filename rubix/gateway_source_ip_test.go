package rubix

import "testing"

func TestNormalizeIPNetwork(t *testing.T) {
	for input, want := range map[string]string{"203.0.113.4": "203.0.113.4/32", "203.0.113.9/24": "203.0.113.0/24", "2001:db8::1": "2001:db8::1/128", "2001:db8::3/48": "2001:db8::/48", "::ffff:203.0.113.4": "203.0.113.4/32", "::ffff:203.0.113.4/120": "203.0.113.0/24"} {
		got, err := NormalizeIPNetwork(input)
		if err != nil || got != want {
			t.Fatalf("%s: %s %v", input, got, err)
		}
	}
	for _, input := range []string{"", "example.com", "203.0.113.1:443", "fe80::1%eth0", "203.0.113.1,198.51.100.1", "10.0.0.1/33", "2001:db8::/129"} {
		if _, err := NormalizeIPNetwork(input); err == nil {
			t.Fatal("invalid source accepted", input)
		}
	}
}

func TestValidGatewaySourceScope(t *testing.T) {
	for _, ids := range [][2]string{{"kx", "developer"}, {"ab", "cd"}, {"a", "b"}, {"acme", "orders"}} {
		if !ValidGatewaySourceScope(GatewaySourceVendor, ids[0], "") {
			t.Errorf("vendor scope rejected %q", ids[0])
		}
		if !ValidGatewaySourceScope(GatewaySourceApp, ids[0], ids[1]) {
			t.Errorf("app scope rejected %q", ids)
		}
	}
	for _, invalid := range []string{"", "KX", "-kx", "kx-", "k/x", "k_x", "kx\n", " kx"} {
		if ValidGatewaySourceScope(GatewaySourceVendor, invalid, "") || ValidGatewaySourceScope(GatewaySourceApp, invalid, "orders") || ValidGatewaySourceScope(GatewaySourceApp, "kx", invalid) {
			t.Errorf("invalid ID accepted: %q", invalid)
		}
	}
	if !ValidGatewaySourceScope(GatewaySourceGlobal, "", "") || ValidGatewaySourceScope(GatewaySourceGlobal, "kx", "") || ValidGatewaySourceScope(GatewaySourceVendor, "kx", "orders") || ValidGatewaySourceScope("unknown", "", "") {
		t.Fatal("invalid scope ownership validation")
	}
}
