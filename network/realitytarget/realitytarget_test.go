package realitytarget

import (
	"context"
	"net/netip"
	"reflect"
	"testing"
)

func TestBuiltInDomainsAreValidAndUnique(t *testing.T) {
	if len(BuiltInDomains) < 50 {
		t.Fatalf("built-in candidate pool is unexpectedly small: %d", len(BuiltInDomains))
	}
	seen := make(map[string]struct{}, len(BuiltInDomains))
	for _, domain := range BuiltInDomains {
		if !validHostname(domain) {
			t.Errorf("invalid built-in domain: %q", domain)
		}
		if _, exists := seen[domain]; exists {
			t.Errorf("duplicate built-in domain: %q", domain)
		}
		seen[domain] = struct{}{}
	}
}

func TestIsPublicAddress(t *testing.T) {
	tests := map[string]bool{
		"1.1.1.1": true, "2606:4700::1111": true,
		"127.0.0.1": false, "10.0.0.1": false, "100.64.0.1": false,
		"169.254.1.1": false, "192.0.2.1": false, "198.18.0.1": false,
		"203.0.113.1": false, "::1": false, "fd00::1": false, "2001:db8::1": false,
	}
	for raw, want := range tests {
		if got := isPublicAddress(netip.MustParseAddr(raw)); got != want {
			t.Errorf("isPublicAddress(%q) = %v, want %v", raw, got, want)
		}
	}
}

func TestSelectBatchIsStableAndChanges(t *testing.T) {
	domains := []string{"a.example", "b.example", "c.example", "d.example"}
	first := SelectBatch(domains, 1, 3)
	again := SelectBatch(domains, 1, 3)
	other := SelectBatch(domains, 2, 3)
	if !reflect.DeepEqual(first, again) {
		t.Fatalf("same batch was not stable: %#v != %#v", first, again)
	}
	if reflect.DeepEqual(first, other) {
		t.Fatalf("different batches unexpectedly matched: %#v", first)
	}
	if !reflect.DeepEqual(domains, []string{"a.example", "b.example", "c.example", "d.example"}) {
		t.Fatalf("input was modified: %#v", domains)
	}
}

func TestScanRejectsInvalidLimits(t *testing.T) {
	if got := Scan(context.Background(), []string{"example.com"}, 0, 8); len(got) != 0 {
		t.Fatalf("Scan with zero attempts returned %#v", got)
	}
	if got := Scan(context.Background(), []string{"example.com"}, 3, 0); len(got) != 0 {
		t.Fatalf("Scan with zero workers returned %#v", got)
	}
}
