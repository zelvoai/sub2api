package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Images2Handler struct {
	service       *service.Images2Service
	openaiGateway *OpenAIGatewayHandler
}

func NewImages2Handler(images2Service *service.Images2Service, openaiGateway *OpenAIGatewayHandler) *Images2Handler {
	return &Images2Handler{service: images2Service, openaiGateway: openaiGateway}
}

func (h *Images2Handler) Generate(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}

	var req service.Images2GenerateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}

	prepared, err := h.service.Prepare(c.Request.Context(), subject.UserID, req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}

	body, err := json.Marshal(map[string]any{
		"model":           prepared.ModelName,
		"prompt":          prepared.Prompt,
		"response_format": "b64_json",
		"size":            prepared.Size,
	})
	if prepared.ImageURL != "" {
		body, err = json.Marshal(map[string]any{
			"model":           prepared.ModelName,
			"prompt":          prepared.Prompt,
			"response_format": "b64_json",
			"size":            prepared.Size,
			"images": []map[string]string{{
				"image_url": prepared.ImageURL,
			}},
		})
	}
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to build image request")
		return
	}

	proxyCtx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	endpoint := "/v1/images/generations"
	if prepared.ImageURL != "" {
		endpoint = "/v1/images/edits"
	}
	proxyReq, err := http.NewRequestWithContext(proxyCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to create image request")
		return
	}
	proxyReq.Header.Set("Content-Type", "application/json")
	proxyReq.Header.Set("User-Agent", c.GetHeader("User-Agent"))

	proxyCtx = context.WithValue(proxyReq.Context(), middleware2.ContextKeyAPIKey, prepared.APIKey)
	proxyReq = proxyReq.WithContext(proxyCtx)
	c.Set(string(middleware2.ContextKeyAPIKey), prepared.APIKey)
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: prepared.User.ID, Concurrency: prepared.User.Concurrency})
	c.Set(string(middleware2.ContextKeyUserRole), prepared.User.Role)

	proxyRecorder := newBufferedResponseRecorder()
	proxyContext, _ := gin.CreateTestContext(proxyRecorder)
	proxyContext.Request = proxyReq
	proxyContext.Set(string(middleware2.ContextKeyAPIKey), prepared.APIKey)
	proxyContext.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: prepared.User.ID, Concurrency: prepared.User.Concurrency})
	proxyContext.Set(string(middleware2.ContextKeyUserRole), prepared.User.Role)
	if prepared.Subscription != nil {
		proxyContext.Set(string(middleware2.ContextKeySubscription), prepared.Subscription)
	}

	h.openaiGateway.Images(proxyContext)

	if proxyRecorder.statusCode >= 400 {
		logger.L().Error("images2.direct_forward_failed",
			zap.Int64("user_id", prepared.User.ID),
			zap.Int("status_code", proxyRecorder.statusCode),
			zap.ByteString("body", proxyRecorder.body.Bytes()),
		)
		c.Data(proxyRecorder.statusCode, proxyRecorder.header.Get("Content-Type"), proxyRecorder.body.Bytes())
		return
	}

	var upstream struct {
		Created int64            `json:"created"`
		Data    []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(proxyRecorder.body.Bytes(), &upstream); err != nil {
		response.Error(c, http.StatusBadGateway, "Failed to parse generated image response")
		return
	}

	response.Success(c, gin.H{
		"images":        upstream.Data,
		"applied_price": prepared.Settings.Images2PricePerImage,
		"balance":       prepared.User.Balance,
		"client_ip":     ip.GetTrustedClientIP(c),
	})
}

type bufferedResponseRecorder struct {
	header     http.Header
	body       bytes.Buffer
	statusCode int
}

func newBufferedResponseRecorder() *bufferedResponseRecorder {
	return &bufferedResponseRecorder{header: make(http.Header), statusCode: http.StatusOK}
}

func (r *bufferedResponseRecorder) Header() http.Header            { return r.header }
func (r *bufferedResponseRecorder) Write(data []byte) (int, error) { return r.body.Write(data) }
func (r *bufferedResponseRecorder) WriteHeader(statusCode int)     { r.statusCode = statusCode }
