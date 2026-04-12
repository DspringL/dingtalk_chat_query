// Package cmd 实现 export 子命令
package cmd

import (
"encoding/json"
"fmt"
"io"
"os"
"time"

"dingtalk-cli/internal/database"

"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var (
flagFormat string
flagOutput string
)

	cmd := &cobra.Command{
		Use:   "export <cid>",
		Short: "导出会话聊天记录",
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
			var messages []database.Message
			if err := db.Where("cid = ?", cid).Order("created_at ASC").Find(&messages).Error; err != nil {
				return fmt.Errorf("查询消息失败: %w", err)
			}
			fillSenderNames(db, messages)
			out := os.Stdout
			if flagOutput != "" {
				f, err := os.Create(flagOutput)
				if err != nil {
					return fmt.Errorf("创建输出文件失败: %w", err)
				}
				defer f.Close()
				out = f
			}
			switch flagFormat {
			case "json", "":
				return exportJSON(out, conv, messages)
			case "text":
				return exportText(out, conv, messages)
			default:
				return fmt.Errorf("不支持的格式: %s（可选: json, text）", flagFormat)
			}
		},
	}

	cmd.Flags().StringVar(&flagFormat, "format", "json", "导出格式: json / text")
	cmd.Flags().StringVarP(&flagOutput, "output", "o", "", "输出文件路径（默认输出到终端）")
	return cmd
}

func exportJSON(out io.Writer, conv database.Conversation, messages []database.Message) error {
	type ExportData struct {
		Conversation database.Conversation `json:"conversation"`
		Messages     []database.Message    `json:"messages"`
		ExportedAt   string                `json:"exported_at"`
	}
	data := ExportData{
		Conversation: conv,
		Messages:     messages,
		ExportedAt:   time.Now().Format(time.RFC3339),
	}
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

func exportText(out io.Writer, conv database.Conversation, messages []database.Message) error {
	convType := "群聊"
	if conv.Type == database.ConversationTypeSingle {
		convType = "单聊"
	}
	fmt.Fprintf(out, "会话标题: %s\n", conv.Title)
	fmt.Fprintf(out, "会话类型: %s\n", convType)
	fmt.Fprintf(out, "消息总数: %d\n", conv.MessageCount)
	fmt.Fprintf(out, "导出时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Fprintf(out, "%s\n\n", repeatStr("=", 60))
	for _, msg := range messages {
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
		fmt.Fprintf(out, "[%s] %s\n%s\n\n", formatTime(msg.CreatedAt), sender, content)
	}
	return nil
}

// formatTime 格式化毫秒时间戳为可读字符串
func formatTime(ms int64) string {
	if ms <= 0 {
		return ""
	}
	return time.UnixMilli(ms).Format("2006-01-02 15:04:05")
}
