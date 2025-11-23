# Approval Gin

基于 `approval-kit` 审批流核心库的 REST API 服务,提供审批模板和审批任务的完整 API 接口.

## 项目结构

```
approval-gin/
├── cmd/              # 命令行工具
│   ├── root.go      # 根命令
│   └── server.go    # 服务器启动命令
├── internal/         # 内部代码
│   ├── api/         # API 层
│   ├── service/     # 服务层
│   ├── repository/  # 数据访问层
│   └── model/       # 数据模型
├── tests/           # 测试文件
├── docs/            # 文档
├── migrations/      # 数据库迁移脚本
├── go.mod
├── go.sum
└── main.go
```

## 快速开始

### 安装依赖

```bash
go mod download
```

### 配置环境变量

创建 `.env` 文件或设置环境变量:

```bash
# 数据库配置
APP_DATABASE_HOST=localhost
APP_DATABASE_PORT=5432
APP_DATABASE_USER=postgres
APP_DATABASE_PASSWORD=password
APP_DATABASE_NAME=approval

# 服务器配置
APP_SERVER_HOST=0.0.0.0
APP_SERVER_PORT=8080

# Keycloak 配置
APP_KEYCLOAK_ISSUER=https://keycloak.example.com/realms/your-realm

# OpenFGA 配置
APP_OPENFGA_API_URL=http://localhost:8081
APP_OPENFGA_STORE_ID=your-store-id
APP_OPENFGA_MODEL_ID=your-model-id
```

### 运行服务

```bash
# 开发模式
go run main.go server

# 或使用编译后的二进制文件
go build -o approval-gin main.go
./approval-gin server
```

### 数据库迁移

```bash
go run main.go migrate
```

## API 概览

### 模板管理 API

- `POST /api/v1/templates` - 创建模板
- `GET /api/v1/templates` - 获取模板列表
- `GET /api/v1/templates/:id` - 获取模板详情
- `PUT /api/v1/templates/:id` - 更新模板
- `DELETE /api/v1/templates/:id` - 删除模板
- `GET /api/v1/templates/:id/versions` - 获取模板版本列表

### 任务管理 API

- `POST /api/v1/tasks` - 创建任务
- `GET /api/v1/tasks` - 获取任务列表
- `GET /api/v1/tasks/:id` - 获取任务详情
- `POST /api/v1/tasks/:id/submit` - 提交任务
- `POST /api/v1/tasks/:id/approve` - 审批同意
- `POST /api/v1/tasks/:id/reject` - 审批拒绝
- `POST /api/v1/tasks/:id/cancel` - 取消任务
- `POST /api/v1/tasks/:id/withdraw` - 撤回任务
- `POST /api/v1/tasks/:id/transfer` - 转交审批
- `POST /api/v1/tasks/:id/approvers` - 添加审批人
- `DELETE /api/v1/tasks/:id/approvers` - 移除审批人
- `POST /api/v1/tasks/:id/pause` - 暂停任务
- `POST /api/v1/tasks/:id/resume` - 恢复任务
- `POST /api/v1/tasks/:id/rollback` - 回退到指定节点
- `POST /api/v1/tasks/:id/approvers/replace` - 替换审批人

### 查询和统计 API

- `GET /api/v1/tasks` - 查询任务列表(支持多条件查询、分页、排序)
- `GET /api/v1/tasks/:id/records` - 获取审批记录
- `GET /api/v1/tasks/:id/history` - 获取状态历史
- `GET /api/v1/statistics/tasks` - 任务统计
- `GET /api/v1/statistics/approvals` - 审批统计

## 使用示例

### 创建模板

```bash
curl -X POST http://localhost:8080/api/v1/templates \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "name": "请假审批模板",
    "description": "员工请假审批流程",
    "nodes": {
      "start": {
        "id": "start",
        "name": "开始",
        "type": "start",
        "order": 1
      },
      "approval": {
        "id": "approval",
        "name": "部门经理审批",
        "type": "approval",
        "order": 2
      },
      "end": {
        "id": "end",
        "name": "结束",
        "type": "end",
        "order": 3
      }
    },
    "edges": [
      {"from": "start", "to": "approval"},
      {"from": "approval", "to": "end"}
    ]
  }'
```

### 创建任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "template_id": "tpl-001",
    "business_id": "leave-001",
    "params": {
      "days": 3,
      "reason": "年假"
    }
  }'
```

### 提交任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks/task-001/submit \
  -H "Authorization: Bearer <token>"
```

### 审批任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks/task-001/approve \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <token>" \
  -d '{
    "node_id": "approval",
    "comment": "同意"
  }'
```

## API 文档

启动服务后,访问 Swagger UI 查看完整的 API 文档:

```
http://localhost:8080/swagger/index.html
```

## 开发规范

- 严格遵循 TDD 开发原则
- 所有功能必须基于 approval-kit 实现
- 代码必须通过 go vet 检查
- 所有公共 API 必须有文档注释和 Swagger 注释

## 部署

详细的部署说明请参考 [部署文档](docs/DEPLOYMENT.md).

## License

Copyright © 2025

