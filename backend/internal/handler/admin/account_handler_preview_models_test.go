package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupPreviewModelsRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(newStubAdminService(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.POST("/api/v1/admin/accounts/models/preview", handler.PreviewAvailableModels)
	return router
}

func TestAccountHandlerPreviewAvailableModels_OpenAI(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		require.Equal(t, "Bearer sk-openai", r.Header.Get("Authorization"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"gpt-5.4","display_name":"GPT-5.4","owned_by":"OpenAI"}]}`))
	}))
	defer upstream.Close()

	router := setupPreviewModelsRouter()
	body := `{"platform":"openai","credentials":{"base_url":"` + upstream.URL + `","api_key":"sk-openai"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []PreviewAccountModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "gpt-5.4", resp.Data[0].ID)
	require.Equal(t, "OpenAI", resp.Data[0].Provider)
	require.Equal(t, "remote", resp.Data[0].Source)
}

func TestAccountHandlerPreviewAvailableModels_Anthropic(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/models", r.URL.Path)
		require.Equal(t, "sk-ant-test", r.Header.Get("x-api-key"))
		require.Equal(t, "2023-06-01", r.Header.Get("anthropic-version"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"claude-sonnet-4-6","display_name":"Claude Sonnet 4.6"}]}`))
	}))
	defer upstream.Close()

	router := setupPreviewModelsRouter()
	body := `{"platform":"anthropic","credentials":{"base_url":"` + upstream.URL + `","api_key":"sk-ant-test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []PreviewAccountModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "claude-sonnet-4-6", resp.Data[0].ID)
	require.Equal(t, "Anthropic", resp.Data[0].Provider)
}

func TestAccountHandlerPreviewAvailableModels_Gemini(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1beta/models", r.URL.Path)
		require.Equal(t, "AIza-test", r.Header.Get("x-goog-api-key"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"models":[{"name":"models/gemini-2.5-pro","displayName":"Gemini 2.5 Pro"}]}`))
	}))
	defer upstream.Close()

	router := setupPreviewModelsRouter()
	body := `{"platform":"gemini","credentials":{"base_url":"` + upstream.URL + `","api_key":"AIza-test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []PreviewAccountModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data, 1)
	require.Equal(t, "gemini-2.5-pro", resp.Data[0].ID)
	require.Equal(t, "Gemini", resp.Data[0].Provider)
}

func TestAccountHandlerPreviewAvailableModels_AntigravityUsesStaticCatalog(t *testing.T) {
	router := setupPreviewModelsRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/preview", strings.NewReader(`{"platform":"antigravity"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Data []PreviewAccountModel `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.NotEmpty(t, resp.Data)
	require.Contains(t, []string{"Anthropic", "Gemini", "Antigravity"}, resp.Data[0].Provider)
	require.Equal(t, "static", resp.Data[0].Source)
}

func TestAccountHandlerPreviewAvailableModels_UpstreamFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized sk-openai", http.StatusUnauthorized)
	}))
	defer upstream.Close()

	router := setupPreviewModelsRouter()
	body := `{"platform":"openai","credentials":{"base_url":"` + upstream.URL + `","api_key":"sk-openai"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/models/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.NotContains(t, rec.Body.String(), "sk-openai")
}
