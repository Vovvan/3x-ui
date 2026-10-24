package service

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// fakeRuntime counts runtime dispatch calls so tests can assert which
// mutation path reached the core.
type fakeRuntime struct {
	addInbound    atomic.Int32
	delInbound    atomic.Int32
	addClient     atomic.Int32
	deleteUser    atomic.Int32
	removeUser    atomic.Int32
	updateInbound atomic.Int32
	updateSubSort atomic.Int32
	updateUser    atomic.Int32
}

func (f *fakeRuntime) Name() string { return "fake-local" }

func (f *fakeRuntime) AddInbound(context.Context, *model.Inbound) error {
	f.addInbound.Add(1)
	return nil
}

func (f *fakeRuntime) DelInbound(context.Context, *model.Inbound) error {
	f.delInbound.Add(1)
	return nil
}

func (f *fakeRuntime) UpdateInbound(context.Context, *model.Inbound, *model.Inbound) error {
	f.updateInbound.Add(1)
	return nil
}

func (f *fakeRuntime) AddUser(context.Context, *model.Inbound, map[string]any) error { return nil }

func (f *fakeRuntime) RemoveUser(context.Context, *model.Inbound, string) error {
	f.removeUser.Add(1)
	return nil
}

func (f *fakeRuntime) UpdateUser(context.Context, *model.Inbound, string, model.Client) error {
	f.updateUser.Add(1)
	return nil
}

func (f *fakeRuntime) DeleteUser(context.Context, *model.Inbound, string) error {
	f.deleteUser.Add(1)
	return nil
}

func (f *fakeRuntime) AddClient(context.Context, *model.Inbound, model.Client) error {
	f.addClient.Add(1)
	return nil
}

func (f *fakeRuntime) RestartXray(context.Context) error { return nil }

func (f *fakeRuntime) ResetClientTraffic(context.Context, *model.Inbound, string) error {
	return nil
}

func (f *fakeRuntime) ResetInboundTraffic(context.Context, *model.Inbound) error {
	return nil
}

func (f *fakeRuntime) ResetAllTraffics(context.Context) error { return nil }

func startSerializedWriter(t *testing.T) {
	t.Helper()
	resetTrafficWriterForTest(t)
	StartTrafficWriter()
}

func useTestRuntimeManager(t *testing.T) *runtime.Manager {
	t.Helper()
	prev := runtime.GetManager()
	mgr := runtime.NewManager(runtime.LocalDeps{APIPort: func() int { return 0 }, SetNeedRestart: func() {}})
	runtime.SetManager(mgr)
	t.Cleanup(func() { runtime.SetManager(prev) })
	return mgr
}

type panicRuntime struct{ fakeRuntime }

func (p *panicRuntime) AddClient(context.Context, *model.Inbound, model.Client) error {
	panic("boom from runtime")
}

// setupFakeRuntime wires the local runtime override.
func setupFakeRuntime(t *testing.T) *fakeRuntime {
	t.Helper()
	fake := &fakeRuntime{}
	useTestRuntimeManager(t).SetLocalRuntimeOverride(fake)
	return fake
}

func fakeRuntimeInbound(t *testing.T, port int, clients []model.Client) *model.Inbound {
	t.Helper()
	if clients == nil {
		clients = []model.Client{}
	}
	ib := &model.Inbound{
		UserId: 1, Tag: fmt.Sprintf("in-%d", port), Enable: true,
		Port: port, Protocol: model.VLESS, Settings: clientsSettings(t, clients),
	}
	if err := database.GetDB().Create(ib).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	if err := (&ClientService{}).SyncInbound(nil, ib.Id, clients); err != nil {
		t.Fatalf("seed SyncInbound: %v", err)
	}
	return ib
}

func initTrafficTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	dbtest.InitDB(t, filepath.Join(dbDir, "x-ui.db"))
	return database.GetDB()
}

func seedClientRow(t *testing.T, email string, inboundId int, up, down, total int64) {
	t.Helper()
	if err := database.GetDB().Create(&xray.ClientTraffic{
		InboundId: inboundId, Email: email, Enable: true, Up: up, Down: down, Total: total,
	}).Error; err != nil {
		t.Fatalf("seed client_traffics %q: %v", email, err)
	}
}

func readTraffic(t *testing.T, db *gorm.DB, email string) xray.ClientTraffic {
	t.Helper()
	var ct xray.ClientTraffic
	if err := db.Model(xray.ClientTraffic{}).Where("email = ?", email).First(&ct).Error; err != nil {
		t.Fatalf("read client_traffics %q: %v", email, err)
	}
	return ct
}

func setupClientIpTestDB(t *testing.T) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	dbtest.InitDB(t, filepath.Join(dbDir, "x-ui.db"))
}

func marshalIps(t *testing.T, entries ...clientIpEntry) string {
	t.Helper()
	b, err := json.Marshal(entries)
	if err != nil {
		t.Fatalf("marshal ips: %v", err)
	}
	return string(b)
}

func readClientIps(t *testing.T, email string) (map[string]int64, bool) {
	t.Helper()
	var row model.InboundClientIps
	err := database.GetDB().Where("client_email = ?", email).First(&row).Error
	if database.IsNotFound(err) {
		return nil, false
	}
	if err != nil {
		t.Fatalf("read client ips for %s: %v", email, err)
	}
	var entries []clientIpEntry
	if row.Ips != "" {
		if err := json.Unmarshal([]byte(row.Ips), &entries); err != nil {
			t.Fatalf("unmarshal stored ips for %s: %v", email, err)
		}
	}
	out := make(map[string]int64, len(entries))
	for _, e := range entries {
		out[e.IP] = e.Timestamp
	}
	return out, true
}
