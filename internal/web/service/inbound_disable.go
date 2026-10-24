package service

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"

	"gorm.io/gorm"
)

func (s *InboundService) disableInvalidInbounds(tx *gorm.DB, mutationBatch *trafficMutationBatch) (bool, int64, error) {
	now := time.Now().Unix() * 1000
	var inbounds []model.Inbound
	if err := tx.Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Find(&inbounds).Error; err != nil {
		return false, 0, err
	}
	for i := range inbounds {
		mutationBatch.localPlans = append(mutationBatch.localPlans, trafficLocalApplyPlan{
			action: trafficDisableInbound, inbound: inbounds[i],
		})
	}

	result := tx.Model(model.Inbound{}).
		Where("((total > 0 and up + down >= total) or (expiry_time > 0 and expiry_time <= ?)) and enable = ?", now, true).
		Update("enable", false)
	err := result.Error
	count := result.RowsAffected
	return false, count, err
}

// depletedClientsCond matches clients that exhausted their quota or expired.
// Placeholder: now.
const depletedClientsCond = `((total > 0 AND up + down >= total)
	OR (expiry_time > 0 AND expiry_time <= ?))`

func (s *InboundService) disableInvalidClients(tx *gorm.DB, mutationBatch *trafficMutationBatch) (bool, int64, []int, error) {
	now := time.Now().UnixMilli()
	nowMS := time.Now().UnixMilli()
	var depletedRows []xray.ClientTraffic
	err := tx.Model(xray.ClientTraffic{}).
		Where(depletedClientsCond+" AND enable = ?", nowMS, true).
		Find(&depletedRows).Error
	if err != nil {
		return false, 0, nil, err
	}
	if len(depletedRows) == 0 {
		return false, 0, nil, nil
	}

	depletedEmails := make([]string, 0, len(depletedRows))
	for i := range depletedRows {
		if depletedRows[i].Email == "" {
			continue
		}
		depletedEmails = append(depletedEmails, depletedRows[i].Email)
	}

	type target struct {
		InboundID int `gorm:"column:inbound_id"`
		Tag       string
		Email     string
	}
	var targets []target
	if len(depletedEmails) > 0 {
		err = tx.Raw(`
			SELECT inbounds.id AS inbound_id,
			       inbounds.tag AS tag, clients.email AS email
			FROM clients
			JOIN client_inbounds ON client_inbounds.client_id = clients.id
			JOIN inbounds        ON inbounds.id = client_inbounds.inbound_id
			WHERE clients.email IN ?
		`, depletedEmails).Scan(&targets).Error
		if err != nil {
			return false, 0, nil, err
		}
	}

	byInbound := make(map[int][]target)
	for _, t := range targets {
		byInbound[t.InboundID] = append(byInbound[t.InboundID], t)
	}

	for inboundID, group := range byInbound {
		emails := make(map[string]struct{}, len(group))
		for _, t := range group {
			emails[t.Email] = struct{}{}
		}
		_, inbound, mErr := s.markClientsDisabledInSettings(tx, inboundID, emails)
		if mErr != nil {
			return false, 0, nil, mErr
		}
		for email := range emails {
			mutationBatch.localPlans = append(mutationBatch.localPlans, trafficLocalApplyPlan{
				action: trafficRemoveUser, inbound: *inbound, email: email,
			})
		}
	}
	// Flip the rows already collected above by primary key instead of
	// re-evaluating the depleted predicate, which was a second full scan of
	// client_traffics on every poll. Sorted ids keep the lock order stable.
	ids := make([]int, 0, len(depletedRows))
	for i := range depletedRows {
		ids = append(ids, depletedRows[i].Id)
	}
	slices.Sort(ids)
	var count int64
	for _, batch := range chunkInts(ids, sqlInChunk) {
		result := tx.Model(xray.ClientTraffic{}).
			Where("id IN ? AND enable = ?", batch, true).
			Update("enable", false)
		if result.Error != nil {
			return false, count, nil, result.Error
		}
		count += result.RowsAffected
	}

	if len(depletedEmails) > 0 {
		if err := tx.Model(&model.ClientRecord{}).
			Where("email IN ?", depletedEmails).
			Updates(map[string]any{"enable": false, "updated_at": now}).Error; err != nil {
			return false, count, nil, err
		}
	}

	return false, count, nil, nil
}

// markClientsDisabledInSettings flips client.enable=false in the inbound's
// stored settings JSON for the given emails and returns the post snapshot.
func (s *InboundService) markClientsDisabledInSettings(tx *gorm.DB, inboundID int, emails map[string]struct{}) (oldIb, newIb *model.Inbound, err error) {
	var ib model.Inbound
	if err := tx.Model(&model.Inbound{}).Where("id = ?", inboundID).First(&ib).Error; err != nil {
		return nil, nil, err
	}
	snapshot := ib

	settings := map[string]any{}
	if err := json.Unmarshal([]byte(ib.Settings), &settings); err != nil {
		return nil, nil, err
	}
	clients, _ := settings["clients"].([]any)
	now := time.Now().Unix() * 1000
	mutated := false
	for i := range clients {
		entry, ok := clients[i].(map[string]any)
		if !ok {
			continue
		}
		email, _ := entry["email"].(string)
		if _, hit := emails[email]; !hit {
			continue
		}
		if cur, _ := entry["enable"].(bool); !cur {
			continue
		}
		entry["enable"] = false
		entry["updated_at"] = now
		clients[i] = entry
		mutated = true
	}
	if !mutated {
		return &snapshot, &ib, nil
	}
	settings["clients"] = clients
	bs, marshalErr := json.MarshalIndent(settings, "", "  ")
	if marshalErr != nil {
		return nil, nil, marshalErr
	}
	ib.Settings = string(bs)
	if err := tx.Model(&model.Inbound{}).Where("id = ?", inboundID).
		Update("settings", ib.Settings).Error; err != nil {
		return nil, nil, err
	}
	// A just-disabled client must leave subscriptions within this tick.
	InvalidateSubData()
	return &snapshot, &ib, nil
}

// trafficDisabledEmails reports which emails have a switched-off stats row. The
// table is email-keyed and its inbound_id goes stale, so never filter on it.
func trafficDisabledEmails(db *gorm.DB, emails []string) (map[string]struct{}, error) {
	disabled := make(map[string]struct{})
	for _, batch := range chunkStrings(uniqueNonEmptyStrings(emails), sqlInChunk) {
		var page []string
		if err := db.Model(xray.ClientTraffic{}).
			Where("email IN ? AND enable = ?", batch, false).
			Pluck("email", &page).Error; err != nil {
			return nil, err
		}
		for _, e := range page {
			disabled[e] = struct{}{}
		}
	}
	return disabled, nil
}
