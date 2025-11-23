package repository_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/model"
	"github.com/mautops/approval-gin/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB 创建测试数据库
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// 迁移数据库
	err = db.AutoMigrate(&model.TemplateModel{})
	require.NoError(t, err)

	return db
}

// TestTemplateRepository_Save 测试保存模板
func TestTemplateRepository_Save(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		Data:        []byte(`{"id":"tpl-001","name":"测试模板"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(template)
	assert.NoError(t, err)

	// 验证模板已保存
	var saved model.TemplateModel
	err = db.Where("id = ? AND version = ?", "tpl-001", 1).First(&saved).Error
	assert.NoError(t, err)
	assert.Equal(t, "tpl-001", saved.ID)
	assert.Equal(t, "测试模板", saved.Name)
}

// TestTemplateRepository_FindByID 测试根据 ID 查找模板
func TestTemplateRepository_FindByID(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 先保存模板
	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		Data:        []byte(`{"id":"tpl-001"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := repo.Save(template)
	require.NoError(t, err)

	// 查找模板
	found, err := repo.FindByID("tpl-001", 1)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "tpl-001", found.ID)
	assert.Equal(t, 1, found.Version)
}

// TestTemplateRepository_FindByID_NotFound 测试查找不存在的模板
func TestTemplateRepository_FindByID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 查找不存在的模板
	found, err := repo.FindByID("tpl-999", 1)
	assert.Error(t, err)
	assert.Nil(t, found)
}

// TestTemplateRepository_FindAll 测试查找所有模板
func TestTemplateRepository_FindAll(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 保存多个模板
	for i := 1; i <= 3; i++ {
		template := &model.TemplateModel{
			ID:          "tpl-001",
			Name:        "测试模板",
			Description: "这是一个测试模板",
			Version:     i,
			Data:        []byte(`{"id":"tpl-001"}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.Save(template)
		require.NoError(t, err)
	}

	// 查找所有模板
	templates, err := repo.FindAll()
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(templates), 3)
}

// TestTemplateRepository_Delete 测试删除模板
func TestTemplateRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 先保存模板
	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		Data:        []byte(`{"id":"tpl-001"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := repo.Save(template)
	require.NoError(t, err)

	// 删除模板
	err = repo.Delete("tpl-001")
	assert.NoError(t, err)

	// 验证模板已删除
	var deleted model.TemplateModel
	err = db.Where("id = ?", "tpl-001").First(&deleted).Error
	assert.Error(t, err)
}

// TestTemplateRepository_FindByID_LatestVersion 测试查找最新版本
func TestTemplateRepository_FindByID_LatestVersion(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 保存多个版本的模板
	for i := 1; i <= 3; i++ {
		template := &model.TemplateModel{
			ID:          "tpl-001",
			Name:        "测试模板",
			Description: "这是一个测试模板",
			Version:     i,
			Data:        []byte(`{"id":"tpl-001","version":` + strconv.Itoa(i) + `}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.Save(template)
		require.NoError(t, err)
	}

	// 查找最新版本(version = 0 表示最新版本)
	found, err := repo.FindByID("tpl-001", 0)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, 3, found.Version) // 应该返回版本 3
}

// TestTemplateRepository_Save_Update 测试更新已存在的模板
func TestTemplateRepository_Save_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 先保存模板
	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		Data:        []byte(`{"id":"tpl-001"}`),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	err := repo.Save(template)
	require.NoError(t, err)

	// 更新模板
	template.Name = "更新后的模板"
	template.Description = "更新后的描述"
	err = repo.Save(template)
	assert.NoError(t, err)

	// 验证模板已更新
	found, err := repo.FindByID("tpl-001", 1)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, "更新后的模板", found.Name)
	assert.Equal(t, "更新后的描述", found.Description)
}

// TestTemplateRepository_Delete_NotFound 测试删除不存在的模板
func TestTemplateRepository_Delete_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 删除不存在的模板(应该不报错)
	err := repo.Delete("tpl-999")
	assert.NoError(t, err)
}

// TestTemplateRepository_FindAll_Empty 测试查找空列表
func TestTemplateRepository_FindAll_Empty(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 查找所有模板(应该返回空列表)
	templates, err := repo.FindAll()
	assert.NoError(t, err)
	assert.Empty(t, templates)
}

// TestTemplateRepository_Save_EmptyData 测试保存空数据
func TestTemplateRepository_Save_EmptyData(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	template := &model.TemplateModel{
		ID:          "tpl-001",
		Name:        "测试模板",
		Description: "这是一个测试模板",
		Version:     1,
		Data:        []byte{},
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	err := repo.Save(template)
	assert.NoError(t, err)

	// 验证模板已保存
	found, err := repo.FindByID("tpl-001", 1)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Empty(t, found.Data)
}

// TestTemplateRepository_FindByID_MultipleVersions 测试查找特定版本
func TestTemplateRepository_FindByID_MultipleVersions(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewTemplateRepository(db)

	// 保存多个版本的模板
	for i := 1; i <= 5; i++ {
		template := &model.TemplateModel{
			ID:          "tpl-001",
			Name:        "测试模板",
			Description: "这是一个测试模板",
			Version:     i,
			Data:        []byte(`{"id":"tpl-001","version":` + strconv.Itoa(i) + `}`),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
		}
		err := repo.Save(template)
		require.NoError(t, err)
	}

	// 查找版本 3
	found, err := repo.FindByID("tpl-001", 3)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, 3, found.Version)

	// 查找版本 5
	found, err = repo.FindByID("tpl-001", 5)
	assert.NoError(t, err)
	assert.NotNil(t, found)
	assert.Equal(t, 5, found.Version)

	// 查找不存在的版本
	found, err = repo.FindByID("tpl-001", 10)
	assert.Error(t, err)
	assert.Nil(t, found)
}
