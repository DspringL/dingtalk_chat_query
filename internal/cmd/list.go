// Package cmd 实现 list 子命令
package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"dingtalk-cli/internal/database"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var (
		flagType  string // single / group / top / all
		flagLimit int
		flagJSON  bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "列出会话列表",
		Example: `  dtchat list                   # 列出所有会话（按最后消息时间排序）
  dtchat list --type single     # 只列出单聊
  dtchat list --type group      # 只列出群聊
  dtchat list --type top        # 只列出置顶会话
  dtchat list --limit 20        # 限制显示数量
  dtchat list --json            # JSON 格式输出`,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := loadDB()
			if err != nil {
				return err
			}

			query := db.Model(&database.Conversation{}).Order("last_message_at DESC")

			switch flagType {
			case "single":
				query = query.Where("type = ?", database.ConversationTypeSingle)
			case "group":
				query = query.Where("type = ?", database.ConversationTypeGroup)
			case "top":
				query = query.Where("is_top = ?", true)
			case "all", "":
				// 不过滤
			default:
				return fmt.Errorf("无效的会话类型: %s（可选: single, group, top, all）", flagType)
			}

			if flagLimit > 0 {
				query = query.Limit(flagLimit)
			}

			var conversations []database.Conversation
			if err := query.Find(&conversations).Error; err != nil {
				return fmt.Errorf("查询会话失败: %w", err)
			}

			if flagJSON {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(conversations)
			}

			if len(conversations) == 0 {
				fmt.Println("没有找到会话")
				return nil
			}

			fmt.Printf("%-20s  %-6s  %-8s  %-20s  %s\n",
				"CID", "类型", "消息数", "最后消息时间", "标题")
			fmt.Println(repeatStr("-", 80))

			for _, conv := range conversations {
				convType := "群聊"
				if conv.Type == database.ConversationTypeSingle {
					convType = "单聊"
				}
				if conv.IsTop {
					convType += "⭐"
				}

				lastTime := ""
				if conv.LastMessageAt > 0 {
					t := time.UnixMilli(conv.LastMessageAt)
					lastTime = t.Format("2006-01-02 15:04")
				}

				cid := conv.CID
				if len(cid) > 18 {
					cid = cid[:15] + "..."
				}

				fmt.Printf("%-20s  %-6s  %-8d  %-20s  %s\n",
					cid, convType, conv.MessageCount, lastTime, conv.Title)
			}

			fmt.Printf("\n共 %d 个会话\n", len(conversations))
			return nil
		},
	}

	cmd.Flags().StringVar(&flagType, "type", "all", "会话类型: single/group/top/all")
	cmd.Flags().IntVar(&flagLimit, "limit", 0, "最多显示条数（0 表示不限制）")
	cmd.Flags().BoolVar(&flagJSON, "json", false, "以 JSON 格式输出")
	return cmd
}

func repeatStr(s string, n int) string {
	return strings.Repeat(s, n)
}
