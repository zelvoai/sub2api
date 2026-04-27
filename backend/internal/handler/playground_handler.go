package handler

import (
	"bytes"
	"errors"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
)

type PlaygroundHandler struct {
	playgroundService  *service.PlaygroundService
	openaiGateway      *OpenAIGatewayHandler
	billingCache       *service.BillingCacheService
}

func NewPlaygroundHandler(
	playgroundService *service.PlaygroundService,
	openaiGateway *OpenAIGatewayHandler,
	billingCache *service.BillingCacheService,
) *PlaygroundHandler {
	return &PlaygroundHandler{
		playgroundService: playgroundService,
		openaiGateway:     openaiGateway,
		billingCache:      billingCache,
	}
}

func (h *PlaygroundHandler) GetGroups(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	groups, err := h.playgroundService.ListGroups(c.Request.Context(), subject.UserID)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, groups)
}

func (h *PlaygroundHandler) GetModels(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	groupID, err := parseOptionalGroupID(c.Query("group_id"))
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if v, convErr := strconv.Atoi(raw); convErr == nil && v > 0 {
			limit = v
		}
	}
	items, svcErr := h.playgroundService.ListModels(c.Request.Context(), subject.UserID, groupID, c.Query("search"), limit)
	if response.ErrorFrom(c, svcErr) {
		return
	}
	response.Success(c, items)
}

func (h *PlaygroundHandler) GetGroupModels(c *gin.Context) {
	groupID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || groupID <= 0 {
		response.BadRequest(c, "invalid group id")
		return
	}
	c.Request.URL.RawQuery = ""
	c.Request.URL.RawQuery = c.Request.URL.Query().Encode()
	q := c.Request.URL.Query()
	q.Set("group_id", strconv.FormatInt(groupID, 10))
	c.Request.URL.RawQuery = q.Encode()
	h.GetModels(c)
}

func (h *PlaygroundHandler) ChatCompletions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	body, err := readPlaygroundBody(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	groupID := gjson.GetBytes(body, "group_id").Int()
	if groupID <= 0 {
		response.ErrorFrom(c, service.ErrPlaygroundGroupRequired)
		return
	}
	execCtx, err := h.playgroundService.ResolveExecutionContext(c.Request.Context(), subject.UserID, groupID)
	if response.ErrorFrom(c, err) {
		return
	}
	if h.billingCache != nil {
		if err := h.billingCache.CheckBillingEligibility(c.Request.Context(), execCtx.User, execCtx.RuntimeAPIKey, execCtx.Group, execCtx.Subscription); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	c.Set(string(middleware2.ContextKeyAPIKey), execCtx.RuntimeAPIKey)
	if execCtx.Subscription != nil {
		c.Set(string(middleware2.ContextKeySubscription), execCtx.Subscription)
	}
	c.Set(string(middleware2.ContextKeyUserRole), execCtx.User.Role)
	middleware2.SetGroupContextForRuntime(c, execCtx.Group)
	h.openaiGateway.ChatCompletions(c)
}

func (h *PlaygroundHandler) ImagesGenerations(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "User not authenticated")
		return
	}
	body, err := readPlaygroundBody(c)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	groupID := gjson.GetBytes(body, "group_id").Int()
	if groupID <= 0 {
		response.ErrorFrom(c, service.ErrPlaygroundGroupRequired)
		return
	}
	execCtx, err := h.playgroundService.ResolveExecutionContext(c.Request.Context(), subject.UserID, groupID)
	if response.ErrorFrom(c, err) {
		return
	}
	if h.billingCache != nil {
		if err := h.billingCache.CheckBillingEligibility(c.Request.Context(), execCtx.User, execCtx.RuntimeAPIKey, execCtx.Group, execCtx.Subscription); err != nil {
			response.ErrorFrom(c, err)
			return
		}
	}
	c.Set(string(middleware2.ContextKeyAPIKey), execCtx.RuntimeAPIKey)
	if execCtx.Subscription != nil {
		c.Set(string(middleware2.ContextKeySubscription), execCtx.Subscription)
	}
	c.Set(string(middleware2.ContextKeyUserRole), execCtx.User.Role)
	middleware2.SetGroupContextForRuntime(c, execCtx.Group)
	h.openaiGateway.Images(c)
}

func parseOptionalGroupID(raw string) (*int64, error) {
	if raw == "" {
		return nil, nil
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return nil, service.ErrPlaygroundGroupInvalid
	}
	return &id, nil
}

func readPlaygroundBody(c *gin.Context) ([]byte, error) {
	body, err := c.GetRawData()
	if err != nil {
		return nil, err
	}
	if len(body) == 0 || !gjson.ValidBytes(body) {
		return nil, errors.New("invalid JSON body")
	}
	c.Request.Body = ioNopCloser(bytes.NewReader(body))
	return body, nil
}

type nopCloser struct{ *bytes.Reader }

func (nopCloser) Close() error { return nil }

func ioNopCloser(r *bytes.Reader) nopCloser { return nopCloser{Reader: r} }
