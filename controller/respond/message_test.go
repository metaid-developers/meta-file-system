package respond

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"meta-file-system/node"
)

func init() { gin.SetMode(gin.TestMode) }

func newCtx() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/", nil)
	return c, w
}

// decode reads the Message written into the test recorder.
func decode(t *testing.T, w *httptest.ResponseRecorder) Message {
	t.Helper()
	var m Message
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode response: %v (body=%s)", err, w.Body.String())
	}
	return m
}

func TestSuccess_IncludesRequestId(t *testing.T) {
	c, w := newCtx()
	RequestIDMiddleware()(c)
	Success(c, gin.H{"ok": true})

	m := decode(t, w)
	if m.Code != CodeSuccess {
		t.Errorf("code = %d, want %d", m.Code, CodeSuccess)
	}
	if m.RequestId == "" {
		t.Error("expected requestId to be populated on success")
	}
}

func TestRequestID_HonorsIncomingHeader(t *testing.T) {
	c, w := newCtx()
	c.Request.Header.Set(HeaderNameRequestID, "client-supplied-123")
	RequestIDMiddleware()(c)

	Success(c, nil)
	m := decode(t, w)
	if m.RequestId != "client-supplied-123" {
		t.Errorf("requestId = %q, want echoed client-supplied-123", m.RequestId)
	}
	// Header must be echoed back too.
	if got := w.Header().Get(HeaderNameRequestID); got != "client-supplied-123" {
		t.Errorf("response header = %q, want client-supplied-123", got)
	}
}

func TestBroadcastError_Unreachable(t *testing.T) {
	c, w := newCtx()
	RequestIDMiddleware()(c)
	err := errors.Join(node.ErrUpstreamNodeUnreachable, errors.New("dial tcp 172.31.168.215:9882: connection refused"))

	BroadcastError(c, err)

	m := decode(t, w)
	if m.Code != CodeUpstreamNodeUnreachable {
		t.Errorf("code = %d, want %d", m.Code, CodeUpstreamNodeUnreachable)
	}
	if m.ErrorCode != ErrorCodeUpstreamNodeUnreachable {
		t.Errorf("errorCode = %q, want %q", m.ErrorCode, ErrorCodeUpstreamNodeUnreachable)
	}
	if m.RequestId == "" {
		t.Error("expected requestId on broadcast error")
	}
}

func TestBroadcastError_Timeout(t *testing.T) {
	c, w := newCtx()
	RequestIDMiddleware()(c)
	err := errors.Join(node.ErrBroadcastTimeout, errors.New("context deadline exceeded"))

	BroadcastError(c, err)

	m := decode(t, w)
	if m.Code != CodeBroadcastTimeout {
		t.Errorf("code = %d, want %d", m.Code, CodeBroadcastTimeout)
	}
	if m.ErrorCode != ErrorCodeBroadcastTimeout {
		t.Errorf("errorCode = %q, want %q", m.ErrorCode, ErrorCodeBroadcastTimeout)
	}
}

func TestBroadcastError_GenericFallback(t *testing.T) {
	c, w := newCtx()
	RequestIDMiddleware()(c)

	BroadcastError(c, errors.New("some unrelated failure"))

	m := decode(t, w)
	if m.Code != CodeServerError {
		t.Errorf("code = %d, want %d (generic fallback)", m.Code, CodeServerError)
	}
	if m.ErrorCode != "" {
		t.Errorf("errorCode = %q, want empty for generic errors", m.ErrorCode)
	}
}
