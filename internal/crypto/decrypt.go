// Package crypto 提供钉钉数据库解密功能
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

const pageSize = 4096

// v3 PBKDF2 参数（通过逆向分析得出）
const (
	v3PBKDFSalt       = "666DingTalk888"
	v3PBKDFIterations = 1000
	v3PBKDFKeyLen     = 32
)

// GenerateKeyV2 根据用户 UID 生成 v2 版本 AES 解密密钥
// 算法：MD5(uid) 取前 16 位十六进制字符
func GenerateKeyV2(userID string) []byte {
	hash := md5.Sum([]byte(userID))
	hexHash := hex.EncodeToString(hash[:])
	return []byte(hexHash[:16])
}

// GenerateKey 兼容旧版接口，等同于 GenerateKeyV2
func GenerateKey(userID string) []byte {
	return GenerateKeyV2(userID)
}

// GenerateKeyV3 根据用户 UID 和 salt 生成 v3 版本 AES 解密密钥
// 算法：MD5(PBKDF2-HMAC-SHA1(uid+salt, "666DingTalk888", 1000, 32))[:16]
func GenerateKeyV3(userID, salt string) []byte {
	// 1. 拼接 uid + salt 作为 password
	password := []byte(userID + salt)

	// 2. PBKDF2-HMAC-SHA1 派生 32 字节
	derived := pbkdf2.Key(password, []byte(v3PBKDFSalt), v3PBKDFIterations, v3PBKDFKeyLen, sha1.New)

	// 3. 对派生结果求 MD5，取前 16 位十六进制字符作为密钥
	hash := md5.Sum(derived)
	hexHash := hex.EncodeToString(hash[:])
	return []byte(hexHash[:16])
}

// ReadSaltFromUserConfig 从 v3 user_config 文件中读取 salt 值
// user_config 内容是一段 Base64 编码的 JSON，JSON 中含有 "salt" 字段
func ReadSaltFromUserConfig(configPath string) (string, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("读取 user_config 失败: %w", err)
	}

	// 去掉可能存在的换行/空格后解码 Base64
	decoded, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		// 尝试 RawStdEncoding（无 padding）
		decoded, err = base64.RawStdEncoding.DecodeString(string(data))
		if err != nil {
			return "", fmt.Errorf("Base64 解码 user_config 失败: %w", err)
		}
	}

	var config map[string]any
	if err := json.Unmarshal(decoded, &config); err != nil {
		return "", fmt.Errorf("解析 user_config JSON 失败: %w", err)
	}

	salt, ok := config["salt"].(string)
	if !ok || salt == "" {
		return "", fmt.Errorf("user_config 中未找到 salt 字段")
	}
	return salt, nil
}

// DecryptDatabase 解密钉钉 v2 加密数据库（AES-ECB，按页解密）
func DecryptDatabase(inputPath, outputPath string, key []byte) error {
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("创建 AES cipher 失败: %w", err)
	}

	inFile, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("打开输入文件失败: %w", err)
	}
	defer inFile.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer outFile.Close()

	buf := make([]byte, pageSize)
	for {
		n, err := inFile.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			_ = os.Remove(outputPath) // 读取失败，清理不完整的输出文件
			return fmt.Errorf("读取输入文件失败: %w", err)
		}

		if n == pageSize {
			decryptECB(block, buf)
		}

		if _, err := outFile.Write(buf[:n]); err != nil {
			_ = os.Remove(outputPath) // 写入失败，清理不完整的输出文件
			return fmt.Errorf("写入输出文件失败: %w", err)
		}
	}

	return nil
}

// decryptECB 使用 AES-ECB 模式逐块解密
func decryptECB(block cipher.Block, data []byte) {
	blockSize := block.BlockSize()
	for i := 0; i < len(data); i += blockSize {
		block.Decrypt(data[i:i+blockSize], data[i:i+blockSize])
	}
}
