package auth

// GetPermissionModel 获取 OpenFGA 权限模型定义
func GetPermissionModel() string {
	return `model
  schema 1.1

type user

type template
  relations
    define owner: [user]
    define viewer: [user]
    define editor: [user] or owner
    define deleter: [user] or owner

type task
  relations
    define creator: [user]
    define approver: [user]
    define viewer: [user] or creator or approver
    define operator: [user] or creator or approver`
}


