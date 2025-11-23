package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/service"
	"github.com/mautops/approval-kit/pkg/types"
)

// QueryController 查询控制器
type QueryController struct {
	queryService service.QueryService
}

// NewQueryController 创建查询控制器
func NewQueryController(queryService service.QueryService) *QueryController {
	return &QueryController{
		queryService: queryService,
	}
}

// ListTasks 列出任务
// @Summary      获取任务列表
// @Description  分页获取任务列表,支持多条件查询、排序
// @Tags         查询统计
// @Accept       json
// @Produce      json
// @Param        state query string false "任务状态"
// @Param        template_id query string false "模板 ID"
// @Param        business_id query string false "业务 ID"
// @Param        approver query string false "审批人"
// @Param        created_at_start query string false "创建时间起始"
// @Param        created_at_end query string false "创建时间结束"
// @Param        page query int false "页码" default(1)
// @Param        page_size query int false "每页数量" default(20)
// @Param        sort_by query string false "排序字段" default(created_at)
// @Param        order query string false "排序方向" Enums(asc, desc) default(desc)
// @Success      200  {object}  PaginatedResponse
// @Failure      400  {object}  ErrorResponse
// @Failure      500  {object}  ErrorResponse
// @Router       /tasks [get]
// @Security     BearerAuth
func (c *QueryController) ListTasks(ctx *gin.Context) {
	var filter service.ListTasksFilter
	if err := ctx.ShouldBindQuery(&filter); err != nil {
		Error(ctx, http.StatusBadRequest, "invalid query parameters", err.Error())
		return
	}

	// 手动解析 state 参数（因为 Gin 无法直接将字符串绑定到 types.TaskState）
	if stateStr := ctx.Query("state"); stateStr != "" {
		state := types.TaskState(stateStr)
		filter.State = &state
	}

	// 手动解析 page_size 参数（因为 Gin 可能无法正确绑定下划线参数）
	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		var pageSize int
		if _, err := fmt.Sscanf(pageSizeStr, "%d", &pageSize); err == nil && pageSize > 0 {
			filter.PageSize = pageSize
		}
	}

	// 手动解析 page 参数
	if pageStr := ctx.Query("page"); pageStr != "" {
		var page int
		if _, err := fmt.Sscanf(pageStr, "%d", &page); err == nil && page > 0 {
			filter.Page = page
		}
	}

	// 设置默认值
	if filter.Page <= 0 {
		filter.Page = 1
	}
	if filter.PageSize <= 0 {
		filter.PageSize = 20
	}

	tasks, total, err := c.queryService.ListTasks(&filter)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to list tasks", err.Error())
		return
	}

	// 计算总页数
	totalPage := int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	Paginated(ctx, tasks, PaginationInfo{
		Page:      filter.Page,
		PageSize:  filter.PageSize,
		Total:     total,
		TotalPage: totalPage,
	})
}

// GetRecords 获取审批记录
func (c *QueryController) GetRecords(ctx *gin.Context) {
	taskID := ctx.Param("id")

	records, err := c.queryService.GetRecords(taskID)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to get records", err.Error())
		return
	}

	Success(ctx, records)
}

// GetHistory 获取状态历史
func (c *QueryController) GetHistory(ctx *gin.Context) {
	taskID := ctx.Param("id")

	history, err := c.queryService.GetHistory(taskID)
	if err != nil {
		Error(ctx, http.StatusInternalServerError, "failed to get history", err.Error())
		return
	}

	Success(ctx, history)
}

