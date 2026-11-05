package sql

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func credentialFixture(t *testing.T) *Provider {
	t.Helper()
	p := newTestProvider(t)
	t.Cleanup(func() { _ = p.Close() })
	for _, a := range []string{"orders", "billing"} {
		if err := p.StorePlatformApplication(rubix.PlatformApplication{VendorID: "acme", AppID: a, Endpoint: "https://example.test"}); err != nil {
			t.Fatal(err)
		}
	}
	return p
}
func TestGatewayCredentialStorageAndLifecycle(t *testing.T) {
	p := credentialFixture(t)
	issued, err := p.CreateGatewayCredential("acme", "orders", "Production", "alice", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	verified, err := p.VerifyGatewayCredential(issued.Token)
	if err != nil || verified.ID != issued.Credential.ID {
		t.Fatalf("verify: %+v %v", verified, err)
	}
	if _, err := p.VerifyGatewayCredential(issued.Token[:69] + "!"); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("invalid secret accepted", err)
	}
	rows, err := p.GetGatewayCredentials("acme", "orders")
	if err != nil || len(rows) != 1 {
		t.Fatalf("list: %+v %v", rows, err)
	}
	metadata, _ := json.Marshal(rows)
	if strings.Contains(string(metadata), issued.Token) || strings.Contains(string(metadata), "token_hash") || strings.Contains(string(metadata), "digest") {
		t.Fatal("secret or digest exposed by metadata")
	}
	var stored string
	if err := p.primaryConnection.QueryRow("SELECT token_hash FROM gateway_credentials WHERE id=?", issued.Credential.ID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if len(stored) != 64 || stored == issued.Token || stored == strings.Split(issued.Token, ".")[1] {
		t.Fatal("raw secret stored")
	}
	if _, err := p.RotateGatewayCredential("acme", "billing", issued.Credential.ID, "alice", 0); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("cross-app rotation accepted", err)
	}
	if err := p.RevokeGatewayCredential("other", "orders", issued.Credential.ID); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("cross-vendor revoke accepted", err)
	}
	next, err := p.RotateGatewayCredential("acme", "orders", issued.Credential.ID, "alice", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.VerifyGatewayCredential(issued.Token); err != nil {
		t.Fatal("overlap rejected old credential", err)
	}
	if _, err = p.VerifyGatewayCredential(next.Token); err != nil {
		t.Fatal("replacement invalid", err)
	}
	if _, err = p.RotateGatewayCredential("acme", "orders", issued.Credential.ID, "alice", time.Hour); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("old credential rotated twice", err)
	}
	if err = p.RevokeGatewayCredential("acme", "orders", issued.Credential.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = p.VerifyGatewayCredential(issued.Token); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("revoked credential accepted", err)
	}
	last, err := p.RotateGatewayCredential("acme", "orders", next.Credential.ID, "alice", 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.VerifyGatewayCredential(next.Token); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("immediate rotation retained old credential", err)
	}
	if _, err = p.VerifyGatewayCredential(last.Token); err != nil {
		t.Fatal(err)
	}
	if _, err = p.primaryConnection.Exec("UPDATE gateway_credentials SET expires_at=? WHERE id=?", time.Now().Add(-time.Second).Unix(), last.Credential.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = p.VerifyGatewayCredential(last.Token); !errors.Is(err, rubix.ErrGatewayCredentialInvalid) {
		t.Fatal("expired credential accepted", err)
	}
}
func TestGatewayCredentialRotationIsAtomic(t *testing.T) {
	p := credentialFixture(t)
	issued, err := p.CreateGatewayCredential("acme", "orders", "Test", "admin", time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = p.primaryConnection.Exec("CREATE TRIGGER block_credentials BEFORE INSERT ON gateway_credentials BEGIN SELECT RAISE(ABORT, 'test failure'); END;"); err != nil {
		t.Fatal(err)
	}
	if _, err = p.RotateGatewayCredential("acme", "orders", issued.Credential.ID, "admin", 0); err == nil {
		t.Fatal("rotation should fail")
	}
	if _, err = p.VerifyGatewayCredential(issued.Token); err != nil {
		t.Fatal("failed rotation invalidated old token", err)
	}
	if _, err = p.primaryConnection.Exec("DROP TRIGGER block_credentials"); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			_, err := p.RotateGatewayCredential("acme", "orders", issued.Credential.ID, "admin", time.Hour)
			results <- err
		})
	}
	wg.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("concurrent rotations succeeded %d times", successes)
	}
}
