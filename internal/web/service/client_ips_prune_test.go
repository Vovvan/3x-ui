package service

import (
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
)

// Rows for clients absent from the online scan are never rewritten, so the
// sweep is the only thing standing between them and indefinite retention.
func TestPruneStaleClientIpsExpiresUnobservedRows(t *testing.T) {
	setupClientIpTestDB(t)
	db := database.GetDB()

	now := time.Now().Unix()
	stale := now - clientIpStaleAfterSeconds - 300
	fresh := now - 60

	seed := []any{
		&model.InboundClientIps{ClientEmail: "offline", Ips: marshalIps(t, model.ClientIpEntry{IP: "198.51.100.7", Timestamp: stale})},
		&model.InboundClientIps{ClientEmail: "mixed", Ips: marshalIps(t,
			model.ClientIpEntry{IP: "198.51.100.8", Timestamp: stale},
			model.ClientIpEntry{IP: "203.0.113.9", Timestamp: fresh})}}
	for _, row := range seed {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}

	if err := (&InboundService{}).PruneStaleClientIps(); err != nil {
		t.Fatalf("PruneStaleClientIps: %v", err)
	}

	if _, exists := readClientIps(t, "offline"); exists {
		t.Fatal("fully stale inbound_client_ips row must be deleted")
	}
	got, exists := readClientIps(t, "mixed")
	if !exists {
		t.Fatal("row with a fresh entry must survive")
	}
	if len(got) != 1 || got["203.0.113.9"] != fresh {
		t.Fatalf("mixed row = %v, want only 203.0.113.9@%d", got, fresh)
	}

}
