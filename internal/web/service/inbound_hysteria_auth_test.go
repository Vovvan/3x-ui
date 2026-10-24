package service

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// An inbound save keeps the stored clients, so switching to Hysteria is judged
// on them: ones with no auth would leave an inbound nobody can connect to.
func TestUpdateInbound_RejectsHysteriaClientWithoutAuth(t *testing.T) {
	setupConflictDB(t)
	seedInboundConflict(t, "in-45001-tcp", "0.0.0.0", 45001, model.VLESS,
		`{"network":"tcp"}`, `{"clients":[{"email":"hysteria@x","enable":true,"password":"not-hysteria-auth"}]}`)

	var existing model.Inbound
	if err := database.GetDB().Where("tag = ?", "in-45001-tcp").First(&existing).Error; err != nil {
		t.Fatalf("read seeded row: %v", err)
	}

	update := existing
	update.Protocol = model.Hysteria
	update.Settings = `{"clients":[]}`

	svc := &InboundService{}
	if _, _, err := svc.UpdateInbound(&update); err == nil || !strings.Contains(err.Error(), "empty client ID") {
		t.Fatalf("UpdateInbound error = %v, want empty client ID", err)
	}

	var reloaded model.Inbound
	if err := database.GetDB().First(&reloaded, existing.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Protocol != existing.Protocol {
		t.Fatalf("persisted protocol = %q, want unchanged %q", reloaded.Protocol, existing.Protocol)
	}
	if reloaded.Settings != existing.Settings {
		t.Fatalf("rejected settings were persisted\ngot:  %s\nwant: %s", reloaded.Settings, existing.Settings)
	}
}

func TestUpdateInbound_AllowsHysteriaWithoutClients(t *testing.T) {
	setupConflictDB(t)
	seedInboundConflict(t, "in-45003-udp", "0.0.0.0", 45003, model.Hysteria,
		`{"network":"hysteria"}`, `{"clients":[]}`)

	var existing model.Inbound
	if err := database.GetDB().Where("tag = ?", "in-45003-udp").First(&existing).Error; err != nil {
		t.Fatalf("read seeded row: %v", err)
	}

	update := existing
	update.Remark = "updated without clients"

	svc := &InboundService{}
	if _, _, err := svc.UpdateInbound(&update); err != nil {
		t.Fatalf("UpdateInbound: %v", err)
	}

	var reloaded model.Inbound
	if err := database.GetDB().First(&reloaded, existing.Id).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.Remark != update.Remark {
		t.Fatalf("persisted remark = %q, want %q", reloaded.Remark, update.Remark)
	}
}
