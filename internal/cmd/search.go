// Package cmd 实现 search 子命令
package cmd

import (
"encoding/json"
"fmt"
"os"
"strings"

"dingtalk-cli/internal/database"

"github.com/spf13/cobra"
)

func newSearchCmd() *cobra.Command {
	var (
flagCID   string
flagLimit int
flagJSON  bool
)

	cmd := &cobra.Command{
		Use:   "search <关键词>",
		Short: "搜索聊天消息",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyword := args[0]
			db, err := loadDB()
			if err != nil {
				return err
			}
			query := db.Model(&database.Message{}).
				Where("content_text LIKE ?", "%"+keyword+"%").
				Order("created_at DESC")
			if flagCID != "" {
				query = query.Where("cid = ?", flagCID)
			}
			limit := flagLimit
			if limit <= 0 {
				limit = 50
			}
			query = query.Limit(limit)
			var messages []database.Message
			if err := query.Find(&messages).Error; err != nil {
				return fmt.Errorf("搜索失败: %w", err)
			}
			fillSenderNames(db, messages)
			cidTitles := make(map[string]string)
			for _, msg := range messages {
				if _, ok := cidTitles[msg.CID]; !ok {
					var conv database.Conversation
					if err := db.Where("cid = ?", msg.CID).First(&conv).Error; err == nil {
						cidTitles[msg.CID] = conv.Title
					}
				}
			}
			if flagJSON {
				type SearchResult struct {
					database.Message
					ConvTitle string `json:"conv_title"`
				}
				results := make([]SearchResult, len(messages))
				for i, m := range messages {
					results[i] = SearchResult{Message: m, ConvTitle: cidTitles[m.CID]}
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(results)
			}
			if len(messages) == 0 {
				fmt.Printf("没有找到包含 \"%s\" 的消息\n", keyword)
				return nil
			}
			fmt.Printf("搜索 \"%s\" 找到 %d 条消息:\n\n", keyword, len(messages))
			for _, msg := range messages {
				convTitle := cidTitles[msg.CID]
				if convTitle == "" {
					convTitle = msg.CID
				}
				sender := msg.SenderName
				if sender == "" {
					sender = fmt.Sprintf("用户%d", msg.SenderID)
				}
				content := highlightKeyword(msg.ContentText, keyword)
				fmt.Printf("[%s] %s > %s: %s\n", formatTime(msg.CreatedAt), convTitle, sender, content)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&flagCID, "cid", "", "限定在指定会话内搜索")
	cmd.Flags().IntVar(&flagLimit, "limit", 50, "最多返回条数")
	cmd.Flags().BoolVar(&flagJSON, "json", false, "以 JSON 格式输出")
	return cmd
}

// highlightKeyword 在终端中用 ANSI 粗体高亮关键词
func highlightKeyword(text, keyword string) string {
	if keyword == "" || text == "" {
		return text
	}
	lower := strings.ToLower(text)
	lowerKw := strings.ToLower(keyword)
	idx := strings.Index(lower, lowerKw)
	if idx < 0 {
		return text
	}
	return text[:idx] + "\033[1m" + text[idx:idx+len(keyword)] + "\033[0m" + text[idx+len(keyword):]
}
