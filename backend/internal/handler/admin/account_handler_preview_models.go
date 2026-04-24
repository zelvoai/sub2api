package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/antigravity"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/geminicli"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type PreviewAccountModelsRequest struct {
	Platform    string         `json:"platform" binding:"required"`
	Type        string         `json:"type"`
	Credentials map[string]any `json:"credentials"`
}

type PreviewAccountModel struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Provider    string `json:"provider"`
	Source      string `json:"source"`
}

const previewModelsTimeout = 15 * time.Second

// PreviewAvailableModels fetches models for unsaved account credentials.
// POST /api/v1/admin/accounts/models/preview
func (h *AccountHandler) PreviewAvailableModels(c *gin.Context) {
	var req PreviewAccountModelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	platform := strings.TrimSpace(req.Platform)
	switch platform {
	case service.PlatformOpenAI, service.PlatformAnthropic, service.PlatformGemini, service.PlatformAntigravity:
	default:
		response.BadRequest(c, "Unsupported platform")
		return
	}

	if platform == service.PlatformAntigravity {
		response.Success(c, previewModelsFromAntigravityDefaults())
		return
	}

	apiKey := strings.TrimSpace(stringCredential(req.Credentials, "api_key"))
	if apiKey == "" {
		response.BadRequest(c, "API key is required")
		return
	}

	baseURL := previewModelsBaseURL(platform, stringCredential(req.Credentials, "base_url"))
	if baseURL == "" {
		response.BadRequest(c, "Base URL is required")
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), previewModelsTimeout)
	defer cancel()

	models, err := fetchPreviewModels(ctx, platform, baseURL, apiKey)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	response.Success(c, models)
}

func fetchPreviewModels(ctx context.Context, platform, baseURL, apiKey string) ([]PreviewAccountModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, previewModelsURL(platform, baseURL), nil)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadRequest, "INVALID_BASE_URL", "Failed to build preview request")
	}

	switch platform {
	case service.PlatformOpenAI:
		req.Header.Set("Authorization", "Bearer "+apiKey)
	case service.PlatformAnthropic:
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	case service.PlatformGemini:
		req.Header.Set("x-goog-api-key", apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "PREVIEW_MODELS_UPSTREAM_FAILED", "Failed to fetch models from upstream")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, infraerrors.New(resp.StatusCode, "PREVIEW_MODELS_UPSTREAM_FAILED", fmt.Sprintf("Upstream model listing failed with status %d", resp.StatusCode))
	}

	switch platform {
	case service.PlatformOpenAI:
		return parseOpenAIModelPreview(resp)
	case service.PlatformAnthropic:
		return parseAnthropicModelPreview(resp)
	case service.PlatformGemini:
		return parseGeminiModelPreview(resp)
	default:
		return nil, infraerrors.New(http.StatusBadRequest, "UNSUPPORTED_PLATFORM", "Unsupported platform")
	}
}

func parseOpenAIModelPreview(resp *http.Response) ([]PreviewAccountModel, error) {
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
			OwnedBy     string `json:"owned_by"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "PREVIEW_MODELS_INVALID_RESPONSE", "Invalid upstream model payload")
	}
	models := make([]PreviewAccountModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		provider := strings.TrimSpace(item.OwnedBy)
		if provider == "" {
			provider = inferPreviewModelProvider(modelID, service.PlatformOpenAI)
		}
		models = append(models, PreviewAccountModel{ID: modelID, DisplayName: displayName, Provider: provider, Source: "remote"})
	}
	return models, nil
}

func parseAnthropicModelPreview(resp *http.Response) ([]PreviewAccountModel, error) {
	var payload struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "PREVIEW_MODELS_INVALID_RESPONSE", "Invalid upstream model payload")
	}
	models := make([]PreviewAccountModel, 0, len(payload.Data))
	for _, item := range payload.Data {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		models = append(models, PreviewAccountModel{ID: modelID, DisplayName: displayName, Provider: inferPreviewModelProvider(modelID, service.PlatformAnthropic), Source: "remote"})
	}
	return models, nil
}

func parseGeminiModelPreview(resp *http.Response) ([]PreviewAccountModel, error) {
	var payload struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
		} `json:"models"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, infraerrors.New(http.StatusBadGateway, "PREVIEW_MODELS_INVALID_RESPONSE", "Invalid upstream model payload")
	}
	models := make([]PreviewAccountModel, 0, len(payload.Models))
	for _, item := range payload.Models {
		modelID := strings.TrimSpace(item.Name)
		if modelID == "" {
			continue
		}
		modelID = strings.TrimPrefix(modelID, "models/")
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		models = append(models, PreviewAccountModel{ID: modelID, DisplayName: displayName, Provider: inferPreviewModelProvider(modelID, service.PlatformGemini), Source: "remote"})
	}
	return models, nil
}

func previewModelsFromAntigravityDefaults() []PreviewAccountModel {
	defaults := antigravity.DefaultModels()
	models := make([]PreviewAccountModel, 0, len(defaults))
	for _, item := range defaults {
		modelID := strings.TrimSpace(item.ID)
		if modelID == "" {
			continue
		}
		displayName := strings.TrimSpace(item.DisplayName)
		if displayName == "" {
			displayName = modelID
		}
		models = append(models, PreviewAccountModel{ID: modelID, DisplayName: displayName, Provider: inferPreviewModelProvider(modelID, service.PlatformAntigravity), Source: "static"})
	}
	return models
}

func previewModelsBaseURL(platform, raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed != "" {
		return strings.TrimRight(trimmed, "/")
	}
	switch platform {
	case service.PlatformOpenAI:
		return "https://api.openai.com"
	case service.PlatformAnthropic:
		return "https://api.anthropic.com"
	case service.PlatformGemini:
		return geminicli.AIStudioBaseURL
	default:
		return ""
	}
}

func previewModelsURL(platform, baseURL string) string {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	switch platform {
	case service.PlatformOpenAI, service.PlatformAnthropic:
		if strings.HasSuffix(trimmed, "/v1") {
			return trimmed + "/models"
		}
		return trimmed + "/v1/models"
	case service.PlatformGemini:
		if strings.HasSuffix(trimmed, "/v1beta") {
			return trimmed + "/models"
		}
		return trimmed + "/v1beta/models"
	default:
		return trimmed
	}
}

func inferPreviewModelProvider(modelID, platform string) string {
	lowered := strings.ToLower(strings.TrimSpace(modelID))
	switch {
	case strings.HasPrefix(lowered, "gpt"), strings.HasPrefix(lowered, "o1"), strings.HasPrefix(lowered, "o3"), strings.HasPrefix(lowered, "o4"), strings.HasPrefix(lowered, "chatgpt"):
		return "OpenAI"
	case strings.HasPrefix(lowered, "claude"):
		return "Anthropic"
	case strings.HasPrefix(lowered, "gemini"):
		return "Gemini"
	}
	switch platform {
	case service.PlatformOpenAI:
		return "OpenAI"
	case service.PlatformAnthropic:
		return "Anthropic"
	case service.PlatformGemini:
		return "Gemini"
	case service.PlatformAntigravity:
		return "Antigravity"
	default:
		return "Other"
	}
}

func stringCredential(credentials map[string]any, key string) string {
	if credentials == nil {
		return ""
	}
	raw, ok := credentials[key]
	if !ok || raw == nil {
		return ""
	}
	switch value := raw.(type) {
	case string:
		return value
	default:
		return strings.TrimSpace(fmt.Sprint(value))
	}
}
