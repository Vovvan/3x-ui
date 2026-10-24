package service

import (
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

const onlineGracePeriodMs = int64(20_000)

func (s *InboundService) GetClientsLastOnline() (map[string]int64, error) {
	db := database.GetDB()
	var rows []xray.ClientTraffic
	if err := db.Model(&xray.ClientTraffic{}).Select("email, last_online").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(rows))
	for _, r := range rows {
		result[r.Email] = r.LastOnline
	}
	return result, nil
}

func (s *InboundService) RefreshLocalOnlineClients(activeEmails []string) {
	if process := currentXrayProcess(); process != nil {
		process.RefreshLocalOnline(activeEmails, time.Now().UnixMilli(), onlineGracePeriodMs)
	}
}
