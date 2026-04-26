package admin

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type AccountUpstreamModelHandler struct {
	service *service.AccountUpstreamModelService
}

func NewAccountUpstreamModelHandler(service *service.AccountUpstreamModelService) *AccountUpstreamModelHandler {
	return &AccountUpstreamModelHandler{service: service}
}

func (h *AccountUpstreamModelHandler) Preview(c *gin.Context) {
	var req service.AccountUpstreamModelPreviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Preview(c.Request.Context(), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountUpstreamModelHandler) Detect(c *gin.Context) {
	id, ok := parseAccountUpstreamModelID(c)
	if !ok {
		return
	}
	result, err := h.service.Detect(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountUpstreamModelHandler) Apply(c *gin.Context) {
	id, ok := parseAccountUpstreamModelID(c)
	if !ok {
		return
	}
	var req service.AccountUpstreamModelApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.Apply(c.Request.Context(), id, req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func (h *AccountUpstreamModelHandler) ImportCatalog(c *gin.Context) {
	var req service.AccountUpstreamModelImportCatalogRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.ImportCatalog(c.Request.Context(), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func parseAccountUpstreamModelID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid account id")
		return 0, false
	}
	return id, true
}
