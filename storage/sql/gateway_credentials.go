package sql

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
	"github.com/openbyte-os/sdk-go/app"
)

const credentialColumns = "`id`,`vendor_id`,`app_id`,`name`,`prefix`,`created_by`,`created_at`,`expires_at`,`revoked_at`,`replaced_by`"

func credentialTime(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.UTC().Unix()
}
func credentialDate(t int64) time.Time {
	if t == 0 {
		return time.Time{}
	}
	return time.Unix(t, 0).UTC()
}
func credentialDigest(token string) string {
	d := sha256.Sum256([]byte(token))
	return hex.EncodeToString(d[:])
}
func newCredential(vendor, application, name, actor string, expires time.Time) (rubix.IssuedGatewayCredential, string, error) {
	name = strings.TrimSpace(name)
	if app.NewID(vendor, application).Validate(false) != nil || name == "" || len(name) > 128 || actor == "" || len(actor) > 255 || (!expires.IsZero() && !expires.After(time.Now())) {
		return rubix.IssuedGatewayCredential{}, "", rubix.ErrGatewayCredentialInput
	}
	raw := make([]byte, 48)
	if _, err := rand.Read(raw); err != nil {
		return rubix.IssuedGatewayCredential{}, "", err
	}
	id := base64.RawURLEncoding.EncodeToString(raw[:16])
	token := "rxa_" + id + "." + base64.RawURLEncoding.EncodeToString(raw[16:])
	c := rubix.GatewayCredential{ID: id, VendorID: vendor, AppID: application, Name: name, Prefix: "rxa_" + id, CreatedBy: actor, CreatedAt: time.Now().UTC().Truncate(time.Second), ExpiresAt: expires.UTC()}
	return rubix.IssuedGatewayCredential{Credential: c, Token: token}, credentialDigest(token), nil
}

type credentialScanner interface{ Scan(...any) error }

func scanCredential(row credentialScanner) (rubix.GatewayCredential, error) {
	var c rubix.GatewayCredential
	var created, expires, revoked int64
	err := row.Scan(&c.ID, &c.VendorID, &c.AppID, &c.Name, &c.Prefix, &c.CreatedBy, &created, &expires, &revoked, &c.ReplacedBy)
	c.CreatedAt = credentialDate(created)
	c.ExpiresAt = credentialDate(expires)
	c.RevokedAt = credentialDate(revoked)
	return c, err
}

type credentialExecer interface {
	Exec(string, ...any) (sql.Result, error)
}

func insertCredential(db credentialExecer, c rubix.GatewayCredential, digest string) error {
	_, err := db.Exec("INSERT INTO `gateway_credentials` ("+credentialColumns+",`token_hash`) VALUES (?,?,?,?,?,?,?,?,?,?,?)", c.ID, c.VendorID, c.AppID, c.Name, c.Prefix, c.CreatedBy, credentialTime(c.CreatedAt), credentialTime(c.ExpiresAt), int64(0), "", digest)
	return err
}
func (p *Provider) CreateGatewayCredential(vendor, application, name, actor string, expires time.Time) (rubix.IssuedGatewayCredential, error) {
	issued, digest, err := newCredential(vendor, application, name, actor, expires)
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	var count int
	if err = p.queryRow("SELECT COUNT(*) FROM `platform_applications` WHERE `vendor_id`=? AND `app_id`=?", vendor, application).Scan(&count); err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	if count == 0 {
		return rubix.IssuedGatewayCredential{}, rubix.ErrGatewayCredentialInput
	}
	if err = insertCredential(p.primaryConnection, issued.Credential, digest); err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	return issued, nil
}
func (p *Provider) GetGatewayCredentials(vendor, application string) ([]rubix.GatewayCredential, error) {
	rows, err := p.query("SELECT "+credentialColumns+" FROM `gateway_credentials` WHERE `vendor_id`=? AND `app_id`=? ORDER BY `created_at` DESC, `id`", vendor, application)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []rubix.GatewayCredential{}
	for rows.Next() {
		c, err := scanCredential(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (p *Provider) VerifyGatewayCredential(token string) (*rubix.GatewayCredential, error) {
	if len(token) != 70 || !strings.HasPrefix(token, "rxa_") || token[26] != '.' {
		return nil, rubix.ErrGatewayCredentialInvalid
	}
	var digest string
	err := p.queryRow("SELECT `token_hash` FROM `gateway_credentials` WHERE `id`=?", token[4:26]).Scan(&digest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rubix.ErrGatewayCredentialInvalid
	}
	if err != nil {
		return nil, err
	}
	if subtle.ConstantTimeCompare([]byte(digest), []byte(credentialDigest(token))) != 1 {
		return nil, rubix.ErrGatewayCredentialInvalid
	}
	c, err := scanCredential(p.queryRow("SELECT "+credentialColumns+" FROM `gateway_credentials` WHERE `id`=?", token[4:26]))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, rubix.ErrGatewayCredentialInvalid
	}
	if err != nil {
		return nil, err
	}
	if !c.Active() {
		return nil, rubix.ErrGatewayCredentialInvalid
	}
	return &c, nil
}

// RotateGatewayCredential atomically creates a replacement and retires the old
// credential. A guarded update prevents concurrent rotations or revocations
// from extending its life. Overlap may be 0 (immediate) through 24 hours.
func (p *Provider) RotateGatewayCredential(vendor, application, id, actor string, overlap time.Duration) (rubix.IssuedGatewayCredential, error) {
	if overlap < 0 || overlap > 24*time.Hour {
		return rubix.IssuedGatewayCredential{}, rubix.ErrGatewayCredentialInput
	}
	tx, err := p.primaryConnection.Begin()
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	defer tx.Rollback()
	old, err := scanCredential(tx.QueryRow("SELECT "+credentialColumns+" FROM `gateway_credentials` WHERE `id`=? AND `vendor_id`=? AND `app_id`=?", id, vendor, application))
	if errors.Is(err, sql.ErrNoRows) {
		return rubix.IssuedGatewayCredential{}, rubix.ErrGatewayCredentialInvalid
	}
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	if !old.Active() || old.ReplacedBy != "" {
		return rubix.IssuedGatewayCredential{}, rubix.ErrGatewayCredentialInvalid
	}
	issued, digest, err := newCredential(vendor, application, old.Name, actor, old.ExpiresAt)
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	retire := time.Now().UTC().Add(overlap)
	if !old.ExpiresAt.IsZero() && old.ExpiresAt.Before(retire) {
		retire = old.ExpiresAt
	}
	result, err := tx.Exec("UPDATE `gateway_credentials` SET `expires_at`=?,`replaced_by`=? WHERE `id`=? AND `vendor_id`=? AND `app_id`=? AND `revoked_at`=0 AND `replaced_by`=''", credentialTime(retire), issued.Credential.ID, id, vendor, application)
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	if n != 1 {
		return rubix.IssuedGatewayCredential{}, rubix.ErrGatewayCredentialInvalid
	}
	if err = insertCredential(tx, issued.Credential, digest); err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	if err = tx.Commit(); err != nil {
		return rubix.IssuedGatewayCredential{}, err
	}
	return issued, nil
}
func (p *Provider) RevokeGatewayCredential(vendor, application, id string) error {
	result, err := p.primaryConnection.Exec("UPDATE `gateway_credentials` SET `revoked_at`=? WHERE `id`=? AND `vendor_id`=? AND `app_id`=? AND `revoked_at`=0", time.Now().UTC().Unix(), id, vendor, application)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return rubix.ErrGatewayCredentialInvalid
	}
	return nil
}
