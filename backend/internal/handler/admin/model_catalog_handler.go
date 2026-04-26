package admin

import (
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type ModelVendorHandler struct {
	service *service.ModelVendorService
}

func NewModelVendorHandler(service *service.ModelVendorService) *ModelVendorHandler {
	return &ModelVendorHandler{service: service}
}

func (h *ModelVendorHandler) List(c *gin.Context) {
	params := parseAdminPagination(c)
	items, pr, err := h.service.List(c.Request.Context(), params, c.Query("search"), c.Query("status"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.PaginatedWithResult(c, items, modelCatalogResponsePagination(pr))
}

func (h *ModelVendorHandler) Create(c *gin.Context) {
	var req service.ModelVendor
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if response.ErrorFrom(c, h.service.Create(c.Request.Context(), &req)) {
		return
	}
	response.Created(c, req)
}

func (h *ModelVendorHandler) Update(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	var req service.ModelVendor
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.ID = id
	if response.ErrorFrom(c, h.service.Update(c.Request.Context(), &req)) {
		return
	}
	response.Success(c, req)
}

func (h *ModelVendorHandler) Delete(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	if response.ErrorFrom(c, h.service.Delete(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

type ModelCatalogHandler struct {
	service    *service.ModelCatalogService
	capability *service.AccountModelCapabilityService
}

func NewModelCatalogHandler(service *service.ModelCatalogService, capability *service.AccountModelCapabilityService) *ModelCatalogHandler {
	return &ModelCatalogHandler{service: service, capability: capability}
}

func (h *ModelCatalogHandler) List(c *gin.Context) {
	params := parseAdminPagination(c)
	filters := service.ModelCatalogFilters{
		Search: c.Query("search"),
		Status: c.Query("status"),
	}
	if vendorID := c.Query("vendor_id"); vendorID != "" {
		if id, err := strconv.ParseInt(vendorID, 10, 64); err == nil && id > 0 {
			filters.VendorID = &id
		}
	}
	if rule := c.Query("name_rule"); rule != "" {
		if v, err := strconv.Atoi(rule); err == nil {
			filters.NameRule = &v
		}
	}
	items, pr, err := h.service.List(c.Request.Context(), params, filters)
	if response.ErrorFrom(c, err) {
		return
	}
	response.PaginatedWithResult(c, items, modelCatalogResponsePagination(pr))
}

func (h *ModelCatalogHandler) Search(c *gin.Context) {
	h.List(c)
}

func (h *ModelCatalogHandler) Get(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	model, err := h.service.Get(c.Request.Context(), id)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, model)
}

func (h *ModelCatalogHandler) Create(c *gin.Context) {
	var req service.ModelCatalog
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if response.ErrorFrom(c, h.service.Create(c.Request.Context(), &req)) {
		return
	}
	response.Created(c, req)
}

func (h *ModelCatalogHandler) Update(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	var req service.ModelCatalog
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	req.ID = id
	if response.ErrorFrom(c, h.service.Update(c.Request.Context(), &req)) {
		return
	}
	response.Success(c, req)
}

func (h *ModelCatalogHandler) UpdateStatus(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	var req struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if response.ErrorFrom(c, h.service.UpdateStatus(c.Request.Context(), id, req.Status)) {
		return
	}
	response.Success(c, gin.H{"status": req.Status})
}

func (h *ModelCatalogHandler) Delete(c *gin.Context) {
	id, ok := parseModelCatalogIDParam(c, "id")
	if !ok {
		return
	}
	if response.ErrorFrom(c, h.service.Delete(c.Request.Context(), id)) {
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

func (h *ModelCatalogHandler) BatchDelete(c *gin.Context) {
	var req struct {
		IDs []int64 `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if response.ErrorFrom(c, h.service.BatchDelete(c.Request.Context(), req.IDs)) {
		return
	}
	response.Success(c, gin.H{"deleted": len(req.IDs)})
}

func (h *ModelCatalogHandler) Missing(c *gin.Context) {
	items, err := h.service.Missing(c.Request.Context())
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func (h *ModelCatalogHandler) GroupAvailable(c *gin.Context) {
	groupIDs := parseInt64CSV(c.Query("group_ids"))
	if len(groupIDs) == 0 {
		response.Success(c, []service.GroupAvailableModel{})
		return
	}
	limit := 100
	if raw := c.Query("limit"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			limit = v
		}
	}
	items, err := h.capability.ListGroupModels(c.Request.Context(), groupIDs, c.Query("search"), limit)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, items)
}

func (h *ModelCatalogHandler) SyncPreview(c *gin.Context) {
	preview, err := h.service.SyncPreview(c.Request.Context(), c.Query("locale"))
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, preview)
}

func (h *ModelCatalogHandler) SyncUpstream(c *gin.Context) {
	var req service.ModelSyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.service.SyncUpstream(c.Request.Context(), req)
	if response.ErrorFrom(c, err) {
		return
	}
	response.Success(c, result)
}

func parseAdminPagination(c *gin.Context) pagination.PaginationParams {
	page, pageSize := response.ParsePagination(c)
	return pagination.PaginationParams{
		Page:      page,
		PageSize:  pageSize,
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}
}

func parseModelCatalogIDParam(c *gin.Context, name string) (int64, bool) {
	id, err := strconv.ParseInt(c.Param(name), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "invalid id")
		return 0, false
	}
	return id, true
}

func parseInt64CSV(raw string) []int64 {
	if raw == "" {
		return nil
	}
	var out []int64
	for _, part := range strings.Split(raw, ",") {
		id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64)
		if err == nil && id > 0 {
			out = append(out, id)
		}
	}
	return out
}

func modelCatalogResponsePagination(pr *pagination.PaginationResult) *response.PaginationResult {
	if pr == nil {
		return nil
	}
	return &response.PaginationResult{
		Total:    pr.Total,
		Page:     pr.Page,
		PageSize: pr.PageSize,
		Pages:    pr.Pages,
	}
}
