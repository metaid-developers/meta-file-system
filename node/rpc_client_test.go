package node

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"syscall"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	t.Log(BasicAuth("showpay", "showpay88.."))
}

// TestClassifyBroadcastError_Unreachable covers the failure mode seen in the
// production outage: a dial against a decommissioned node returns an
// i/o timeout wrapped in *url.Error, which must surface as
// ErrUpstreamNodeUnreachable so handlers can map it to a structured code.
func TestClassifyBroadcastError_Unreachable(t *testing.T) {
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "dial i/o timeout via url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://172.31.168.215:9882",
				Err: &net.OpError{Op: "dial", Net: "tcp", Err: &dialTimeoutStub{}},
			},
		},
		{
			name: "connection refused via url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://172.31.168.215:9882",
				Err: &net.OpError{Op: "dial", Net: "tcp", Err: syscall.ECONNREFUSED},
			},
		},
		{
			name: "no such host via url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://dead.node:9882",
				Err: &net.DNSError{Err: "no such host", Name: "dead.node", IsNotFound: true},
			},
		},
		{
			name: "unexpected eof via url.Error",
			err: &url.Error{
				Op:  "Post",
				URL: "http://172.31.168.215:9882",
				Err: io.ErrUnexpectedEOF,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyBroadcastError(tc.err)
			if !errors.Is(got, ErrUpstreamNodeUnreachable) {
				t.Fatalf("want errors.Is(_, ErrUpstreamNodeUnreachable), got %v", got)
			}
		})
	}
}

// TestClassifyBroadcastError_Timeout covers the overall HTTP client timeout
// (node accepted the connection but never answered) → ErrBroadcastTimeout.
func TestClassifyBroadcastError_Timeout(t *testing.T) {
	// http.Client.Timeout surfaces as *url.Error whose Timeout() == true.
	err := &url.Error{Op: "Post", URL: "http://node:9882", Err: context.DeadlineExceeded}
	got := classifyBroadcastError(err)
	if !errors.Is(got, ErrBroadcastTimeout) {
		t.Fatalf("want errors.Is(_, ErrBroadcastTimeout), got %v", got)
	}
}

// TestClassifyBroadcastError_RPCErrorUnchanged ensures server-side RPC
// errors (e.g. "txn-mempool-conflict") are NOT reclassified, so the
// duplicate-tx detection below still works on the original message.
func TestClassifyBroadcastError_RPCErrorUnchanged(t *testing.T) {
	rpcErr := errors.New("[-26]txn-mempool-conflict")
	got := classifyBroadcastError(rpcErr)
	if got != rpcErr {
		t.Fatalf("RPC error should pass through unchanged, got %v", got)
	}
}

func TestIsDuplicateBroadcastError(t *testing.T) {
	cases := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{errors.New("[-27]txn-already-known"), true},
		{errors.New("Transaction already in chain"), true},
		{errors.New("[-1]inputs already spent"), true},
		{errors.New("[-26]txn-mempool-conflict"), false},
	}
	for _, tc := range cases {
		if got := IsDuplicateBroadcastError(tc.err); got != tc.want {
			t.Errorf("IsDuplicateBroadcastError(%v) = %v, want %v", tc.err, got, tc.want)
		}
	}
}

// dialTimeoutStub stands in for the "i/o timeout" produced by a net.Dialer
// dial phase; it satisfies the interface net.OpError.Err uses to report a
// timeout (Timeout() bool) — same shape as the real os/syscall timeout.
type dialTimeoutStub struct{}

func (dialTimeoutStub) Error() string   { return "i/o timeout" }
func (dialTimeoutStub) Timeout() bool   { return true }
func (dialTimeoutStub) Temporary() bool { return true }
