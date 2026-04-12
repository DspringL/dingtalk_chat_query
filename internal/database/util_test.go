// Package database 的单元测试 - 演示工具函数调用
package database

import (
	"fmt"
	"testing"
)

// TestParseCID 测试单聊 CID 解析
func TestParseCID(t *testing.T) {
	cases := []struct {
		cid         string
		expectOK    bool
		expectID1   int64
		expectID2   int64
		description string
	}{
		{"123456:789012", true, 123456, 789012, "标准格式"},
		{"505256109:678901234", true, 505256109, 678901234, "真实 UID 格式"},
		{"abc:def", false, 0, 0, "非数字格式"},
		{"123456", false, 0, 0, "缺少分隔符"},
		{"", false, 0, 0, "空字符串"},
		{"1:2:3", false, 0, 0, "多个分隔符"},
	}

	fmt.Println("=== 测试 ParseCID ===")
	for _, c := range cases {
		id1, id2, ok := ParseCID(c.cid)
		status := "✓"
		if ok != c.expectOK {
			status = "✗"
			t.Errorf("[%s] CID=%q: 期望 ok=%v，实际 ok=%v", c.description, c.cid, c.expectOK, ok)
		}
		fmt.Printf("  %s CID=%-25q => id1=%-12d id2=%-12d ok=%v  (%s)\n",
			status, c.cid, id1, id2, ok, c.description)
	}
}

// TestGetOtherUserID 测试从单聊 CID 获取对方用户 ID
func TestGetOtherUserID(t *testing.T) {
	fmt.Println("\n=== 测试 GetOtherUserID ===")
	cases := []struct {
		cid           string
		currentUserID int64
		expectOtherID int64
		expectErr     bool
		description   string
	}{
		{"100:200", 100, 200, false, "当前用户是 id1"},
		{"100:200", 200, 100, false, "当前用户是 id2"},
		{"invalid", 100, 0, true, "无效 CID"},
	}

	for _, c := range cases {
		otherID, err := GetOtherUserID(c.cid, c.currentUserID)
		if c.expectErr {
			fmt.Printf("  CID=%-12q currentUser=%-5d => 错误: %v\n", c.cid, c.currentUserID, err)
			if err == nil {
				t.Errorf("[%s] 期望返回错误，但实际返回 nil", c.description)
			}
		} else {
			fmt.Printf("  CID=%-12q currentUser=%-5d => otherUser=%d\n", c.cid, c.currentUserID, otherID)
			if err != nil {
				t.Errorf("[%s] 意外错误: %v", c.description, err)
			}
			if otherID != c.expectOtherID {
				t.Errorf("[%s] 期望 otherID=%d，实际 %d", c.description, c.expectOtherID, otherID)
			}
		}
	}
}

// TestFindMostFrequentID 测试找出出现次数最多的 ID
func TestFindMostFrequentID(t *testing.T) {
	fmt.Println("\n=== 测试 FindMostFrequentID ===")
	cases := []struct {
		idCounts    map[int64]int
		expectID    int64
		description string
	}{
		{
			map[int64]int{100: 5, 200: 3, 300: 1},
			100,
			"ID=100 出现最多",
		},
		{
			map[int64]int{505256109: 10, 678901234: 7},
			505256109,
			"真实 UID 场景",
		},
		{
			map[int64]int{},
			0,
			"空 map 返回 0",
		},
	}

	for _, c := range cases {
		result := FindMostFrequentID(c.idCounts)
		status := "✓"
		if result != c.expectID {
			status = "✗"
			t.Errorf("[%s] 期望 ID=%d，实际 %d", c.description, c.expectID, result)
		}
		fmt.Printf("  %s %-30s => 最高频 ID=%d\n", status, c.description, result)
	}
}

// TestExtractContentText 测试从各种消息类型中提取可读文本
func TestExtractContentText(t *testing.T) {
	fmt.Println("\n=== 测试 ExtractContentText ===")
	cases := []struct {
		contentType MessageContentType
		contentJson string
		expectText  string
		description string
	}{
		{
			MessageContentTypeText,
			`{"text":"你好，世界！"}`,
			"你好，世界！",
			"文本消息",
		},
		{
			MessageContentTypeImage,
			`{"filename":"photo.jpg"}`,
			"[图片] photo.jpg",
			"图片消息（含文件名）",
		},
		{
			MessageContentTypeImage,
			`{}`,
			"[图片]",
			"图片消息（无文件名）",
		},
		{
			MessageContentTypeDocument,
			`{"filename":"report.pdf"}`,
			"[文件] report.pdf",
			"文档消息",
		},
		{
			MessageContentTypeSticker,
			`{}`,
			"[表情]",
			"表情贴纸",
		},
		{
			MessageContentTypeCard,
			`{}`,
			"[名片]",
			"名片消息",
		},
		{
			MessageContentTypeShortVideo,
			`{}`,
			"[短视频]",
			"短视频消息",
		},
		{
			MessageContentTypeCalendar,
			`{}`,
			"[日程]",
			"日程消息",
		},
		{
			MessageContentTypeVote,
			`{}`,
			"[投票]",
			"投票消息",
		},
		{
			MessageContentTypeText,
			`invalid-json`,
			"",
			"无效 JSON 返回空字符串",
		},
	}

	for _, c := range cases {
		result := ExtractContentText(c.contentType, c.contentJson)
		status := "✓"
		if result != c.expectText {
			status = "✗"
			t.Errorf("[%s] 期望=%q，实际=%q", c.description, c.expectText, result)
		}
		fmt.Printf("  %s %-20s => %q\n", status, c.description, result)
	}
}

// TestMessageContentTypeString 测试消息类型的中文描述
func TestMessageContentTypeString(t *testing.T) {
	fmt.Println("\n=== 测试 MessageContentType.String() ===")
	types := []MessageContentType{
		MessageContentTypeText,
		MessageContentTypeImage,
		MessageContentTypeDocument,
		MessageContentTypeShareLink,
		MessageContentTypeLocation,
		MessageContentTypeLink,
		MessageContentTypeFile,
		MessageContentTypeSticker,
		MessageContentTypeCard,
		MessageContentTypeVideo,
		MessageContentTypeShortVideo,
		MessageContentTypeVideoCall,
		MessageContentTypeCalendar,
		MessageContentTypeVote,
		MessageContentTypeRobot,
		MessageContentTypeActionCard,
		MessageContentTypeMiniProgram,
		MessageContentType(9999), // 未知类型
	}

	for _, mt := range types {
		fmt.Printf("  类型值=%-5d => %s\n", int(mt), mt.String())
	}
}
