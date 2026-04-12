// Package cmd 提供数据库加载逻辑（自动发现 + 解密 + 迁移）
package cmd

import (
	"fmt"
	"os"

	"dingtalk-cli/internal/crypto"
	"dingtalk-cli/internal/database"
	"dingtalk-cli/internal/finder"

	"gorm.io/gorm"
)

// loadDB 加载数据库：自动发现或使用指定路径，支持解密
func loadDB() (*gorm.DB, error) {
	dbPath, userID, err := resolveDBPath()
	if err != nil {
		return nil, err
	}

	finalPath := dbPath

	// 如果需要解密
	if userID != "" {
		if flagVerbose {
			fmt.Fprintf(os.Stderr, "[信息] 正在解密数据库，用户 UID: %s\n", userID)
		}

		key := crypto.GenerateKey(userID)

		tmpFile, err := os.CreateTemp("", "dtchat-*.db")
		if err != nil {
			return nil, fmt.Errorf("创建临时文件失败: %w", err)
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()

		if err := crypto.DecryptDatabase(dbPath, tmpPath, key); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("解密失败: %w", err)
		}

		if err := database.ValidateDB(tmpPath); err != nil {
			_ = os.Remove(tmpPath)
			return nil, fmt.Errorf("解密后数据库验证失败（密钥可能错误）: %w", err)
		}

		finalPath = tmpPath
		// 注册清理函数（进程退出时删除临时文件）
		defer func() {
			// 注意：这里不能 defer os.Remove，因为 db 还在使用中
			// 临时文件会在进程退出时由 OS 清理，或在 main 中处理
		}()
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[信息] 正在加载数据库: %s\n", finalPath)
	}

	db, err := database.MigrateToMemory(finalPath)
	if err != nil {
		return nil, fmt.Errorf("加载数据库失败: %w", err)
	}

	// 解密后的临时文件可以删除（数据已在内存中）
	if userID != "" && finalPath != dbPath {
		_ = os.Remove(finalPath)
	}

	return db, nil
}

// resolveDBPath 解析数据库路径和用户 ID
func resolveDBPath() (dbPath, userID string, err error) {
	// 优先使用命令行指定的路径
	if flagDBPath != "" {
		dbPath = flagDBPath
		userID = flagUserID
		return
	}

	// 自动发现
	locations, err := finder.FindDingTalkDBs()
	if err != nil {
		return "", "", fmt.Errorf("自动发现数据库失败: %w", err)
	}

	if len(locations) == 0 {
		return "", "", fmt.Errorf("未找到钉钉数据库，请使用 -d 参数手动指定路径")
	}

	// 多个账号时选第一个，并提示
	loc := locations[0]
	if len(locations) > 1 {
		fmt.Fprintf(os.Stderr, "[提示] 发现多个钉钉账号，使用第一个 (UID: %s)\n", loc.UserID)
		fmt.Fprintf(os.Stderr, "       使用 -d 参数指定其他数据库路径\n")
	}

	dbPath = loc.DBPath
	// 自动发现时，如果未指定 key，使用目录中的 UID 作为解密密钥
	if flagUserID != "" {
		userID = flagUserID
	} else {
		userID = loc.UserID
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[信息] 自动发现数据库: %s (UID: %s)\n", dbPath, userID)
	}

	return
}
