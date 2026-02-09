package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	encPrefix  = "enc:"
	pbkdf2Iter = 100000
	keyLen     = 32
	saltFixed  = "vmcat-salt-2026"
)

// deriveKey 从机器标识派生 AES-256 密钥
func deriveKey() []byte {
	machineID := getMachineID()
	salt := []byte(saltFixed + machineID)
	return pbkdf2.Key([]byte(machineID), salt, pbkdf2Iter, keyLen, sha256.New)
}

// Encrypt AES-256-GCM 加密，返回 "enc:base64" 格式
func Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	encoded := base64.StdEncoding.EncodeToString(ciphertext)
	return encPrefix + encoded, nil
}

// Decrypt 解密 "enc:base64" 格式的密文
func Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	// 非加密格式直接返回（兼容旧数据）
	if !strings.HasPrefix(ciphertext, encPrefix) {
		return ciphertext, nil
	}

	encoded := strings.TrimPrefix(ciphertext, encPrefix)
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("base64 decode: %w", err)
	}

	key := deriveKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ctext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ctext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w", err)
	}

	return string(plaintext), nil
}

// IsEncrypted 检查是否已加密
func IsEncrypted(s string) bool {
	return strings.HasPrefix(s, encPrefix)
}

// getMachineID 获取机器唯一标识
func getMachineID() string {
	switch runtime.GOOS {
	case "darwin":
		// macOS: IOPlatformUUID
		data, err := os.ReadFile("/etc/machine-id")
		if err == nil && len(strings.TrimSpace(string(data))) > 0 {
			return strings.TrimSpace(string(data))
		}
		// 备选: 用hostname
		name, _ := os.Hostname()
		if name != "" {
			return name
		}
	case "linux":
		data, err := os.ReadFile("/etc/machine-id")
		if err == nil {
			return strings.TrimSpace(string(data))
		}
		data, err = os.ReadFile("/var/lib/dbus/machine-id")
		if err == nil {
			return strings.TrimSpace(string(data))
		}
	case "windows":
		// Windows: MachineGuid from registry
		name, _ := os.Hostname()
		if name != "" {
			return name
		}
	}
	// 最终备选
	return "vmcat-default-machine-id"
}
