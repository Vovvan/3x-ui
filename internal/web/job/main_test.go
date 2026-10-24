package job

import (
	"context"
	"os"
	"testing"

	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/runtime"
)

// Job tests drive services that commit before they touch the core; a no-op
// runtime keeps the DB assertions independent of a running Xray.
type noopRuntime struct{}

func (noopRuntime) Name() string { return "noop" }

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

func TestMain(m *testing.M) {
	mgr := runtime.NewManager(runtime.LocalDeps{APIPort: func() int { return 0 }})
	mgr.SetLocalRuntimeOverride(noopRuntime{})
	runtime.SetManager(mgr)
	os.Exit(m.Run())
}
