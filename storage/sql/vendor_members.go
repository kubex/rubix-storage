package sql

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

const vendorMemberSelect = "SELECT m.`vendor_id`, m.`user`, m.`is_admin`, m.`is_owner`, m.`added_by`, m.`since`, u.`name`, u.`email` " +
	"FROM `vendor_members` AS m LEFT JOIN `users` AS u ON u.`user` = m.`user` "

func (p *Provider) queryVendorMembers(where, order string, args ...any) ([]rubix.VendorMember, error) {
	rows, err := p.query(vendorMemberSelect+"WHERE "+where+" ORDER BY "+order, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []rubix.VendorMember
	for rows.Next() {
		var m rubix.VendorMember
		since := sql.NullString{}
		name := sql.NullString{}
		email := sql.NullString{}
		if err := rows.Scan(&m.VendorID, &m.UserID, &m.IsAdmin, &m.IsOwner, &m.AddedBy, &since, &name, &email); err != nil {
			return nil, err
		}
		if since.Valid {
			m.Since = timeFromString(since.String)
		}
		m.Name = name.String
		m.Email = email.String
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return members, nil
}

func (p *Provider) GetVendorMembers(vendorID string) ([]rubix.VendorMember, error) {
	return p.queryVendorMembers("m.`vendor_id` = ?", "COALESCE(u.`name`, ''), COALESCE(u.`email`, ''), m.`user`", vendorID)
}

func (p *Provider) GetVendorMember(vendorID, userID string) (*rubix.VendorMember, error) {
	members, err := p.queryVendorMembers("m.`vendor_id` = ? AND m.`user` = ?", "m.`user`", vendorID, userID)
	if err != nil {
		return nil, err
	}
	if len(members) == 0 {
		return nil, nil
	}
	return &members[0], nil
}

func (p *Provider) GetUserVendorMemberships(userID string) ([]rubix.VendorMember, error) {
	return p.queryVendorMembers("m.`user` = ?", "m.`vendor_id`", userID)
}

func (p *Provider) StoreVendorMember(member rubix.VendorMember) error {
	if member.VendorID == "" || member.UserID == "" {
		return errors.New("vendor ID and user ID are required")
	}

	// Owners are always admins
	if member.IsOwner {
		member.IsAdmin = true
	}

	since := member.Since
	if since.IsZero() {
		since = time.Now()
	}
	sinceValue := since.UTC().Format(time.DateTime)

	query := "INSERT INTO `vendor_members` (`vendor_id`, `user`, `is_admin`, `is_owner`, `added_by`, `since`) VALUES (?, ?, ?, ?, ?, ?)"
	if p.SqlLite {
		query += " ON CONFLICT(`vendor_id`, `user`) DO UPDATE SET `is_admin`=excluded.`is_admin`, `is_owner`=excluded.`is_owner`"
	} else {
		query += " ON DUPLICATE KEY UPDATE `is_admin`=VALUES(`is_admin`), `is_owner`=VALUES(`is_owner`)"
	}

	_, err := p.primaryConnection.Exec(query, member.VendorID, member.UserID, member.IsAdmin, member.IsOwner, member.AddedBy, sinceValue)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) RemoveVendorMember(vendorID, userID string) error {
	_, err := p.primaryConnection.Exec("DELETE FROM `vendor_members` WHERE `vendor_id` = ? AND `user` = ?", vendorID, userID)
	if err == nil {
		p.update()
	}
	return err
}

func (p *Provider) scanGlobalUser(row *sql.Row) (*rubix.User, error) {
	var user rubix.User
	name := sql.NullString{}
	email := sql.NullString{}
	if err := row.Scan(&user.UserID, &name, &email); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	user.Name = name.String
	user.Email = email.String
	return &user, nil
}

func (p *Provider) GetUserByEmail(email string) (*rubix.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, nil
	}
	return p.scanGlobalUser(p.queryRow(
		"SELECT `user`, `name`, `email` FROM `users` WHERE LOWER(TRIM(`email`)) = ? ORDER BY `user` LIMIT 1", email))
}

func (p *Provider) GetUserByID(userID string) (*rubix.User, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, nil
	}
	return p.scanGlobalUser(p.queryRow(
		"SELECT `user`, `name`, `email` FROM `users` WHERE `user` = ?", userID))
}
