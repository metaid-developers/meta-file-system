package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"meta-file-system/conf"

	"github.com/gin-gonic/gin"
)

func TestSetupIndexerRouterDoesNotRegisterAdminRescanRoutesByDefault(t *testing.T) {
	oldCfg := conf.Cfg
	defer func() { conf.Cfg = oldCfg }()

	gin.SetMode(gin.TestMode)
	conf.Cfg = &conf.Config{
		Indexer: conf.IndexerConfig{
			SwaggerBaseUrl: "localhost:7281",
		},
	}

	router := SetupIndexerRouter(nil, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/rescan/status", nil)

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("GET /api/v1/admin/rescan/status status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestSetupIndexerRouterRegistersAdminRescanRoutesWhenEnabled(t *testing.T) {
	oldCfg := conf.Cfg
	defer func() { conf.Cfg = oldCfg }()

	gin.SetMode(gin.TestMode)
	conf.Cfg = &conf.Config{
		Indexer: conf.IndexerConfig{
			AdminEnabled:   true,
			SwaggerBaseUrl: "localhost:7281",
		},
	}

	router := SetupIndexerRouter(nil, nil)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/rescan/status", nil)

	router.ServeHTTP(w, req)

	if w.Code == http.StatusNotFound {
		t.Fatalf("GET /api/v1/admin/rescan/status status = %d, want route registered", w.Code)
	}
}
