// Package finder 提供自动发现钉钉数据库路径的功能
package finder

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// DBLocation 钉钉数据库位置信息
type DBLocation struct {
	UserID         string // 用户 UID
	DBPath         string // 数据库文件路径
	Version        string // 版本目录名（如 v2、v3）
	UserConfigPath string // v3 专用：user_config 文件路径（为空则表示 v2）
}

// FindDingTalkDBs 自动发现本机所有钉钉数据库
func FindDingTalkDBs() ([]DBLocation, error) {
	switch runtime.GOOS {
	case "darwin":
		return findMacOS()
	case "windows":
		return findWindows()
	case "linux":
		return findLinux()
	default:
		return nil, fmt.Errorf("不支持的操作系统: %s", runtime.GOOS)
	}
}

// findMacOS 在 macOS 上查找钉钉数据库
// 路径：~/Library/Containers/5ZSL2CJU2T.com.dingtalk.mac/Data/Library/Application Support/DingTalkMac/{uid}_v2/DBFiles/dingtalk.db
func findMacOS() ([]DBLocation, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	baseDir := filepath.Join(homeDir, "Library", "Containers",
		"5ZSL2CJU2T.com.dingtalk.mac", "Data", "Library",
		"Application Support", "DingTalkMac")

	results, err := scanUserDirs(baseDir)
	if err != nil {
		return nil, err
	}

	// 从 macOS plist 读取真实数字 uid（DTLastUser），覆盖目录名中提取的十六进制串
	// 钉钉加密密钥派生使用的是数字格式 uid，而非目录名前缀
	if realUID := readDingTalkLastUser(); realUID != "" {
		for i := range results {
			results[i].UserID = realUID
		}
	}

	return results, nil
}

// readDingTalkLastUser 从 macOS 用户偏好设置中读取上次登录的钉钉数字 uid
// 对应 plist key: DTLastUser（值为整数，如 505256109）
func readDingTalkLastUser() string {
	out, err := exec.Command("defaults", "read", "com.alibaba.DingTalkMac", "DTLastUser").Output()
	if err != nil {
		return ""
	}
	uid := strings.TrimSpace(string(out))
	// 验证是纯数字（防止读到非预期值）
	for _, c := range uid {
		if c < '0' || c > '9' {
			return ""
		}
	}
	return uid
}

// findWindows 在 Windows 上查找钉钉数据库
// 路径：%APPDATA%\DingTalk\{uid}_{version}\DBFiles\dingtalk.db
func findWindows() ([]DBLocation, error) {
	appData := os.Getenv("APPDATA")
	if appData == "" {
		return nil, fmt.Errorf("无法获取 APPDATA 环境变量")
	}
	baseDir := filepath.Join(appData, "DingTalk")
	return scanUserDirs(baseDir)
}

// findLinux 在 Linux 上查找钉钉数据库
func findLinux() ([]DBLocation, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	baseDir := filepath.Join(homeDir, ".config", "DingTalk")
	return scanUserDirs(baseDir)
}

// scanUserDirs 扫描基础目录下的用户子目录，查找 dingtalk.db
func scanUserDirs(baseDir string) ([]DBLocation, error) {
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("钉钉数据目录不存在: %s", baseDir)
		}
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	var results []DBLocation
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dirName := entry.Name()
		// 目录格式：{uid}_v2 或 {uid}_{version}
		uid, version := parseUserDir(dirName)
		if uid == "" {
			continue
		}

		dbPath := filepath.Join(baseDir, dirName, "DBFiles", "dingtalk.db")
		if _, err := os.Stat(dbPath); err == nil {
			loc := DBLocation{
				UserID:  uid,
				DBPath:  dbPath,
				Version: version,
			}
			// v3 版本：查找 user_config 文件
			if version == "v3" {
				userConfigPath := filepath.Join(baseDir, dirName, "user_config")
				if _, err := os.Stat(userConfigPath); err == nil {
					loc.UserConfigPath = userConfigPath
				}
			}
			results = append(results, loc)
		}
	}

	return results, nil
}

// parseUserDir 解析用户目录名，提取 UID 和版本
// 支持格式：505256109_v2、505256109_v3、00486e20a9fd1d615052_v3 等
// UID 部分允许纯数字或十六进制字符串（钉钉不同版本目录名格式不同）
func parseUserDir(dirName string) (uid, version string) {
	idx := strings.LastIndex(dirName, "_")
	if idx < 0 {
		return "", ""
	}
	uid = dirName[:idx]
	version = dirName[idx+1:]

	if uid == "" {
		return "", ""
	}

	// version 必须以 'v' 开头（如 v2、v3），避免误匹配无关目录
	if !strings.HasPrefix(version, "v") {
		return "", ""
	}

	// UID 允许十六进制字符（0-9 a-f A-F）
	for _, c := range uid {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return "", ""
		}
	}
	return uid, version
}
