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

// loadDB 加载数据库：自动发现或使用指定路径，支持 v2/v3 解密
func loadDB() (*gorm.DB, error) {
	dbPath, userID, version, userConfigPath, err := resolveDBPath()
	if err != nil {
		return nil, err
	}

	finalPath := dbPath

	// 需要解密时，根据版本选择密钥生成方式
	if userID != "" {
		key, keyErr := generateKey(userID, version, userConfigPath)
		if keyErr != nil {
			return nil, keyErr
		}

		if flagVerbose {
			fmt.Fprintf(os.Stderr, "[信息] 正在解密 %s 数据库，用户 UID: %s\n", version, userID)
		}

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
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[信息] 正在加载数据库: %s\n", finalPath)
	}

	db, err := database.MigrateToMemory(finalPath)
	if err != nil {
		return nil, fmt.Errorf("加载数据库失败: %w", err)
	}

	// 解密后的临时文件可以删除（数据已在内存中）
	if finalPath != dbPath {
		_ = os.Remove(finalPath)
	}

	return db, nil
}

// generateKey 根据版本生成对应的解密密钥
func generateKey(userID, version, userConfigPath string) ([]byte, error) {
	switch version {
	case "v3":
		if userConfigPath == "" {
			return nil, fmt.Errorf("v3 数据库需要 user_config 文件，但未找到。\n请确认 user_config 文件存在于数据库目录中，或使用 --salt 参数手动指定 salt 值")
		}
		salt, err := crypto.ReadSaltFromUserConfig(userConfigPath)
		if err != nil {
			return nil, fmt.Errorf("读取 v3 salt 失败: %w", err)
		}
		if flagVerbose {
			fmt.Fprintf(os.Stderr, "[信息] v3 salt 已读取，正在派生密钥\n")
		}
		return crypto.GenerateKeyV3(userID, salt), nil
	default:
		// v2 或未知版本，使用 v2 算法
		return crypto.GenerateKeyV2(userID), nil
	}
}

// resolveDBPath 解析数据库路径、用户 ID、版本及 user_config 路径
func resolveDBPath() (dbPath, userID, version, userConfigPath string, err error) {
	// 优先使用命令行指定的路径
	if flagDBPath != "" {
		if _, statErr := os.Stat(flagDBPath); statErr != nil {
			err = fmt.Errorf("数据库文件不存在: %s", flagDBPath)
			return
		}
		dbPath = flagDBPath
		userID = flagUserID
		version = flagVersion
		userConfigPath = flagUserConfigPath
		return
	}

	// 自动发现
	locations, discoverErr := finder.FindDingTalkDBs()
	if discoverErr != nil {
		err = fmt.Errorf("自动发现数据库失败: %w", discoverErr)
		return
	}

	if len(locations) == 0 {
		err = fmt.Errorf("未找到钉钉数据库，请使用 -d 参数手动指定路径")
		return
	}

	// 多个账号时选第一个，并提示
	loc := locations[0]
	if len(locations) > 1 {
		fmt.Fprintf(os.Stderr, "[提示] 发现多个钉钉账号，使用第一个 (UID: %s, 版本: %s)\n", loc.UserID, loc.Version)
		fmt.Fprintf(os.Stderr, "       使用 -d 参数指定其他数据库路径\n")
	}

	dbPath = loc.DBPath
	version = loc.Version
	userConfigPath = loc.UserConfigPath

	// 自动发现时，如果未指定 key，使用目录中的 UID 作为解密密钥
	if flagUserID != "" {
		userID = flagUserID
	} else {
		userID = loc.UserID
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[信息] 自动发现数据库: %s (UID: %s, 版本: %s)\n", dbPath, userID, version)
	}

	return
}
