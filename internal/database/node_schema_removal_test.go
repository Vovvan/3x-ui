package database

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestInitDBRemovesLegacyNodeSchema(t *testing.T) {
	dbPath := openLegacyPortUniqueDB(t)
	raw, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		t.Fatalf("open legacy db: %v", err)
	}
	for _, stmt := range []string{
		"ALTER TABLE inbounds ADD COLUMN node_id integer",
		"ALTER TABLE inbounds ADD COLUMN origin_node_guid text",
		"CREATE TABLE nodes (id integer PRIMARY KEY AUTOINCREMENT, name text)",
		"CREATE TABLE node_client_traffics (id integer PRIMARY KEY AUTOINCREMENT, email text)",
		"CREATE TABLE node_client_ips (id integer PRIMARY KEY AUTOINCREMENT, email text)",
		"CREATE TABLE node_pending_resets (id integer PRIMARY KEY AUTOINCREMENT, email text)",
		"CREATE TABLE client_global_traffics (id integer PRIMARY KEY AUTOINCREMENT, email text)",
	} {
		if err := raw.Exec(stmt).Error; err != nil {
			t.Fatalf("prepare legacy schema %q: %v", stmt, err)
		}
	}
	if err := raw.AutoMigrate(&model.Host{}, &model.ClientRecord{}, &model.ApiToken{}); err != nil {
		t.Fatalf("create current tables: %v", err)
	}
	for _, stmt := range []string{
		"ALTER TABLE hosts ADD COLUMN node_guids text",
		"ALTER TABLE clients ADD COLUMN sync_orphaned_at integer",
		"INSERT INTO api_tokens (name, token, scope) VALUES ('node', 'x', 'node-sync')",
	} {
		if err := raw.Exec(stmt).Error; err != nil {
			t.Fatalf("prepare legacy columns %q: %v", stmt, err)
		}
	}
	sqlDB, err := raw.DB()
	if err != nil {
		t.Fatalf("legacy db handle: %v", err)
	}
	if err := sqlDB.Close(); err != nil {
		t.Fatalf("close legacy db: %v", err)
	}

	if err := InitDB(dbPath); err != nil {
		t.Fatalf("InitDB: %v", err)
	}
	t.Cleanup(func() { _ = CloseDB() })

	for _, table := range []string{"nodes", "node_client_traffics", "node_client_ips", "node_pending_resets", "client_global_traffics"} {
		if GetDB().Migrator().HasTable(table) {
			t.Errorf("legacy table %q still exists", table)
		}
	}
	for _, col := range []struct {
		model any
		name  string
	}{
		{&model.Inbound{}, "node_id"},
		{&model.Inbound{}, "origin_node_guid"},
		{&model.Host{}, "node_guids"},
		{&model.ClientRecord{}, "sync_orphaned_at"},
	} {
		if GetDB().Migrator().HasColumn(col.model, col.name) {
			t.Errorf("legacy column %q still exists on %T", col.name, col.model)
		}
	}
	var n int64
	if err := GetDB().Model(&model.ApiToken{}).Where("scope = ?", "node-sync").Count(&n).Error; err != nil {
		t.Fatalf("count node-sync tokens: %v", err)
	}
	if n != 0 {
		t.Fatalf("legacy node-sync tokens = %d, want 0", n)
	}
}
