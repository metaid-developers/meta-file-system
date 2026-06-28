package node

import (
	"errors"
	"io"
	"net"
	"net/url"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"meta-file-system/conf"
)

// unreachableErr returns a classified unreachable error identical in shape
// to what the live HTTP client produces for a dead node, so the retry logic
// sees it as transient.
func unreachableErr() error {
	raw := &url.Error{
		Op:  "Post",
		URL: "http://172.31.168.215:9882",
		Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
	}
	return classifyBroadcastError(raw)
}

// withShortBackoff swaps the retry sleep for a no-op for the duration of fn
// so the retry tests stay fast.
func withShortBackoff(t *testing.T, fn func()) {
	t.Helper()
	orig := broadcastRetryBackoff
	broadcastRetryJitter = 0
	broadcastRetryBackoff = time.Millisecond
	defer func() { broadcastRetryBackoff = orig }()
	fn()
}

func TestBroadcastTxResilient_PrimarySuccessNoRetry(t *testing.T) {
	var calls int32
	withShortBackoff(t, func() {
		orig := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			atomic.AddInt32(&calls, 1)
			return "deadbeef", nil
		}
		defer func() { broadcastOnce = orig }()

		txID, err := BroadcastTxResilient("mvc", "txhex")
		if err != nil || txID != "deadbeef" {
			t.Fatalf("got txID=%q err=%v, want deadbeef/nil", txID, err)
		}
		if got := atomic.LoadInt32(&calls); got != 1 {
			t.Fatalf("expected exactly 1 primary call (no retry on success), got %d", got)
		}
	})
}

func TestBroadcastTxResilient_RetriesTransientThenSucceeds(t *testing.T) {
	var calls int32
	withShortBackoff(t, func() {
		orig := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			n := atomic.AddInt32(&calls, 1)
			if n < 2 {
				return "", unreachableErr() // transient, retried
			}
			return "ok", nil
		}
		defer func() { broadcastOnce = orig }()

		txID, err := BroadcastTxResilient("mvc", "txhex")
		if err != nil || txID != "ok" {
			t.Fatalf("got txID=%q err=%v, want ok/nil", txID, err)
		}
		if got := atomic.LoadInt32(&calls); got != 2 {
			t.Fatalf("expected 2 primary calls (1 fail + 1 success), got %d", got)
		}
	})
}

func TestBroadcastTxResilient_RPCErrorNotRetried(t *testing.T) {
	var calls int32
	withShortBackoff(t, func() {
		orig := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			atomic.AddInt32(&calls, 1)
			// RPC validation error must NOT be retried.
			return "", errors.New("[-26]txn-mempool-conflict")
		}
		defer func() { broadcastOnce = orig }()

		_, err := BroadcastTxResilient("mvc", "txhex")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if got := atomic.LoadInt32(&calls); got != 1 {
			t.Fatalf("RPC errors must not be retried, expected 1 call, got %d", got)
		}
	})
}

func TestBroadcastTxResilient_DuplicateTreatedAsSuccess(t *testing.T) {
	var calls int32
	withShortBackoff(t, func() {
		orig := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			atomic.AddInt32(&calls, 1)
			return "", errors.New("[-27]txn-already-known")
		}
		defer func() { broadcastOnce = orig }()

		_, err := BroadcastTxResilient("mvc", "txhex")
		if err != nil {
			t.Fatalf("duplicate-tx should be success, got err=%v", err)
		}
		if got := atomic.LoadInt32(&calls); got != 1 {
			t.Fatalf("duplicate-tx must not be retried, expected 1 call, got %d", got)
		}
	})
}

func TestBroadcastTxResilient_UnreachableNoFallbackClassified(t *testing.T) {
	// No fallback configured for "mvc" here.
	withShortBackoff(t, func() {
		orig := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			return "", unreachableErr()
		}
		defer func() { broadcastOnce = orig }()

		_, err := BroadcastTxResilient("mvc", "txhex")
		if !errors.Is(err, ErrUpstreamNodeUnreachable) {
			t.Fatalf("expected ErrUpstreamNodeUnreachable, got %v", err)
		}
	})
}

func TestBroadcastTxResilient_FallbackSucceedsWhenPrimaryDead(t *testing.T) {
	// Configure a fallback URL for the chain under test.
	origCfg := conf.RpcConfigMap["mvc-fb"]
	conf.RpcConfigMap["mvc-fb"] = conf.RpcConfig{
		Url:         "http://primary.invalid:9882",
		Username:    "u",
		Password:    "p",
		FallbackUrl: "http://fallback.invalid:9882",
	}
	defer func() { conf.RpcConfigMap["mvc-fb"] = origCfg }()

	fallbackCalled := int32(0)
	withShortBackoff(t, func() {
		origPrimary := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			return "", unreachableErr()
		}
		origFallback := broadcastFallback
		broadcastFallback = func(u, user, pass, txHex string) (string, error) {
			atomic.AddInt32(&fallbackCalled, 1)
			if u != "http://fallback.invalid:9882" {
				t.Errorf("fallback called with wrong url %q", u)
			}
			return "fallbacktxid", nil
		}
		defer func() {
			broadcastOnce = origPrimary
			broadcastFallback = origFallback
		}()

		txID, err := BroadcastTxResilient("mvc-fb", "txhex")
		if err != nil || txID != "fallbacktxid" {
			t.Fatalf("expected fallback success, got txID=%q err=%v", txID, err)
		}
		if got := atomic.LoadInt32(&fallbackCalled); got != 1 {
			t.Fatalf("fallback should be called once, got %d", got)
		}
	})
}

func TestBroadcastTxResilient_BothFailReturnsClassified(t *testing.T) {
	origCfg := conf.RpcConfigMap["mvc-fb2"]
	conf.RpcConfigMap["mvc-fb2"] = conf.RpcConfig{
		Url:         "http://primary.invalid:9882",
		FallbackUrl: "http://fallback.invalid:9882",
	}
	defer func() { conf.RpcConfigMap["mvc-fb2"] = origCfg }()

	withShortBackoff(t, func() {
		origPrimary := broadcastOnce
		broadcastOnce = func(chain, txHex string) (string, error) {
			return "", unreachableErr()
		}
		origFallback := broadcastFallback
		broadcastFallback = func(u, user, pass, txHex string) (string, error) {
			return "", classifyBroadcastError(&url.Error{Op: "Post", URL: u, Err: io.EOF})
		}
		defer func() {
			broadcastOnce = origPrimary
			broadcastFallback = origFallback
		}()

		_, err := BroadcastTxResilient("mvc-fb2", "txhex")
		if !errors.Is(err, ErrUpstreamNodeUnreachable) {
			t.Fatalf("expected ErrUpstreamNodeUnreachable when both fail, got %v", err)
		}
	})
}
