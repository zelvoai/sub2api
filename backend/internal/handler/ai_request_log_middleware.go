package handler

import (
	"bytes"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

const aiRequestLogResponseLimitBytes = 4 * 1024 * 1024

type aiRequestLogCaptureWriter struct {
	gin.ResponseWriter
	buf   bytes.Buffer
	limit int
}

func (w *aiRequestLogCaptureWriter) Write(p []byte) (int, error) {
	if w.limit > 0 && w.buf.Len() < w.limit {
		remain := w.limit - w.buf.Len()
		if len(p) > remain {
			_, _ = w.buf.Write(p[:remain])
		} else {
			_, _ = w.buf.Write(p)
		}
	}
	return w.ResponseWriter.Write(p)
}

func (w *aiRequestLogCaptureWriter) WriteString(s string) (int, error) {
	if w.limit > 0 && w.buf.Len() < w.limit {
		remain := w.limit - w.buf.Len()
		if len(s) > remain {
			_, _ = w.buf.WriteString(s[:remain])
		} else {
			_, _ = w.buf.WriteString(s)
		}
	}
	return w.ResponseWriter.WriteString(s)
}

func AIRequestLogMiddleware(logService *service.AIRequestLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		originalWriter := c.Writer
		captureWriter := &aiRequestLogCaptureWriter{ResponseWriter: originalWriter, limit: aiRequestLogResponseLimitBytes}
		c.Writer = captureWriter
		c.Next()
		c.Writer = originalWriter

		if logService == nil {
			return
		}
		requestBody := ""
		if v, ok := c.Get(opsRequestBodyKey); ok {
			switch raw := v.(type) {
			case string:
				requestBody = raw
			case []byte:
				requestBody = string(raw)
			}
		}
		if requestBody == "" && c.Request != nil && c.Request.Body == nil {
			return
		}
		model := ""
		if v, ok := c.Get(opsModelKey); ok {
			if s, ok := v.(string); ok {
				model = s
			}
		}
		stream := false
		if v, ok := c.Get(opsStreamKey); ok {
			if b, ok := v.(bool); ok {
				stream = b
			}
		}
		var userID *int64
		var apiKeyID *int64
		var accountID *int64
		var groupID *int64
		if subject, ok := middleware.GetAuthSubjectFromContext(c); ok {
			v := subject.UserID
			userID = &v
		}
		if apiKey, ok := middleware.GetAPIKeyFromContext(c); ok && apiKey != nil {
			v := apiKey.ID
			apiKeyID = &v
			if apiKey.GroupID != nil {
				g := *apiKey.GroupID
				groupID = &g
			}
		}
		if v, ok := c.Get(opsAccountIDKey); ok {
			if id, ok := v.(int64); ok && id > 0 {
				accountID = &id
			}
		}
		platform := guessPlatformFromPath(c.Request.URL.Path)
		if apiKey, ok := middleware.GetAPIKeyFromContext(c); ok {
			platform = resolveOpsPlatform(apiKey, platform)
		}
		requestID := strings.TrimSpace(c.Writer.Header().Get("X-Request-Id"))
		if requestID == "" {
			requestID = strings.TrimSpace(c.Writer.Header().Get("x-request-id"))
		}
		clientRequestID, _ := c.Request.Context().Value(ctxkey.ClientRequestID).(string)
		inboundEndpoint := strings.TrimSpace(GetInboundEndpoint(c))
		upstreamEndpoint := ""
		if apiKey, ok := middleware.GetAPIKeyFromContext(c); ok && apiKey != nil && apiKey.Group != nil {
			upstreamEndpoint = strings.TrimSpace(GetUpstreamEndpoint(c, apiKey.Group.Platform))
		}
		statusCode := c.Writer.Status()
		contentType := ""
		if c.Request != nil {
			contentType = strings.TrimSpace(c.Request.Header.Get("Content-Type"))
		}
		responseContentType := strings.TrimSpace(c.Writer.Header().Get("Content-Type"))
		responseBody := captureWriter.buf.String()
		errorMessage := ""
		if statusCode >= 400 {
			errorMessage = responseBody
		}
		duration := int(time.Since(startedAt).Milliseconds())
		entry := &service.AIRequestLog{
			CreatedAt:           startedAt.UTC(),
			RequestID:           requestID,
			ClientRequestID:     strings.TrimSpace(clientRequestID),
			UserID:              userID,
			APIKeyID:            apiKeyID,
			AccountID:           accountID,
			GroupID:             groupID,
			Platform:            platform,
			Model:               strings.TrimSpace(model),
			RequestPath:         strings.TrimSpace(c.Request.URL.Path),
			InboundEndpoint:     inboundEndpoint,
			UpstreamEndpoint:    upstreamEndpoint,
			Method:              c.Request.Method,
			StatusCode:          statusCode,
			Stream:              stream,
			RequestBody:         requestBody,
			ResponseBody:        responseBody,
			ErrorMessage:        errorMessage,
			DurationMs:          &duration,
			ContentType:         contentType,
			ResponseContentType: responseContentType,
		}
		_ = logService.Record(c.Request.Context(), entry)
	}
}
