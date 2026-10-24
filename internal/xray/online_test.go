package xray

import (
	"slices"
	"testing"
)

func newOnlineTestProcess() *Process {
	return &Process{newProcess(nil)}
}

func assertSameSet(t *testing.T, label string, got, want []string) {
	t.Helper()
	g := append([]string(nil), got...)
	w := append([]string(nil), want...)
	slices.Sort(g)
	slices.Sort(w)
	if !slices.Equal(g, w) {
		t.Errorf("%s = %v, want %v", label, got, want)
	}
}

// TestGetOnlineClientsReturnsLocalActivity confirms the client view is built
// only from this panel's xray activity.
func TestGetOnlineClientsReturnsLocalActivity(t *testing.T) {
	p := newOnlineTestProcess()
	p.RefreshLocalOnline([]string{"user1", "user2"}, 1000, 20000)

	assertSameSet(t, "online", p.GetOnlineClients(), []string{"user1", "user2"})
}

// TestRefreshLocalOnlineGraceWindow checks the in-memory local set honours the
// grace window: idle-but-recent clients stay online, stale ones age out, and
// the set is derived only from local activity (never the shared DB column).
func TestRefreshLocalOnlineGraceWindow(t *testing.T) {
	p := newOnlineTestProcess()
	const grace = 20000

	p.RefreshLocalOnline([]string{"user1"}, 1000, grace)
	if got := p.GetOnlineClients(); !slices.Contains(got, "user1") {
		t.Fatalf("user1 should be online right after activity, got %v", got)
	}

	p.RefreshLocalOnline([]string{"user2"}, 11000, grace)
	got := p.GetOnlineClients()
	if !slices.Contains(got, "user1") || !slices.Contains(got, "user2") {
		t.Fatalf("both within grace window, got %v", got)
	}

	p.RefreshLocalOnline(nil, 22000, grace)
	got = p.GetOnlineClients()
	if slices.Contains(got, "user1") {
		t.Errorf("user1 (idle 21s, past grace) should have aged out, got %v", got)
	}
	if !slices.Contains(got, "user2") {
		t.Errorf("user2 (idle 11s, within grace) should still be online, got %v", got)
	}
}

// TestOnlineAPISupportTriState pins the lazy capability probe contract: a new
// process starts Unknown (so the first caller probes), and the flag holds
// whatever the probe recorded until the process is replaced on restart.
func TestOnlineAPISupportTriState(t *testing.T) {
	p := newOnlineTestProcess()
	if got := p.OnlineAPISupport(); got != OnlineAPIUnknown {
		t.Fatalf("new process must start with OnlineAPIUnknown, got %v", got)
	}
	p.SetOnlineAPISupport(OnlineAPISupported)
	if got := p.OnlineAPISupport(); got != OnlineAPISupported {
		t.Fatalf("expected OnlineAPISupported, got %v", got)
	}
	p.SetOnlineAPISupport(OnlineAPIUnsupported)
	if got := p.OnlineAPISupport(); got != OnlineAPIUnsupported {
		t.Fatalf("expected OnlineAPIUnsupported, got %v", got)
	}
}
