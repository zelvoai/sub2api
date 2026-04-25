package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AIRequestLogHandler struct {
	service *service.AIRequestLogService
}

func NewAIRequestLogHandler(service *service.AIRequestLogService) *AIRequestLogHandler {
	return &AIRequestLogHandler{service: service}
}

func (h *AIRequestLogHandler) List(c *gin.Context) {
	if h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "AI request log service not available")
		return
	}
	page, pageSize := response.ParsePagination(c)
	filter := &service.AIRequestLogFilter{Page: page, PageSize: pageSize}
	filter.RequestID = strings.TrimSpace(c.Query("request_id"))
	filter.ClientRequestID = strings.TrimSpace(c.Query("client_request_id"))
	filter.Platform = strings.TrimSpace(c.Query("platform"))
	filter.Model = strings.TrimSpace(c.Query("model"))
	filter.Query = strings.TrimSpace(c.Query("q"))
	if v := strings.TrimSpace(c.Query("status_code")); v != "" {
		code, err := strconv.Atoi(v)
		if err != nil {
			response.BadRequest(c, "Invalid status_code")
			return
		}
		filter.StatusCode = &code
	}
	if v := strings.TrimSpace(c.Query("user_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid user_id")
			return
		}
		filter.UserID = &id
	}
	if v := strings.TrimSpace(c.Query("api_key_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid api_key_id")
			return
		}
		filter.APIKeyID = &id
	}
	if v := strings.TrimSpace(c.Query("account_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid account_id")
			return
		}
		filter.AccountID = &id
	}
	if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			response.BadRequest(c, "Invalid group_id")
			return
		}
		filter.GroupID = &id
	}
	if v := strings.TrimSpace(c.Query("start_time")); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid start_time")
			return
		}
		filter.StartTime = &tm
	}
	if v := strings.TrimSpace(c.Query("end_time")); v != "" {
		tm, err := time.Parse(time.RFC3339, v)
		if err != nil {
			response.BadRequest(c, "Invalid end_time")
			return
		}
		filter.EndTime = &tm
	}
	out, err := h.service.List(c.Request.Context(), filter)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, out.Items, out.Total, out.Page, out.PageSize)
}

func (h *AIRequestLogHandler) GetByID(c *gin.Context) {
	if h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "AI request log service not available")
		return
	}
	id, err := strconv.ParseInt(strings.TrimSpace(c.Param("id")), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid log id")
		return
	}
	item, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *AIRequestLogHandler) GetRetentionSettings(c *gin.Context) {
	if h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "AI request log service not available")
		return
	}
	item, err := h.service.GetRetentionSettings(c.Request.Context())
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *AIRequestLogHandler) UpdateRetentionSettings(c *gin.Context) {
	if h.service == nil {
		response.Error(c, http.StatusServiceUnavailable, "AI request log service not available")
		return
	}
	var req service.AIRequestLogRetentionSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body")
		return
	}
	updated, err := h.service.UpdateRetentionSettings(c.Request.Context(), &req)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, updated)
}
