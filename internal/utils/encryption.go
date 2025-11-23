package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

// Encrypt 加密敏感数据（使用 AES-256-GCM）
func Encrypt(plaintext string, key string) (string, error) {
	// 验证密钥长度（至少 32 字节）
	if len(key) < 32 {
		return "", errors.New("key must be at least 32 bytes long")
	}

	// 将密钥转换为 32 字节（使用 SHA-256 哈希）
	keyHash := sha256.Sum256([]byte(key))
	keyBytes := keyHash[:]

	// 创建 AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 生成随机 nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// 加密数据
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// 返回 base64 编码的密文
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt 解密敏感数据（使用 AES-256-GCM）
func Decrypt(ciphertext string, key string) (string, error) {
	// 验证密钥长度
	if len(key) < 32 {
		return "", errors.New("key must be at least 32 bytes long")
	}

	// 解码 base64
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// 将密钥转换为 32 字节
	keyHash := sha256.Sum256([]byte(key))
	keyBytes := keyHash[:]

	// 创建 AES cipher
	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// 创建 GCM
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// 提取 nonce
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]

	// 解密数据
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// HashPassword 哈希密码（使用 bcrypt）
func HashPassword(password string) (string, error) {
	// 使用 bcrypt 默认 cost (10)
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hashedBytes), nil
}

// VerifyPassword 验证密码
func VerifyPassword(password string, hashedPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}


