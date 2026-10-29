package sub

import (
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mhsanaei/3x-ui/v3/internal/database"
	"github.com/mhsanaei/3x-ui/v3/internal/database/model"
	"github.com/mhsanaei/3x-ui/v3/internal/web/service"
)

func TestInboundSettingsCacheHitExpiryAndPurge(t *testing.T) {
	now := time.Unix(1_000_000, 0)
	cache := newInboundSettingsCache(10 * time.Second)
	cache.now = func() time.Time { return now }
	var loads atomic.Int32
	load := func() (string, error) {
		loads.Add(1)
		return `{"clients":[{"email":"a@e"}],"decryption":"none"}`, nil
	}

	got, err := cache.settingsFor(1, load)
	if err != nil {
		t.Fatalf("miss: %v", err)
	}
	if got != `{"decryption":"none"}` {
		t.Fatalf("settings = %s, want the clients array dropped", got)
	}
	if _, err := cache.settingsFor(1, load); err != nil || loads.Load() != 1 {
		t.Fatalf("hit reloaded: loads = %d, err = %v", loads.Load(), err)
	}
	now = now.Add(11 * time.Second)
	if _, err := cache.settingsFor(1, load); err != nil || loads.Load() != 2 {
		t.Fatalf("expired entry must reload: loads = %d, err = %v", loads.Load(), err)
	}
	cache.purgeAll()
	if _, err := cache.settingsFor(1, load); err != nil || loads.Load() != 3 {
		t.Fatalf("purged entry must reload: loads = %d, err = %v", loads.Load(), err)
	}
}

func TestInboundSettingsCacheSingleFlight(t *testing.T) {
	cache := newInboundSettingsCache(time.Minute)
	release := make(chan struct{})
	var loads atomic.Int32
	load := func() (string, error) {
		loads.Add(1)
		<-release
		return `{"method":"aes-128-gcm"}`, nil
	}

	const callers = 8
	var wg sync.WaitGroup
	for range callers {
		wg.Go(func() {
			if got, err := cache.settingsFor(7, load); err != nil || got != `{"method":"aes-128-gcm"}` {
				t.Errorf("settingsFor = %q, %v", got, err)
			}
		})
	}
	// Let every caller reach the shared load before releasing it.
	time.Sleep(100 * time.Millisecond)
	close(release)
	wg.Wait()
	if loads.Load() != 1 {
		t.Fatalf("%d concurrent misses ran %d loads, want 1", callers, loads.Load())
	}
}

// A load that a panel edit overlaps may have read the row before the edit; it is served once but
// must not be kept, or the edit would stay invisible for a whole TTL.
func TestInboundSettingsCacheDropsALoadOverlappingAPurge(t *testing.T) {
	cache := newInboundSettingsCache(time.Minute)
	var loads atomic.Int32
	load := func() (string, error) {
		if loads.Add(1) == 1 {
			cache.purgeAll()
		}
		return `{"decryption":"none"}`, nil
	}
	if _, err := cache.settingsFor(3, load); err != nil {
		t.Fatalf("first read: %v", err)
	}
	if _, err := cache.settingsFor(3, load); err != nil {
		t.Fatalf("second read: %v", err)
	}
	if loads.Load() != 2 {
		t.Fatalf("loads = %d, want 2: a load that overlapped a purge was kept", loads.Load())
	}
}

// Only the settings blob is cached: the inbound row is read on every fetch, so a port change or a
// disabled inbound reaches the next fetch, while settings wait for a panel edit's purge.
func TestSettingsCacheReadsTheInboundRowFresh(t *testing.T) {
	seedSubDB(t)
	ib := seedSubInbound(t, "s1", "fresh", 4611, `{"network":"tcp","security":"none"}`)
	cached := NewSubService("")
	t.Cleanup(cached.enableSettingsCache())
	cached.subscriptionBody = true

	if _, err := cached.getInboundsBySubId("s1"); err != nil {
		t.Fatalf("priming read: %v", err)
	}
	db := database.GetDB()
	if err := db.Model(ib).Updates(map[string]any{"port": 4612, "settings": `{"clients":[],"decryption":"changed"}`}).Error; err != nil {
		t.Fatalf("edit inbound: %v", err)
	}
	inbounds, err := cached.getInboundsBySubId("s1")
	if err != nil {
		t.Fatalf("hit read: %v", err)
	}
	if len(inbounds) != 1 || inbounds[0].Port != 4612 {
		t.Fatalf("inbounds = %+v, want the new port on the next fetch", inbounds)
	}
	if strings.Contains(inbounds[0].Settings, "changed") {
		t.Fatal("settings changed without a purge must still come from the cache")
	}
	service.InvalidateSubData()
	inbounds, err = cached.getInboundsBySubId("s1")
	if err != nil {
		t.Fatalf("post-purge read: %v", err)
	}
	if len(inbounds) != 1 || !strings.Contains(inbounds[0].Settings, "changed") {
		t.Fatalf("settings = %+v, want the edit after a purge", inbounds)
	}

	if err := db.Model(ib).Update("enable", false).Error; err != nil {
		t.Fatalf("disable inbound: %v", err)
	}
	if inbounds, err = cached.getInboundsBySubId("s1"); err != nil || len(inbounds) != 0 {
		t.Fatalf("disabled inbound still served: %d inbounds, err %v", len(inbounds), err)
	}
}

// Host rows have no invalidation hooks, so a cached fetch must still read them fresh: a host added
// between two reads has to show up on the second.
func TestGetInboundsBySubIdLoadsHostsOnCacheHit(t *testing.T) {
	seedSubDB(t)
	ib := seedSubInbound(t, "s1", "hh", 4603, `{"network":"tcp","security":"none"}`)
	cached := NewSubService("")
	t.Cleanup(cached.enableSettingsCache())
	cached.subscriptionBody = true

	if _, err := cached.getInboundsBySubId("s1"); err != nil {
		t.Fatalf("priming read: %v", err)
	}
	seedHost(t, &model.Host{InboundId: ib.Id, Remark: "edge", Address: "edge.example.com", SortOrder: 1})
	if _, err := cached.getInboundsBySubId("s1"); err != nil {
		t.Fatalf("hit read: %v", err)
	}
	if len(cached.hostsByInbound[ib.Id]) != 1 {
		t.Fatalf("cache hit did not load fresh host rows: %+v", cached.hostsByInbound)
	}
}

// Panel link rendering registered an invalidator per render and nothing ever unregistered one; a
// released invalidator must stop running.
func TestSubDataInvalidatorRelease(t *testing.T) {
	var calls atomic.Int32
	release := service.RegisterSubDataInvalidator(func() { calls.Add(1) })
	service.InvalidateSubData()
	release()
	service.InvalidateSubData()
	if calls.Load() != 1 {
		t.Fatalf("invalidator ran %d times, want 1: a released invalidator kept running", calls.Load())
	}
}
