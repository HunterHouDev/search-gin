package service

import (
	"os"
	"strings"
	"testing"
	"time"

	"search-gin/internal/model"

	"github.com/stretchr/testify/assert"
)

// ── 测试辅助 ──

func setupPeerTest(t *testing.T) {
	t.Helper()
	old := GetOSSetting()
	t.Cleanup(func() { SetOSSetting(old) })
	// 保存默认管理器
	oldManager := defaultManager
	t.Cleanup(func() { defaultManager = oldManager })

	defaultManager = &peerManager{
		peers: make(map[string]*Peer),
	}
}

func seedPeer(id, ip string, lastSeen int64) {
	defaultManager.mu.Lock()
	defaultManager.peers[id] = &Peer{
		ID:       id,
		IP:       ip,
		Hostname: ip,
		Port:     "10081",
		Name:     ip,
		LastSeen: lastSeen,
	}
	defaultManager.mu.Unlock()
}

// ── IsClusterEnabled ──

func TestIsClusterEnabled_Default(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{EnableLanDiscovery: nil})
	assert.True(t, IsClusterEnabled())
}

func TestIsClusterEnabled_ExplicitTrue(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	v := true
	SetOSSetting(model.Setting{EnableLanDiscovery: &v})
	assert.True(t, IsClusterEnabled())
}

func TestIsClusterEnabled_ExplicitFalse(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	v := false
	SetOSSetting(model.Setting{EnableLanDiscovery: &v})
	assert.False(t, IsClusterEnabled())
}

// ── initNodeInfo ──

func TestInitNodeInfo_SetsGlobals(t *testing.T) {
	oldSetting := GetOSSetting()
	defer SetOSSetting(oldSetting)
	SetOSSetting(model.Setting{NodeName: "测试节点"})

	oldPort := PortNo
	defer func() { PortNo = oldPort }()
	PortNo = ":10999"

	initNodeInfo()

	assert.Contains(t, LocalNodeHost, ":10999")
	assert.Equal(t, "测试节点", LocalNodeName)
}

func TestInitNodeInfo_FallbackHostname(t *testing.T) {
	hostname, _ := os.Hostname()
	oldSetting := GetOSSetting()
	defer SetOSSetting(oldSetting)
	SetOSSetting(model.Setting{NodeName: ""})

	initNodeInfo()
	assert.Equal(t, hostname, LocalNodeName)
}

// ── GetOnlinePeers / GetPeer ──

func TestGetOnlinePeers_Empty(t *testing.T) {
	setupPeerTest(t)
	assert.Empty(t, GetOnlinePeers())
}

func TestGetOnlinePeers_ReturnsCopy(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	peers := GetOnlinePeers()
	assert.Len(t, peers, 1)
	assert.Equal(t, "pc1:10081", peers[0].ID)

	// 修改返回的副本不应影响原数据
	peers[0].ID = "hacked"
	defaultManager.mu.RLock()
	assert.Equal(t, "pc1:10081", defaultManager.peers["pc1:10081"].ID)
	defaultManager.mu.RUnlock()
}

func TestGetOnlinePeers_Multiple(t *testing.T) {
	setupPeerTest(t)
	seedPeer("a:10081", "10.0.0.1", time.Now().Unix())
	seedPeer("b:10081", "10.0.0.2", time.Now().Unix())

	assert.Len(t, GetOnlinePeers(), 2)
}

func TestGetPeer_ReturnsCorrect(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	p := GetPeer("pc1:10081")
	assert.NotNil(t, p)
	assert.Equal(t, "10.0.0.1", p.IP)
}

func TestGetPeer_NotFound(t *testing.T) {
	setupPeerTest(t)
	assert.Nil(t, GetPeer("nonexistent:10081"))
}

func TestGetPeer_NilManager(t *testing.T) {
	defaultManager = nil
	assert.Nil(t, GetPeer("x:10081"))
}

// ── ResolvePeerIP ──

func TestResolvePeerIP_Found(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	assert.Equal(t, "10.0.0.1", ResolvePeerIP("pc1:10081"))
}

func TestResolvePeerIP_NotFound(t *testing.T) {
	setupPeerTest(t)
	assert.Empty(t, ResolvePeerIP("ghost:10081"))
}

// ── TogglePeerDisabled ──

func TestTogglePeerDisabled_Enable(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	assert.True(t, TogglePeerDisabled("pc1:10081", true))
	p := GetPeer("pc1:10081")
	assert.True(t, p.Disabled)
}

func TestTogglePeerDisabled_Disable(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	TogglePeerDisabled("pc1:10081", true)
	TogglePeerDisabled("pc1:10081", false)

	p := GetPeer("pc1:10081")
	assert.False(t, p.Disabled)
}

func TestTogglePeerDisabled_NotFound(t *testing.T) {
	setupPeerTest(t)
	assert.False(t, TogglePeerDisabled("nonexistent:10081", true))
}

func TestTogglePeerDisabled_NilManager(t *testing.T) {
	defaultManager = nil
	assert.False(t, TogglePeerDisabled("x:10081", true))
}

// ── IsKnownPeerIP ──

func TestIsKnownPeerIP_AcceptLoopback(t *testing.T) {
	assert.True(t, IsKnownPeerIP("127.0.0.1"))
	assert.True(t, IsKnownPeerIP("::1"))
}

func TestIsKnownPeerIP_KnownPeer(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	assert.True(t, IsKnownPeerIP("10.0.0.1"))
}

func TestIsKnownPeerIP_Unknown(t *testing.T) {
	setupPeerTest(t)
	assert.False(t, IsKnownPeerIP("192.168.1.99"))
}

func TestIsKnownPeerIP_DisabledPeerRejected(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())
	TogglePeerDisabled("pc1:10081", true)

	assert.False(t, IsKnownPeerIP("10.0.0.1"))
}

// ── CleanExpiredPeers ──

func TestCleanExpiredPeers_RemovesExpired(t *testing.T) {
	setupPeerTest(t)
	now := time.Now().Unix()
	seedPeer("active:10081", "10.0.0.1", now)
	seedPeer("stale:10081", "10.0.0.2", now-int64(defaultPeerTimeout.Seconds())-100)

	removed := CleanExpiredPeers()
	assert.Equal(t, 1, removed)
	assert.Nil(t, GetPeer("stale:10081"))
	assert.NotNil(t, GetPeer("active:10081"))
}

func TestCleanExpiredPeers_NoExpired(t *testing.T) {
	setupPeerTest(t)
	now := time.Now().Unix()
	seedPeer("a:10081", "10.0.0.1", now)
	seedPeer("b:10081", "10.0.0.2", now)

	assert.Equal(t, 0, CleanExpiredPeers())
}

func TestCleanExpiredPeers_NilManager(t *testing.T) {
	defaultManager = nil
	assert.Equal(t, 0, CleanExpiredPeers())
}

// ── upsertPeer ──

func TestUpsertPeer_AddsNew(t *testing.T) {
	setupPeerTest(t)

	m := defaultManager
	m.upsertPeer(&Peer{ID: "new:10081", IP: "10.0.0.5"})

	assert.NotNil(t, GetPeer("new:10081"))
}

func TestUpsertPeer_UpdatesExisting(t *testing.T) {
	setupPeerTest(t)
	seedPeer("pc1:10081", "10.0.0.1", time.Now().Unix())

	m := defaultManager
	m.upsertPeer(&Peer{ID: "pc1:10081", IP: "10.0.0.99", Name: "updated"})

	p := GetPeer("pc1:10081")
	assert.Equal(t, "10.0.0.99", p.IP)
	assert.Equal(t, "updated", p.Name)
}

// ── loadStaticPeers ──

func TestLoadStaticPeers_LoadsFromSetting(t *testing.T) {
	setupPeerTest(t)
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{
		DiscoveryPeers: []string{"10.0.0.1:10081", "10.0.0.2:10082:10083"},
	})

	loadStaticPeers()

	assert.NotNil(t, GetPeer("10.0.0.1:10081"))
	assert.NotNil(t, GetPeer("10.0.0.2:10082"))
	p := GetPeer("10.0.0.2:10082")
	assert.Equal(t, "10083", p.FilePort)
}

func TestLoadStaticPeers_NilManager(t *testing.T) {
	defaultManager = nil
	loadStaticPeers() // should not panic
}

func TestLoadStaticPeers_InvalidEntry(t *testing.T) {
	setupPeerTest(t)
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{
		DiscoveryPeers: []string{"invalid"},
	})

	loadStaticPeers() // should not panic, entry with no colon skipped
	assert.Empty(t, GetOnlinePeers())
}

// ── TryVerifyAndAddPeer ──

func TestTryVerifyAndAddPeer_NilManager(t *testing.T) {
	defaultManager = nil
	assert.False(t, TryVerifyAndAddPeer("10.0.0.1"))
}

func TestTryVerifyAndAddPeer_DelegatesToAddPeer(t *testing.T) {
	setupPeerTest(t)
	// without a real server, addPeer's verifyPeer will fail
	ok := TryVerifyAndAddPeer("10.0.0.99")
	assert.False(t, ok, "should fail without real server")
}

// ── peerManager.NilGuard ──

func TestNilManagerGuard(t *testing.T) {
	defaultManager = nil
	assert.Nil(t, GetOnlinePeers())
	assert.Empty(t, ResolvePeerIP("x:10081"))
	assert.Empty(t, ResolvePeerIP(""))

	defaultManager = nil
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{
		DiscoveryPeers: []string{"10.0.0.1:10081"}},
	)
	// RemovePeer should handle nil
	ok := RemovePeer("10.0.0.1:10081")
	assert.False(t, ok)
}

// ── GetLocalSubnet ──

func TestGetLocalSubnet_ReturnsNonEmpty(t *testing.T) {
	subnet := GetLocalSubnet()
	// On machines without network, it returns ""
	// Just ensure no panic and valid format if non-empty
	if subnet != "" {
		parts := strings.Split(subnet, ".")
		assert.Equal(t, 3, len(parts), "subnet should be a /24 prefix (3 octets)")
	}
}

// ── GetPeerStats ──

func TestGetPeerStats_NilManager(t *testing.T) {
	defaultManager = nil
	cnt, size, name := GetPeerStats("x:10081")
	assert.Equal(t, 0, cnt)
	assert.Empty(t, size)
	assert.Empty(t, name)
}

func TestGetPeerStats_PeerNotFound(t *testing.T) {
	setupPeerTest(t)
	cnt, size, name := GetPeerStats("ghost:10081")
	assert.Equal(t, 0, cnt)
	assert.Empty(t, size)
	assert.Empty(t, name)
}

// ── checkSingleHost ──

func TestCheckSingleHost_InvalidFormat(t *testing.T) {
	result := checkSingleHost("not.an.ip")
	assert.Nil(t, result)

	result = checkSingleHost("1.2.3")
	assert.Nil(t, result)

	result = checkSingleHost("")
	assert.Nil(t, result)
}

// ── DiscoverLanPeers ──

func TestDiscoverLanPeers_BadSubnetFormat(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{})

	peers, prefix := DiscoverLanPeers("bad.format")
	assert.Empty(t, peers)
	assert.Equal(t, GetLocalSubnet(), prefix)
}

func TestDiscoverLanPeers_InvalidParts(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{})

	peers, prefix := DiscoverLanPeers("999.999.999")
	assert.Empty(t, peers)
	assert.Equal(t, GetLocalSubnet(), prefix)
}

func TestDiscoverLanPeers_EmptySubnetAndNoLocal(t *testing.T) {
	old := GetOSSetting()
	defer SetOSSetting(old)
	SetOSSetting(model.Setting{})

	peers, prefix := DiscoverLanPeers("")
	// if local prefix is empty, should return empty
	localPrefix := GetLocalSubnet()
	if localPrefix == "" {
		assert.Empty(t, peers)
		assert.Empty(t, prefix)
	}
}
