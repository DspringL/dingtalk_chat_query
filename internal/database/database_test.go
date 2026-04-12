// Package database 功能测试 - 使用真实钉钉数据库验证完整查询流程
package database

import (
	"fmt"
	"testing"
	"time"

	"dingtalk-cli/internal/crypto"
	"dingtalk-cli/internal/finder"

	"gorm.io/gorm"
)

// setupRealDB 自动发现并加载本机钉钉数据库，返回迁移后的内存 DB
// 若本机无数据库则跳过测试
func setupRealDB(t *testing.T) *gorm.DB {
	t.Helper()
	locations, err := finder.FindDingTalkDBs()
	if err != nil || len(locations) == 0 {
		t.Skip("本机未找到钉钉数据库，跳过功能测试")
	}
	loc := locations[0]
	fmt.Printf("  使用数据库: UID=%s\n", loc.UserID)
	key := crypto.GenerateKey(loc.UserID)
	tmpPath := t.TempDir() + "/dingtalk_test.db"
	if err := crypto.DecryptDatabase(loc.DBPath, tmpPath, key); err != nil {
		t.Fatalf("解密数据库失败: %v", err)
	}
	db, err := MigrateToMemory(tmpPath)
	if err != nil {
		t.Fatalf("迁移数据库失败: %v", err)
	}
	return db
}

// TestMigrateToMemory 测试数据库迁移到内存全流程
func TestMigrateToMemory(t *testing.T) {
	fmt.Println("=== 测试 MigrateToMemory（完整迁移流程）===")
	db := setupRealDB(t)

	var convCount, userCount, msgCount int64
	db.Model(&Conversation{}).Count(&convCount)
	db.Model(&User{}).Count(&userCount)
	db.Model(&Message{}).Count(&msgCount)

	fmt.Printf("  会话数: %d\n", convCount)
	fmt.Printf("  用户数: %d\n", userCount)
	fmt.Printf("  消息数: %d\n", msgCount)

	if convCount == 0 {
		t.Error("迁移后会话数为 0，数据可能未正确迁移")
	}
	if userCount == 0 {
		t.Error("迁移后用户数为 0")
	}
	fmt.Println("  ✓ 数据库迁移成功")
}

// TestQueryCurrentUser 测试查询当前登录用户
func TestQueryCurrentUser(t *testing.T) {
	fmt.Println("\n=== 测试查询当前用户 ===")
	db := setupRealDB(t)

	var currentUser CurrentUser
	if err := db.First(&currentUser).Error; err != nil {
		t.Logf("  [提示] 未找到当前用户记录（可能是纯群聊账号）: %v", err)
		return
	}

	fmt.Printf("  用户 ID  : %d\n", currentUser.ID)
	fmt.Printf("  昵称     : %s\n", currentUser.Nickname)
	fmt.Printf("  邮箱     : %s\n", currentUser.Email)

	if currentUser.ID == 0 {
		t.Error("当前用户 ID 为 0")
	}
	fmt.Println("  ✓ 当前用户查询成功")
}

// TestQueryConversations 测试查询会话列表及单聊/群聊过滤
func TestQueryConversations(t *testing.T) {
	fmt.Println("\n=== 测试查询会话列表 ===")
	db := setupRealDB(t)

	var conversations []Conversation
	if err := db.Order("last_message_at DESC").Limit(10).Find(&conversations).Error; err != nil {
		t.Fatalf("查询会话失败: %v", err)
	}

	fmt.Printf("  最近 %d 个会话:\n", len(conversations))
	for i, conv := range conversations {
		convType := "群聊"
		if conv.Type == ConversationTypeSingle {
			convType = "单聊"
		}
		topMark := ""
		if conv.IsTop {
			topMark = " [置顶]"
		}
		lastTime := ""
		if conv.LastMessageAt > 0 {
			lastTime = time.UnixMilli(conv.LastMessageAt).Format("2006-01-02 15:04")
		}
		fmt.Printf("    [%d] %s (%s%s) 消息数=%d 最后活跃=%s\n",
			i+1, conv.Title, convType, topMark, conv.MessageCount, lastTime)
	}

	if len(conversations) == 0 {
		t.Error("查询到 0 个会话")
	}

	var singleCount, groupCount int64
	db.Model(&Conversation{}).Where("type = ?", ConversationTypeSingle).Count(&singleCount)
	db.Model(&Conversation{}).Where("type = ?", ConversationTypeGroup).Count(&groupCount)
	fmt.Printf("  单聊: %d 个，群聊: %d 个\n", singleCount, groupCount)
	fmt.Println("  ✓ 会话列表查询成功")
}

// TestQueryMessages 测试查询指定会话的消息
func TestQueryMessages(t *testing.T) {
	fmt.Println("\n=== 测试查询会话消息 ===")
	db := setupRealDB(t)

	// 找消息最多的会话
	var conv Conversation
	if err := db.Order("message_count DESC").First(&conv).Error; err != nil {
		t.Fatalf("查询会话失败: %v", err)
	}
	fmt.Printf("  选取会话: %q (CID=%s, 共 %d 条消息)\n", conv.Title, conv.CID, conv.MessageCount)

	var messages []Message
	if err := db.Where("cid = ?", conv.CID).
		Order("created_at DESC").
		Limit(20).
		Find(&messages).Error; err != nil {
		t.Fatalf("查询消息失败: %v", err)
	}

	fmt.Printf("  最新 %d 条消息:\n", len(messages))
	for i, msg := range messages {
		ts := time.UnixMilli(msg.CreatedAt).Format("01-02 15:04:05")
		content := msg.ContentText
		if msg.IsRecall {
			content = "[已撤回]"
		}
		if len([]rune(content)) > 30 {
			content = string([]rune(content)[:30]) + "..."
		}
		fmt.Printf("    [%d] %s | %-6s | %s\n", i+1, ts, msg.ContentType.String(), content)
	}

	if len(messages) == 0 {
		t.Errorf("会话 %q 查询到 0 条消息", conv.Title)
	}
	fmt.Println("  ✓ 消息查询成功")
}

// TestSearchMessages 测试全局消息搜索
func TestSearchMessages(t *testing.T) {
	fmt.Println("\n=== 测试消息搜索 ===")
	db := setupRealDB(t)

	keywords := []string{"你好", "谢谢", "好的", "OK", "ok"}
	for _, kw := range keywords {
		var count int64
		db.Model(&Message{}).Where("content_text LIKE ?", "%"+kw+"%").Count(&count)
		fmt.Printf("  关键词 %-6q => 匹配 %d 条\n", kw, count)
		if count > 0 {
			var msg Message
			db.Where("content_text LIKE ?", "%"+kw+"%").Order("created_at DESC").First(&msg)
			ts := time.UnixMilli(msg.CreatedAt).Format("2006-01-02 15:04")
			content := msg.ContentText
			if len([]rune(content)) > 40 {
				content = string([]rune(content)[:40]) + "..."
			}
			fmt.Printf("    最新一条: [%s] %s\n", ts, content)
			fmt.Println("  ✓ 搜索功能正常")
			return
		}
	}
	fmt.Println("  [提示] 常见关键词均未命中，数据库内容可能特殊")
}

// TestQueryTopConversations 测试查询置顶会话
func TestQueryTopConversations(t *testing.T) {
	fmt.Println("\n=== 测试查询置顶会话 ===")
	db := setupRealDB(t)

	var topConvs []Conversation
	db.Where("is_top = ?", true).Order("last_message_at DESC").Find(&topConvs)

	if len(topConvs) == 0 {
		fmt.Println("  [提示] 没有置顶会话")
	} else {
		fmt.Printf("  共 %d 个置顶会话:\n", len(topConvs))
		for i, conv := range topConvs {
			fmt.Printf("    [%d] %s (消息数=%d)\n", i+1, conv.Title, conv.MessageCount)
		}
	}
	fmt.Println("  ✓ 置顶会话查询完成")
}

// TestMessageContentTypeStats 测试各消息类型的分布统计
func TestMessageContentTypeStats(t *testing.T) {
	fmt.Println("\n=== 测试消息类型分布统计 ===")
	db := setupRealDB(t)

	types := []MessageContentType{
		MessageContentTypeText,
		MessageContentTypeImage,
		MessageContentTypeDocument,
		MessageContentTypeFile,
		MessageContentTypeVideo,
		MessageContentTypeShortVideo,
		MessageContentTypeSticker,
		MessageContentTypeLink,
		MessageContentTypeShareLink,
		MessageContentTypeCalendar,
		MessageContentTypeVote,
	}

	fmt.Println("  消息类型统计:")
	var totalNonText int64
	for _, mt := range types {
		var count int64
		db.Model(&Message{}).Where("content_type = ?", mt).Count(&count)
		if count > 0 {
			fmt.Printf("    %-10s (type=%-5d): %d 条\n", mt.String(), int(mt), count)
			if mt != MessageContentTypeText {
				totalNonText += count
			}
		}
	}
	fmt.Printf("  非文本消息合计: %d 条\n", totalNonText)
	fmt.Println("  ✓ 消息类型统计完成")
}
