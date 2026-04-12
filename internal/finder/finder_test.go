// Package finder 的单元测试 - 演示路径解析与数据库发现功能
package finder

import (
	"fmt"
	"testing"
)

// TestParseUserDir 测试用户目录名解析
func TestParseUserDir(t *testing.T) {
	cases := []struct {
		dirName     string
		expectUID   string
		expectVer   string
		description string
	}{
		{"505256109_v2", "505256109", "v2", "标准 v2 格式"},
		{"678901234_v3", "678901234", "v3", "v3 版本"},
		{"123_v2", "123", "v2", "短 UID"},
		{"abc_v2", "", "", "非数字 UID"},
		{"505256109", "", "", "无版本后缀"},
		{"", "", "", "空字符串"},
		{"_v2", "", "v2", "空 UID 部分（UID 为空则忽略）"},
	}

	fmt.Println("=== 测试 parseUserDir ===")
	for _, c := range cases {
		uid, ver := parseUserDir(c.dirName)
		status := "✓"
		if uid != c.expectUID || ver != c.expectVer {
			status = "✗"
			t.Errorf("[%s] dirName=%q: 期望 uid=%q ver=%q，实际 uid=%q ver=%q",
				c.description, c.dirName, c.expectUID, c.expectVer, uid, ver)
		}
		fmt.Printf("  %s 目录=%-20q => uid=%-12q version=%q  (%s)\n",
			status, c.dirName, uid, ver, c.description)
	}
}

// TestFindDingTalkDBs 测试自动发现钉钉数据库（当前机器上可能无数据库，验证错误处理）
func TestFindDingTalkDBs(t *testing.T) {
	fmt.Println("\n=== 测试 FindDingTalkDBs ===")
	dbs, err := FindDingTalkDBs()
	if err != nil {
		fmt.Printf("  未找到钉钉数据库（正常情况）: %v\n", err)
		return
	}

	if len(dbs) == 0 {
		fmt.Println("  未发现任何钉钉数据库文件")
		return
	}

	fmt.Printf("  发现 %d 个钉钉数据库:\n", len(dbs))
	for i, db := range dbs {
		fmt.Printf("    [%d] UID=%-15s Version=%-5s Path=%s\n",
			i+1, db.UserID, db.Version, db.DBPath)
	}
}

// TestDBLocationFields 测试 DBLocation 结构体字段赋值与读取
func TestDBLocationFields(t *testing.T) {
	fmt.Println("\n=== 测试 DBLocation 结构体 ===")
	loc := DBLocation{
		UserID:  "505256109",
		DBPath:  "/Users/test/Library/DingTalk/505256109_v2/DBFiles/dingtalk.db",
		Version: "v2",
	}

	fmt.Printf("  UserID  : %s\n", loc.UserID)
	fmt.Printf("  Version : %s\n", loc.Version)
	fmt.Printf("  DBPath  : %s\n", loc.DBPath)

	if loc.UserID != "505256109" {
		t.Errorf("UserID 期望 505256109，实际 %s", loc.UserID)
	}
	if loc.Version != "v2" {
		t.Errorf("Version 期望 v2，实际 %s", loc.Version)
	}
	fmt.Println("  ✓ 结构体字段读写正常")
}
