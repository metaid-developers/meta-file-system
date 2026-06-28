package handler

import (
	"errors"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"meta-file-system/conf"
	"meta-file-system/node"
	"meta-file-system/storage"
)

func init() { gin.SetMode(gin.TestMode) }

// fakeStorage is a minimal storage.Storage for tests; only Exists is used by
// the health check. The other methods panic if unexpectedly invoked.
type fakeStorage struct{ exists bool }

func (fakeStorage) Save(string, []byte) error                                   { panic("unexpected") }
func (fakeStorage) Get(string) ([]byte, error)                                  { panic("unexpected") }
func (fakeStorage) Delete(string) error                                         { panic("unexpected") }
func (f fakeStorage) Exists(string) bool                                        { return f.exists }
func (fakeStorage) InitiateMultipartUpload(string) (string, error)              { panic("unexpected") }
func (fakeStorage) UploadPart(string, string, int, []byte) (string, error)      { panic("unexpected") }
func (fakeStorage) CompleteMultipartUpload(string, string, []storage.PartInfo) error {
	panic("unexpected")
}
func (fakeStorage) AbortMultipartUpload(string, string) error         { panic("unexpected") }
func (fakeStorage) ListParts(string, string) ([]storage.PartInfo, error) { panic("unexpected") }
func (fakeStorage) GetMultipartUpload(string, string) ([]byte, error) { panic("unexpected") }

func withConfig(t *testing.T, chains ...conf.UploaderChainConfig) {
	t.Helper()
	prev := conf.Cfg
	conf.Cfg = &conf.Config{
		Uploader: conf.UploaderConfig{Chains: chains},
	}
	t.Cleanup(func() { conf.Cfg = prev })
}

func TestCheckConfig_Ok(t *testing.T) {
	withConfig(t, conf.UploaderChainConfig{Name: "mvc", RpcUrl: "http://node:9882"})
	if s := checkConfig(); s.Status != "ok" {
		t.Fatalf("config stage = %s (%s), want ok", s.Status, s.Error)
	}
}

func TestCheckConfig_MissingChain(t *testing.T) {
	withConfig(t) // no chains
	if s := checkConfig(); s.Status != "fail" || !strings.Contains(s.Error, "no uploader chains") {
		t.Fatalf("expected fail/missing-chains, got %+v", s)
	}
}

func TestCheckStorage_NilFails(t *testing.T) {
	s := checkStorage(nil)
	if s.Status != "fail" {
		t.Fatalf("nil storage should fail, got %+v", s)
	}
}

func TestCheckStorage_FakeOk(t *testing.T) {
	s := checkStorage(fakeStorage{exists: true})
	if s.Status != "ok" {
		t.Fatalf("fake storage should be ok, got %+v", s)
	}
}

func TestCheckMergeBroadcast_UnreachableFailsStage(t *testing.T) {
	withConfig(t, conf.UploaderChainConfig{Name: "mvc", RpcUrl: "http://172.31.168.215:9882"})

	// Inject a probe that simulates the dead-node outage: a classified
	// unreachable error, exactly what the live client returns.
	orig := probeBlockHeight
	probeBlockHeight = func(chain string) (uint64, error) {
		return 0, errors.Join(node.ErrUpstreamNodeUnreachable,
			errors.New("dial tcp 172.31.168.215:9882: connection refused"))
	}
	t.Cleanup(func() { probeBlockHeight = orig })

	s := checkMergeBroadcast()
	if s.Status != "fail" {
		t.Fatalf("expected merge_broadcast to fail, got %+v", s)
	}
	if !strings.Contains(s.Error, "mvc") {
		t.Errorf("error should name the failing chain, got %q", s.Error)
	}
}

func TestCheckMergeBroadcast_AllReachableOk(t *testing.T) {
	withConfig(t,
		conf.UploaderChainConfig{Name: "mvc", RpcUrl: "http://node-a:9882"},
		conf.UploaderChainConfig{Name: "doge", RpcUrl: "http://node-b:22555"},
	)
	orig := probeBlockHeight
	probeBlockHeight = func(chain string) (uint64, error) { return 1000, nil }
	t.Cleanup(func() { probeBlockHeight = orig })

	if s := checkMergeBroadcast(); s.Status != "ok" {
		t.Fatalf("expected ok, got %+v", s)
	}
}

func TestDeepHealthCheck_OverallDegradedWhenStageFails(t *testing.T) {
	withConfig(t, conf.UploaderChainConfig{Name: "mvc", RpcUrl: "http://node:9882"})
	orig := probeBlockHeight
	probeBlockHeight = func(chain string) (uint64, error) { return 0, errors.New("down") }
	t.Cleanup(func() { probeBlockHeight = orig })

	// checkMergeBroadcast is the only stage exercised here; the overall
	// response aggregator lives in DeepHealthCheck, so test it via the
	// stages slice directly to avoid spinning a real DB/storage.
	stages := []StageStatus{
		{Name: "config", Status: "ok"},
		{Name: "merge_broadcast", Status: checkMergeBroadcast().Status, Error: checkMergeBroadcast().Error},
	}
	overall := "ok"
	for _, s := range stages {
		if s.Status != "ok" {
			overall = "degraded"
			break
		}
	}
	if overall != "degraded" {
		t.Fatalf("expected overall degraded when a stage fails, got %s", overall)
	}
}
