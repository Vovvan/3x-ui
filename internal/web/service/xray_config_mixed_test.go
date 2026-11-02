package service

import (
	"strings"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
	"github.com/mhsanaei/3x-ui/v3/internal/xray"
)

// A mixed client lives only in the dynamic registry, so its emitted settings
// entry must carry the password buildUserAccount sends as the SOCKS credential.
func TestGetXrayConfig_EmitsMixedClientPasswords(t *testing.T) {
	setupSettingTestDB(t)
	in := &model.Inbound{
		Tag:      "mixed-pass",
		Enable:   true,
		Port:     43201,
		Protocol: model.Mixed,
		Settings: `{"auth":"password","accounts":[],"clients":[]}`,
	}
	if err := database.GetDB().Create(in).Error; err != nil {
		t.Fatalf("create mixed inbound: %v", err)
	}
	if err := (&ClientService{}).SyncInbound(nil, in.Id, []model.Client{
		{Email: "ivy@example.test", Password: "pw-ivy", Enable: true},
	}); err != nil {
		t.Fatalf("SyncInbound: %v", err)
	}

	value, present := emittedClients(t, "mixed-pass")
	if !present {
		t.Fatal("settings.clients must be present on a mixed inbound with clients")
	}
	list, ok := value.([]any)
	if !ok || len(list) != 1 {
		t.Fatalf("settings.clients = %#v, want one client", value)
	}
	entry, _ := list[0].(map[string]any)
	if entry["email"] != "ivy@example.test" || entry["password"] != "pw-ivy" {
		t.Fatalf("emitted mixed client = %#v, want email and password", entry)
	}
}

func TestFillProtocolDefaults_MintsMixedPassword(t *testing.T) {
	client := model.Client{Email: "ivy@example.test"}
	if err := (&ClientService{}).fillProtocolDefaults(&client, &model.Inbound{Protocol: model.Mixed}); err != nil {
		t.Fatalf("fillProtocolDefaults: %v", err)
	}
	if len(client.Password) != 32 {
		t.Fatalf("Password = %q, want a minted 32-character password", client.Password)
	}
	kept := model.Client{Email: "ivy@example.test", Password: "chosen"}
	if err := (&ClientService{}).fillProtocolDefaults(&kept, &model.Inbound{Protocol: model.Mixed}); err != nil {
		t.Fatalf("fillProtocolDefaults: %v", err)
	}
	if kept.Password != "chosen" {
		t.Fatalf("Password = %q, want the chosen password kept", kept.Password)
	}
}

// Mixed serves HTTP on the same port, and Basic auth cannot carry a colon in
// the username, which is the client email.
func TestCreateClient_RefusesColonEmailOnMixed(t *testing.T) {
	setupConflictDB(t)
	in := &model.Inbound{
		Tag:      "mixed-colon",
		Enable:   true,
		Port:     43202,
		Protocol: model.Mixed,
		Settings: `{"auth":"password","accounts":[],"clients":[]}`,
	}
	if _, _, err := (&InboundService{}).AddInbound(in); err != nil {
		t.Fatalf("AddInbound: %v", err)
	}

	_, err := (&ClientService{}).Create(&InboundService{}, &ClientCreatePayload{
		Client:     model.Client{Email: "tg:42", Enable: true},
		InboundIds: []int{in.Id},
	})
	if want := `mixed: client email "tg:42" cannot contain ":"`; err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("Create err = %v, want one containing %q", err, want)
	}
}

func TestMixedUserOps_CarriesOnlyMixedClients(t *testing.T) {
	cfg := &xray.Config{InboundConfigs: []xray.InboundConfig{
		{Tag: "mixed-1", Protocol: "mixed", Settings: json_util.RawMessage(`{"clients":[{"email":"ivy@example.test","password":"pw-ivy"},{"email":"empty@example.test"},{"password":"pw-no-email"}]}`)},
		{Tag: "vless-1", Protocol: "vless", Settings: json_util.RawMessage(`{"clients":[{"email":"other@example.test","id":"uuid"}]}`)},
	}}
	ops := mixedUserOps(cfg)
	if len(ops) != 1 {
		t.Fatalf("mixedUserOps returned %d ops, want only the mixed client", len(ops))
	}
	if ops[0].Tag != "mixed-1" || ops[0].Protocol != "mixed" || ops[0].Email != "ivy@example.test" || ops[0].User["password"] != "pw-ivy" {
		t.Fatalf("op = %#v, want mixed-1/ivy with its password", ops[0])
	}
}

func TestBuildTargetClientFromSource_MixedUsesPassword(t *testing.T) {
	target, err := (&InboundService{}).buildTargetClientFromSource(
		model.Client{ID: "source-uuid", Password: "source-pass"},
		&model.Inbound{Protocol: model.Mixed},
		"ivy@example.test",
		"",
	)
	if err != nil {
		t.Fatalf("buildTargetClientFromSource: %v", err)
	}
	if target.ID != "" || len(target.Password) != 32 {
		t.Fatalf("target = %#v, want a fresh password and no UUID", target)
	}
}
