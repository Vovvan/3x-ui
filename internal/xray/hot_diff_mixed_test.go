package xray

import (
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/util/json_util"
)

func mixedInbound(settings string) InboundConfig {
	return InboundConfig{
		Listen:   json_util.RawMessage(`"0.0.0.0"`),
		Port:     1080,
		Protocol: "mixed",
		Tag:      "inbound-1080",
		Settings: json_util.RawMessage(settings),
	}
}

// A client-only edit must use AlterInbound user ops; replacing the handler
// would drop every live SOCKS/HTTP session on the mixed inbound.
func TestComputeHotDiff_MixedClientOnlyChangeStaysHot(t *testing.T) {
	oldCfg := makeHotConfig()
	oldCfg.InboundConfigs[1] = mixedInbound(`{"auth":"password","udp":false,"clients":[{"email":"a","password":"pa"}]}`)
	newCfg := makeHotConfig()
	newCfg.InboundConfigs[1] = mixedInbound(`{"auth":"password","udp":false,"clients":[{"email":"a","password":"pa"},{"email":"b","password":"pb"}]}`)

	diff, ok := ComputeHotDiff(oldCfg, newCfg)
	if !ok {
		t.Fatal("a mixed client-only change must stay hot-appliable")
	}
	if len(diff.RemovedInboundTags) != 0 || len(diff.AddedInbounds) != 0 {
		t.Fatalf("mixed client-only change must not replace the handler, got %+v", diff)
	}
	if len(diff.AddedUsers) != 1 || diff.AddedUsers[0].Email != "b" || diff.AddedUsers[0].Protocol != "mixed" {
		t.Fatalf("expected user b added via AlterInbound, got %+v", diff.AddedUsers)
	}
	if diff.AddedUsers[0].User["password"] != "pb" {
		t.Fatalf("added user = %#v, want b's password carried through", diff.AddedUsers[0].User)
	}
}

// Static accounts and auth/udp changes cannot be applied through per-user
// ops; the modified core seeds accounts only at handler creation.
func TestComputeHotDiff_MixedStaticSettingsChangeNeedsRestart(t *testing.T) {
	oldCfg := makeHotConfig()
	oldCfg.InboundConfigs[1] = mixedInbound(`{"auth":"password","udp":false,"accounts":[{"user":"a","pass":"pa"}],"clients":[{"email":"a","password":"pa"}]}`)
	newCfg := makeHotConfig()
	newCfg.InboundConfigs[1] = mixedInbound(`{"auth":"password","udp":false,"accounts":[{"user":"a","pass":"pa"},{"user":"b","pass":"pb"}],"clients":[{"email":"a","password":"pa"}]}`)

	if _, ok := ComputeHotDiff(oldCfg, newCfg); ok {
		t.Fatal("a mixed static account change must force a full restart, not a gRPC hot swap")
	}
}
