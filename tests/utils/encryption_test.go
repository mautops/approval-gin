package utils_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEncryptDecrypt 测试加密和解密功能
func TestEncryptDecrypt(t *testing.T) {
	// 测试数据
	plaintext := "sensitive-data-123"
	key := "12345678901234567890123456789012" // 32 字节密钥

	// 加密
	encrypted, err := utils.Encrypt(plaintext, key)
	require.NoError(t, err, "Encryption should succeed")
	assert.NotEmpty(t, encrypted, "Encrypted data should not be empty")
	assert.NotEqual(t, plaintext, encrypted, "Encrypted data should be different from plaintext")

	// 解密
	decrypted, err := utils.Decrypt(encrypted, key)
	require.NoError(t, err, "Decryption should succeed")
	assert.Equal(t, plaintext, decrypted, "Decrypted data should match original")
}

// TestEncryptDecrypt_DifferentKeys 测试使用不同密钥解密应该失败
func TestEncryptDecrypt_DifferentKeys(t *testing.T) {
	plaintext := "sensitive-data-123"
	key1 := "12345678901234567890123456789012" // 32 字节
	key2 := "abcdefghijklmnopqrstuvwxyz123456" // 32 字节

	// 使用 key1 加密
	encrypted, err := utils.Encrypt(plaintext, key1)
	require.NoError(t, err, "Encryption should succeed")

	// 使用 key2 解密应该失败
	_, err = utils.Decrypt(encrypted, key2)
	assert.Error(t, err, "Decryption with wrong key should fail")
}

// TestEncryptDecrypt_EmptyData 测试空数据加密解密
func TestEncryptDecrypt_EmptyData(t *testing.T) {
	plaintext := ""
	key := "12345678901234567890123456789012" // 32 字节

	encrypted, err := utils.Encrypt(plaintext, key)
	require.NoError(t, err, "Encryption of empty data should succeed")

	decrypted, err := utils.Decrypt(encrypted, key)
	require.NoError(t, err, "Decryption should succeed")
	assert.Equal(t, plaintext, decrypted, "Decrypted empty data should match original")
}

// TestEncryptDecrypt_LongData 测试长数据加密解密
func TestEncryptDecrypt_LongData(t *testing.T) {
	// 创建一个较长的测试数据
	plaintext := make([]byte, 1000)
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	key := "12345678901234567890123456789012" // 32 字节

	encrypted, err := utils.Encrypt(string(plaintext), key)
	require.NoError(t, err, "Encryption of long data should succeed")

	decrypted, err := utils.Decrypt(encrypted, key)
	require.NoError(t, err, "Decryption should succeed")
	assert.Equal(t, string(plaintext), decrypted, "Decrypted long data should match original")
}

// TestEncryptDecrypt_InvalidKey 测试无效密钥
func TestEncryptDecrypt_InvalidKey(t *testing.T) {
	plaintext := "sensitive-data-123"
	invalidKey := "short" // 密钥太短

	_, err := utils.Encrypt(plaintext, invalidKey)
	assert.Error(t, err, "Encryption with invalid key should fail")
}

// TestHashPassword 测试密码哈希
func TestHashPassword(t *testing.T) {
	password := "my-secret-password"

	hashed, err := utils.HashPassword(password)
	require.NoError(t, err, "Password hashing should succeed")
	assert.NotEmpty(t, hashed, "Hashed password should not be empty")
	assert.NotEqual(t, password, hashed, "Hashed password should be different from original")

	// 验证密码
	valid := utils.VerifyPassword(password, hashed)
	assert.True(t, valid, "Password verification should succeed")

	// 验证错误密码
	valid = utils.VerifyPassword("wrong-password", hashed)
	assert.False(t, valid, "Wrong password verification should fail")
}

// TestHashPassword_SamePasswordDifferentHash 测试相同密码生成不同的哈希
func TestHashPassword_SamePasswordDifferentHash(t *testing.T) {
	password := "my-secret-password"

	hashed1, err1 := utils.HashPassword(password)
	require.NoError(t, err1)

	hashed2, err2 := utils.HashPassword(password)
	require.NoError(t, err2)

	// 相同密码应该生成不同的哈希（因为使用了随机 salt）
	assert.NotEqual(t, hashed1, hashed2, "Same password should generate different hashes")

	// 但都应该能验证通过
	assert.True(t, utils.VerifyPassword(password, hashed1), "First hash should verify correctly")
	assert.True(t, utils.VerifyPassword(password, hashed2), "Second hash should verify correctly")
}

