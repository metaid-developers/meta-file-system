package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestGetFilesByKeywordAndExtensionValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("blank keyword", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "keyword", Value: "   "}}
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/files/keyword/%20%20%20/extension?extension=.mp3", nil)

		handler := &IndexerQueryHandler{}
		handler.GetFilesByKeywordAndExtension(c)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if code, ok := resp["code"].(float64); !ok || int(code) != 40000 {
			t.Fatalf("code = %v, want 40000", resp["code"])
		}
	})

	t.Run("missing extension", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Params = gin.Params{{Key: "keyword", Value: "周杰伦"}}
		c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/files/keyword/%E5%91%A8%E6%9D%B0%E4%BC%A6/extension", nil)

		handler := &IndexerQueryHandler{}
		handler.GetFilesByKeywordAndExtension(c)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}

		var resp map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Fatalf("json.Unmarshal() error = %v", err)
		}
		if code, ok := resp["code"].(float64); !ok || int(code) != 40000 {
			t.Fatalf("code = %v, want 40000", resp["code"])
		}
	})
}
