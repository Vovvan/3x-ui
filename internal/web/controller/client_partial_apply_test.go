package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/dbtest"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/entity"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

type noopRuntime struct{}

func (noopRuntime) Name() string                                     { return "test" }
func (noopRuntime) AddInbound(context.Context, *model.Inbound) error { return nil }
func (noopRuntime) DelInbound(context.Context, *model.Inbound) error { return nil }
func (noopRuntime) UpdateInbound(context.Context, *model.Inbound, *model.Inbound) error {
	return nil
}
func (noopRuntime) AddUser(context.Context, *model.Inbound, map[string]any) error { return nil }
func (noopRuntime) RemoveUser(context.Context, *model.Inbound, string) error      { return nil }
func (noopRuntime) UpdateUser(context.Context, *model.Inbound, string, model.Client) error {
	return nil
}
func (noopRuntime) DeleteUser(context.Context, *model.Inbound, string) error { return nil }
func (noopRuntime) AddClient(context.Context, *model.Inbound, model.Client) error {
	return nil
}
func (noopRuntime) RestartXray(context.Context) error { return nil }
func (noopRuntime) ResetClientTraffic(context.Context, *model.Inbound, string) error {
	return nil
}
func (noopRuntime) ResetInboundTraffic(context.Context, *model.Inbound) error { return nil }
func (noopRuntime) ResetAllTraffics(context.Context) error                    { return nil }

// seedPartlyApplyingClient puts one client on two inbounds and corrupts the second
// one's settings, so a later op succeeds on one inbound and fails on the other.
func seedPartlyApplyingClient(t *testing.T, email string, basePort int) (healthyID, brokenID int) {
	t.Helper()
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	dbtest.InitDB(t, filepath.Join(dbDir, "x-ui.db"))
	mgr := runtime.NewManager(runtime.LocalDeps{})
	mgr.SetLocalRuntimeOverride(noopRuntime{})
	runtime.SetManager(mgr)
	t.Cleanup(func() { runtime.SetManager(nil) })

	db := database.GetDB()
	ids := make([]int, 0, 2)
	for i := range 2 {
		ib := &model.Inbound{
			UserId: 1, Enable: true, Port: basePort + i,
			Tag:      "in-" + string(rune('a'+i)) + "-partial",
			Protocol: model.VLESS, Settings: `{"clients": []}`,
			StreamSettings: `{"network":"tcp","security":"none"}`,
		}
		if err := db.Create(ib).Error; err != nil {
			t.Fatalf("create inbound %d: %v", i, err)
		}
		ids = append(ids, ib.Id)
	}

	if _, err := (&service.ClientService{}).Create(&service.InboundService{}, &service.ClientCreatePayload{
		Client:     model.Client{Email: email, ID: "11111111-2222-3333-4444-555555555555", SubID: "sub-" + email, Enable: true},
		InboundIds: ids,
	}); err != nil {
		t.Fatalf("seed Create across both inbounds: %v", err)
	}

	if err := db.Model(&model.Inbound{}).Where("id = ?", ids[1]).
		Update("settings", `{"clients":`).Error; err != nil {
		t.Fatalf("corrupt inbound %d settings: %v", ids[1], err)
	}
	return ids[0], ids[1]
}

func postCtx(t *testing.T, email string, body any) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{{Key: "email", Value: email}}
	payload := []byte("{}")
	if body != nil {
		var err error
		if payload, err = json.Marshal(body); err != nil {
			t.Fatalf("marshal body: %v", err)
		}
	}
	c.Request = httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(payload))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, w
}

// assertPartialApply pins that the op really failed on one inbound, so a green
// test cannot be a plain full success that never exercised the error path.
func assertPartialApply(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	var msg entity.Msg
	if err := json.Unmarshal(w.Body.Bytes(), &msg); err != nil {
		t.Fatalf("decode response %q: %v", w.Body.String(), err)
	}
	if msg.Success {
		t.Fatalf("response reports success=true, want the partial apply to report failure: %q", w.Body.String())
	}
}

// TestUpdateHandlerReportsPartialApply pins that an edit committed on some
// inbounds and failed on others reports the failure instead of a full success.
func TestUpdateHandlerReportsPartialApply(t *testing.T) {
	const email = "partial-update@example.com"
	seedPartlyApplyingClient(t, email, 43310)

	a := &ClientController{}
	a.xrayService.IsNeedRestartAndSetFalse()
	c, w := postCtx(t, email, map[string]any{
		"email": email, "id": "11111111-2222-3333-4444-555555555555",
		"subId": "sub-" + email, "enable": true, "comment": "edited",
	})
	a.update(c)

	assertPartialApply(t, w)
}

// TestDeleteHandlerReportsPartialApply is the delete-side twin.
func TestDeleteHandlerReportsPartialApply(t *testing.T) {
	const email = "partial-delete@example.com"
	seedPartlyApplyingClient(t, email, 43320)

	a := &ClientController{}
	a.xrayService.IsNeedRestartAndSetFalse()
	c, w := postCtx(t, email, nil)
	a.delete(c)

	assertPartialApply(t, w)
}

// TestImportHandlerReportsTrafficRestoreFailure: the traffic restore runs after
// the clients are committed, so its failure must be reported.
func TestImportHandlerReportsTrafficRestoreFailure(t *testing.T) {
	dbDir := t.TempDir()
	t.Setenv("XUI_DB_FOLDER", dbDir)
	dbtest.InitDB(t, filepath.Join(dbDir, "x-ui.db"))
	mgr := runtime.NewManager(runtime.LocalDeps{})
	mgr.SetLocalRuntimeOverride(noopRuntime{})
	runtime.SetManager(mgr)
	t.Cleanup(func() { runtime.SetManager(nil) })
	db := database.GetDB()
	ib := &model.Inbound{
		UserId: 1, Enable: true, Port: 43340, Tag: "in-import-partial",
		Protocol: model.VLESS, Settings: `{"clients": []}`,
		StreamSettings: `{"network":"tcp","security":"none"}`,
	}
	if err := db.Create(ib).Error; err != nil {
		t.Fatalf("create inbound: %v", err)
	}
	trigger := `CREATE TRIGGER fail_traffic_restore BEFORE UPDATE OF up ON client_traffics
		BEGIN SELECT RAISE(ABORT, 'injected traffic restore failure'); END`
	if err := db.Exec(trigger).Error; err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	const email = "import-partial@example.com"
	data, err := json.Marshal([]service.ClientCreatePayload{{
		Client:     model.Client{Email: email, SubID: "sub-import-partial", Enable: true},
		InboundIds: []int{ib.Id},
		Traffic:    &service.ClientPortableTraffic{Up: 5, Down: 6},
	}})
	if err != nil {
		t.Fatalf("marshal import data: %v", err)
	}
	a := &ClientController{}
	a.xrayService.IsNeedRestartAndSetFalse()
	c, w := postCtx(t, "", importClientsRequest{Data: string(data)})
	a.importClients(c)

	assertPartialApply(t, w)
	var created int64
	if err := db.Model(&model.ClientRecord{}).Where("email = ?", email).Count(&created).Error; err != nil {
		t.Fatalf("count imported client: %v", err)
	}
	if created != 1 {
		t.Fatalf("imported client count=%d, want 1 committed before the restore failed", created)
	}
}

// TestDetachHandlerFlagsRestartOnPartialApply covers the third converted path.
func TestDetachHandlerFlagsRestartOnPartialApply(t *testing.T) {
	const email = "partial-detach@example.com"
	healthyID, brokenID := seedPartlyApplyingClient(t, email, 43330)

	a := &ClientController{}
	a.xrayService.IsNeedRestartAndSetFalse()
	c, w := postCtx(t, email, attachDetachBody{InboundIds: []int{healthyID, brokenID}})
	a.detach(c)

	assertPartialApply(t, w)
	if !a.xrayService.IsNeedRestartAndSetFalse() {
		t.Fatal("a partly-applied client detach left Xray unflagged for restart")
	}
}
