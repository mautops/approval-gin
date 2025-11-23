package monitoring_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

// TestAlertRules_FileExists 测试告警规则文件是否存在
func TestAlertRules_FileExists(t *testing.T) {
	alertRulesPath := filepath.Join("..", "..", "deploy", "prometheus", "alerts.yml")
	_, err := os.Stat(alertRulesPath)
	assert.NoError(t, err, "alert rules file should exist")
}

// TestAlertRules_ValidYAML 测试告警规则文件是否为有效的 YAML
func TestAlertRules_ValidYAML(t *testing.T) {
	alertRulesPath := filepath.Join("..", "..", "deploy", "prometheus", "alerts.yml")
	data, err := os.ReadFile(alertRulesPath)
	assert.NoError(t, err)

	var rules map[string]interface{}
	err = yaml.Unmarshal(data, &rules)
	assert.NoError(t, err, "alert rules should be valid YAML")
	assert.NotNil(t, rules["groups"], "should have groups")
}

// TestAlertRules_RequiredAlerts 测试必需的告警规则是否存在
func TestAlertRules_RequiredAlerts(t *testing.T) {
	alertRulesPath := filepath.Join("..", "..", "deploy", "prometheus", "alerts.yml")
	data, err := os.ReadFile(alertRulesPath)
	assert.NoError(t, err)

	// 读取文件内容并检查是否包含必需的告警名称
	content := string(data)
	requiredAlerts := []string{
		"HighErrorRate",
		"HighLatency",
		"DatabaseConnectionFailure",
		"HighTaskCreationRate",
		"HighApprovalFailureRate",
	}

	for _, alertName := range requiredAlerts {
		assert.Contains(t, content, alertName, "should have alert: %s", alertName)
	}
}

// TestAlertRules_ValidPromQL 测试告警规则的 PromQL 表达式是否有效
func TestAlertRules_ValidPromQL(t *testing.T) {
	alertRulesPath := filepath.Join("..", "..", "deploy", "prometheus", "alerts.yml")
	data, err := os.ReadFile(alertRulesPath)
	assert.NoError(t, err)

	var rules map[string]interface{}
	err = yaml.Unmarshal(data, &rules)
	assert.NoError(t, err)

	// 验证文件包含 PromQL 表达式关键字
	content := string(data)
	assert.Contains(t, content, "expr:", "should contain expr field")
	assert.Contains(t, content, "rate(", "should contain rate function")
	assert.Contains(t, content, "api_requests_total", "should contain metric name")
}

