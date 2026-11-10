package sql

import (
	"encoding/json"
	"time"

	"github.com/kubex/rubix-storage/rubix"
)

func (p *Provider) GetVendorAppOrders() (map[string][]string, error) {
	rows, err := p.query("SELECT `vendor_id`,`app_ids` FROM `vendor_app_order`")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var vendorID, data string
		if err := rows.Scan(&vendorID, &data); err != nil {
			return nil, err
		}
		var ids []string
		if err := json.Unmarshal([]byte(data), &ids); err != nil {
			return nil, err
		}
		out[vendorID] = ids
	}
	return out, rows.Err()
}

func (p *Provider) StoreVendorAppOrder(order rubix.VendorAppOrder) error {
	if err := order.Validate(); err != nil {
		return err
	}
	var err error
	if len(order.AppIDs) == 0 {
		_, err = p.primaryConnection.Exec("DELETE FROM `vendor_app_order` WHERE `vendor_id`=?", order.VendorID)
	} else {
		data, mErr := json.Marshal(order.AppIDs)
		if mErr != nil {
			return mErr
		}
		query := "REPLACE INTO `vendor_app_order` (`vendor_id`,`app_ids`,`updated_by`,`updated_at`) VALUES (?,?,?,?)"
		if p.SqlLite {
			query = "INSERT INTO `vendor_app_order` (`vendor_id`,`app_ids`,`updated_by`,`updated_at`) VALUES (?,?,?,?)" +
				" ON CONFLICT(`vendor_id`) DO UPDATE SET `app_ids`=excluded.`app_ids`,`updated_by`=excluded.`updated_by`,`updated_at`=excluded.`updated_at`"
		}
		_, err = p.primaryConnection.Exec(query, order.VendorID, string(data), order.UpdatedBy, time.Now().Unix())
	}
	if err == nil {
		p.update()
	}
	return err
}
