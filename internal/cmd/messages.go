// Package cmd 实现 messages 子命令
package cmd

import (
"encoding/json"
"fmt"
"os"
"time"

"dingtalk-cli/internal/database"

"github.com/spf13/cobra"
"gorm.io/gorm"
)

func newMessagesCmd() *cobra.Command {
	var (
flagLimit  int
flagBefore int64
flagJSON   bool
)

	cmd := &cobra.Command{
		Use:   "messages <cid>",
		Short: "读取指定会话的消息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cid := args[0]
			db, err := loadDB()
			if err != nil {
				return err
			}
			var conv database.Conversation
			if err := db.Where("cid = ?", cid).First(&conv).Error; err != nil {
				return fmt.Errorf("会话不存在: %s", cid)
			}
			query := db.Where("cid = ?", cid).Order("created_at DESC")
			if flagBefore > 0 {
				query = query.Where("created_at < ?", flagBefore)
			}
			limit := flagLimit
			if limit <= 0 {
				limit = 50
			}
			query = query.Limit(limit)
			var messages []database.Message
			if err := query.Find(&messages).Error; err != nil {
				return fmt.Errorf("查询消息失败: %w", err)
			}
			for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
				messages[i], messages[j] = messages[j], messages[i]
			}
			fillSenderNames(db, messages)
			if flagJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(messages)
			}
			convType := "群聊"
			if conv.Type == database.ConversationTypeSingle {
				convType = "单聊"
			}
			fmt.Printf("=== %s [%s] (共 %d 条消息) ===\n\n", conv.Title, convType, conv.MessageCount)
			for _, msg := range messages {
				printMessage(msg)
			}
			fmt.Printf("\n显示 %d 条消息\n", len(messages))
			return nil
		},
	}

	cmd.Flags().IntVar(&flagLimit, "limit", 50, "最多显示条数")
	cmd.Flags().Int64Var(&flagBefore, "before", 0, "只显示此时间戳之前的消息（毫秒）")
	cmd.Flags().BoolVar(&flagJSON, "json", false, "以 JSON 格式输出")
	return cmd
}

// printMessage 格式化打印单条消息
func printMessage(msg database.Message) {
	t := time.UnixMilli(msg.CreatedAt).Format("2006-01-02 15:04:05")
	sender := msg.SenderName
	if sender == "" {
		sender = fmt.Sprintf("用户%d", msg.SenderID)
	}
	content := msg.ContentText
	if msg.IsRecall {
		content = "[已撤回]"
	}
	if content == "" {
		content = fmt.Sprintf("[%s]", msg.ContentType.String())
	}
	fmt.Printf("[%s] %s: %s\n", t, sender, content)
}

// fillSenderNames 批量填充消息发送者昵称（供多个子命令复用）
func fillSenderNames(db *gorm.DB, messages []database.Message) {
	idSet := make(map[int64]struct{})
	for _, m := range messages {
		idSet[m.SenderID] = struct{}{}
	}
	ids := make([]int64, 0, len(idSet))
	for id := range idSet {
		ids = append(ids, id)
	}
	var users []database.User
	db.Where("id IN ?", ids).Find(&users)
	userMap := make(map[int64]string, len(users))
	for _, u := range users {
		userMap[u.ID] = u.Nickname
	}
	for i := range messages {
		if name, ok := userMap[messages[i].SenderID]; ok {
			messages[i].SenderName = name
		}
	}
}
