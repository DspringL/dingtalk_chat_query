// Package crypto 提供钉钉数据库解密功能
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

const pageSize = 4096

// GenerateKey 根据用户 UID 生成 AES 解密密钥
func GenerateKey(userID string) []byte {
	hash := md5.Sum([]byte(userID))
	hexHash := hex.EncodeToString(hash[:])
	return []byte(hexHash[:16])
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
