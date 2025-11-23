package auth

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/openfga/go-sdk/client"
	"github.com/openfga/go-sdk/credentials"
)

// OpenFGAClient OpenFGA 客户端
type OpenFGAClient struct {
	client  *client.OpenFgaClient
	storeID string
	modelID string
}

// NewOpenFGAClient 创建 OpenFGA 客户端
func NewOpenFGAClient(apiURL string, storeID string, modelID string) (*OpenFGAClient, error) {
	configuration := client.ClientConfiguration{
		ApiUrl:  apiURL,
		StoreId: storeID,
		Credentials: &credentials.Credentials{
			Method: credentials.CredentialsMethodNone,
		},
	}

	fgaClient, err := client.NewSdkClient(&configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenFGA client: %w", err)
	}

	return &OpenFGAClient{
		client:  fgaClient,
		storeID: storeID,
		modelID: modelID,
	}, nil
}

// CheckPermission 检查权限
func (c *OpenFGAClient) CheckPermission(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) (bool, error) {
	body := client.ClientCheckRequest{
		User:     fmt.Sprintf("user:%s", userID),
		Relation: relation,
		Object:   fmt.Sprintf("%s:%s", objectType, objectID),
	}

	response, err := c.client.Check(ctx).Body(body).Execute()
	if err != nil {
		return false, fmt.Errorf("failed to check permission: %w", err)
	}

	return response.GetAllowed(), nil
}

// SetRelation 设置权限关系
func (c *OpenFGAClient) SetRelation(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) error {
	body := client.ClientWriteRequest{
		Writes: []client.ClientTupleKey{
			{
				User:     fmt.Sprintf("user:%s", userID),
				Relation: relation,
				Object:   fmt.Sprintf("%s:%s", objectType, objectID),
			},
		},
	}

	_, err := c.client.Write(ctx).Body(body).Execute()
	if err != nil {
		return fmt.Errorf("failed to set relation: %w", err)
	}

	return nil
}

// DeleteRelation 删除权限关系
func (c *OpenFGAClient) DeleteRelation(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) error {
	body := client.ClientWriteRequest{
		Deletes: []client.ClientTupleKeyWithoutCondition{
			{
				User:     fmt.Sprintf("user:%s", userID),
				Relation: relation,
				Object:   fmt.Sprintf("%s:%s", objectType, objectID),
			},
		},
	}

	_, err := c.client.Write(ctx).Body(body).Execute()
	if err != nil {
		return fmt.Errorf("failed to delete relation: %w", err)
	}

	return nil
}

// PermissionMiddleware 权限检查中间件
func PermissionMiddleware(
	fgaClient *OpenFGAClient,
	objectType string,
	relation string,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "unauthorized",
			})
			c.Abort()
			return
		}

		objectID := c.Param("id")
		if objectID == "" {
			objectID = c.Query("id")
		}

		allowed, err := fgaClient.CheckPermission(
			c.Request.Context(),
			userID.(string),
			relation,
			objectType,
			objectID,
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "permission check failed",
				"detail":  err.Error(),
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "forbidden",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// NewOpenFGAClientWithRetry 带重试的 OpenFGA 客户端创建
func NewOpenFGAClientWithRetry(apiURL string, storeID string, modelID string, maxRetries int, retryInterval time.Duration) (*OpenFGAClient, error) {
	var fgaClient *OpenFGAClient
	var err error

	for i := 0; i < maxRetries; i++ {
		fgaClient, err = NewOpenFGAClient(apiURL, storeID, modelID)
		if err == nil {
			// 测试连接
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, testErr := fgaClient.client.Read(ctx).Execute()
			cancel()
			if testErr == nil {
				return fgaClient, nil
			}
		}

		// 如果不是最后一次重试，等待后重试
		if i < maxRetries-1 {
			time.Sleep(retryInterval)
			retryInterval *= 2 // 指数退避
		}
	}

	return nil, fmt.Errorf("failed to create OpenFGA client after %d retries: %w", maxRetries, err)
}

// CheckHealth 检查 OpenFGA 连接健康状态
func (c *OpenFGAClient) CheckHealth(ctx context.Context) bool {
	if c == nil || c.client == nil {
		return false
	}

	// 尝试执行一个简单的读取操作来检查连接
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err := c.client.Read(ctx).Execute()
	return err == nil
}

// Reconnect 重新连接 OpenFGA
func (c *OpenFGAClient) Reconnect(apiURL string, storeID string, modelID string) error {
	newClient, err := NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		return fmt.Errorf("failed to reconnect OpenFGA: %w", err)
	}

	c.client = newClient.client
	c.storeID = newClient.storeID
	c.modelID = newClient.modelID

	return nil
}

