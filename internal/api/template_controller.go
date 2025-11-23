package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-gin/internal/utils"
)

// TemplateController 模板控制器
type TemplateController struct {
	templateService service.TemplateService
}

// NewTemplateController 创建模板控制器
func NewTemplateController(templateService service.TemplateService) *TemplateController {
	return &TemplateController{
		templateService: templateService,
	}
}

// Create 创建模板
// @Summary      创建审批模板
// @Description  创建新的审批模板
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        request body service.CreateTemplateRequest true "模板信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates [post]
// @Security    BearerAuth
func (c *TemplateController) Create(ctx *gin.Context) {
	var req service.CreateTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	// 输入验证和清理
	if err := utils.ValidateTemplateName(req.Name); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid template name", err.Error())
		return
	}
	// 清理模板名称
	req.Name, _ = utils.TrimAndValidate(req.Name, 255)
	if req.Description != "" {
		req.Description, _ = utils.TrimAndValidate(req.Description, 1000)
	}

	template, err := c.templateService.Create(ctx.Request.Context(), &req)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to create template", err.Error())
		return
	}

	Success(ctx, template)
}

// Get 获取模板
// @Summary      获取模板详情
// @Description  根据 ID 获取模板详情,支持版本查询
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        id path string true "模板 ID"
// @Param        version query int false "版本号,不传则获取最新版本"
// @Success      200  {object}  Response
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates/{id} [get]
// @Security     BearerAuth
func (c *TemplateController) Get(ctx *gin.Context) {
	id := ctx.Param("id")
	
	// 验证模板 ID 格式
	if err := utils.ValidateTemplateID(id); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid template id", err.Error())
		return
	}
	
	versionStr := ctx.Query("version")
	
	version := 0
	if versionStr != "" {
		var err error
		version, err = strconv.Atoi(versionStr)
		if err != nil {
			Error(ctx, http.StatusBadRequest, "invalid version", err.Error())
			return
		}
	}

	template, err := c.templateService.Get(id, version)
	if err != nil {
		Error(ctx, http.StatusNotFound, "template not found", err.Error())
		return
	}

	Success(ctx, template)
}

// Update 更新模板
// @Summary      更新审批模板
// @Description  更新现有审批模板(创建新版本)
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        id path string true "模板 ID"
// @Param        request body service.UpdateTemplateRequest true "模板信息"
// @Success      200  {object}  Response
// @Failure      400  {object}  ErrorResponse
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates/{id} [put]
// @Security     BearerAuth
func (c *TemplateController) Update(ctx *gin.Context) {
	id := ctx.Param("id")
	
	// 验证模板 ID 格式
	if err := utils.ValidateTemplateID(id); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid template id", err.Error())
		return
	}
	
	var req service.UpdateTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid request", err.Error())
		return
	}
	
	// 输入验证和清理
	if err := utils.ValidateTemplateName(req.Name); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid template name", err.Error())
		return
	}
	// 清理模板名称
	req.Name, _ = utils.TrimAndValidate(req.Name, 255)
	if req.Description != "" {
		req.Description, _ = utils.TrimAndValidate(req.Description, 1000)
	}

	template, err := c.templateService.Update(ctx.Request.Context(), id, &req)
	if err != nil {
		// 检查是否是模板不存在的错误
		if strings.Contains(err.Error(), "template not found") {
			Error(ctx, http.StatusNotFound, "template not found", err.Error())
			return
		}
		Error(ctx, http.StatusInternalServerError, "failed to update template", err.Error())
		return
	}

	Success(ctx, template)
}

// Delete 删除模板
// @Summary      删除审批模板
// @Description  删除指定的审批模板
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        id path string true "模板 ID"
// @Success      200  {object}  Response
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates/{id} [delete]
// @Security     BearerAuth
func (c *TemplateController) Delete(ctx *gin.Context) {
	id := ctx.Param("id")
	
	// 验证模板 ID 格式
	if err := utils.ValidateTemplateID(id); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid template id", err.Error())
		return
	}

	if err := c.templateService.Delete(ctx.Request.Context(), id); err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to delete template", err.Error())
		return
	}

	Success(ctx, nil)
}

// List 列出模板
// @Summary      获取模板列表
// @Description  分页获取模板列表,支持搜索和排序
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(20)
// @Param        search query string false "搜索关键词"
// @Param        sort_by query string false "排序字段" default(created_at)
// @Param        order query string false "排序方向" Enums(asc, desc) default(desc)
// @Success      200  {object}  Response
// @Failure      401  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates [get]
// @Security     BearerAuth
func (c *TemplateController) List(ctx *gin.Context) {
	var filter service.TemplateListFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid query parameters", err.Error())
		return
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	response, err := c.templateService.List(&filter)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list templates", err.Error())
		return
	}

	Paginated(ctx, response.Data, PaginationInfo{
		Page:      response.Pagination.Page,
		PageSize:  response.Pagination.PageSize,
		Total:     response.Pagination.Total,
		TotalPage: response.Pagination.TotalPage,
	})
}

// ListVersions 列出模板版本
// @Summary      获取模板版本列表
// @Description  获取指定模板的所有版本号列表
// @Tags         模板管理
// @Accept       json
// @Produce      json
// @Param        id path string true "模板 ID"
// @Success      200  {object}  Response
// @Failure      404  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /templates/{id}/versions [get]
// @Security     BearerAuth
func (c *TemplateController) ListVersions(ctx *gin.Context) {
	id := ctx.Param("id")

	versions, err := c.templateService.ListVersions(id)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list versions", err.Error())
		return
	}

	Success(ctx, versions)
}

