package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"meta-file-system/model"
)

func init() { gin.SetMode(gin.TestMode) }

// TestWriteHeadHeaders_SetsContentHeadersNoBody is the core RFC-7231 unit:
// HEAD must return the same Content-* headers as GET but no body.
func TestWriteHeadHeaders_SetsContentHeadersNoBody(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodHead, "/", nil)

	writeHeadHeaders(c, &model.IndexerFile{
		ContentType: "image/png",
		FileName:    "bigavatar.png",
		FileSize:    5002337,
	})
	c.Status(200)

	if got := w.Code; got != 200 {
		t.Errorf("status = %d, want 200", got)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/png" {
		t.Errorf("Content-Type = %q, want image/png", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd != `inline; filename="bigavatar.png"` {
		t.Errorf("Content-Disposition = %q, want inline filename", cd)
	}
	if cl := w.Header().Get("Content-Length"); cl != "5002337" {
		t.Errorf("Content-Length = %q, want 5002337", cl)
	}
	// The handler must not write a body for HEAD.
	if w.Body.Len() != 0 {
		t.Errorf("HEAD wrote %d body bytes, want 0", w.Body.Len())
	}
}

// TestWriteHeadHeaders_EmptyFileOmitsHeaders verifies we don't emit empty /
// misleading headers when metadata is sparse.
func TestWriteHeadHeaders_EmptyFileOmitsHeaders(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodHead, "/", nil)

	writeHeadHeaders(c, &model.IndexerFile{})
	c.Status(200)

	for _, h := range []string{"Content-Type", "Content-Disposition", "Content-Length"} {
		if v := w.Header().Get(h); v != "" {
			t.Errorf("expected no %s for empty metadata, got %q", h, v)
		}
	}
}

// TestHeadHandlers_MissingParamBailsBeforeServiceLookup guards the empty-param
// branch, which returns 40000 before the (nil in this test) service is touched.
func TestHeadHandlers_MissingParamBailsBeforeServiceLookup(t *testing.T) {
	cases := []struct {
		name string
		fn   func(*gin.Context)
	}{
		{"HeadFileContent", (&IndexerQueryHandler{}).HeadFileContent},
		{"HeadFastFileContent", (&IndexerQueryHandler{}).HeadFastFileContent},
		{"HeadLatestFileContentByFirstPinID", (&IndexerQueryHandler{}).HeadLatestFileContentByFirstPinID},
		{"HeadLatestFastFileContentByFirstPinID", (&IndexerQueryHandler{}).HeadLatestFastFileContentByFirstPinID},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Request = httptest.NewRequest(http.MethodHead, "/", nil)
			// No params set -> pinId/firstPinId == "" -> InvalidParam (HTTP 200, code 40000).
			tc.fn(c)
			if w.Code != 200 {
				t.Fatalf("status = %d, want 200", w.Code)
			}
		})
	}
}
