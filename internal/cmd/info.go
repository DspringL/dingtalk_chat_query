// Package cmd 实现 info 子命令
package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"dingtalk-cli/internal/database"

	"github.com/spf13/cobra"
)

func newInfoCmd() *cobra.Command {
	var flagJSON bool

	cmd := &cobra.Command{
		Use:   "info",
		Short: "显示当前登录用户信息",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := loadDB()
			if err != nil {
				return err
			}

			var currentUser database.CurrentUser
			if err := db.First(&currentUser).Error; err != nil {
				return fmt.Errorf("获取用户信息失败: %w", err)
			}

			var totalConv, totalMsg int64
			db.Model(&database.Conversation{}).Count(&totalConv)
			db.Model(&database.Message{}).Count(&totalMsg)

			if flagJSON {
				out := map[string]any{
					"id":                 currentUser.ID,
					"nickname":           currentUser.Nickname,
					"email":              currentUser.Email,
					"total_conversations": totalConv,
					"total_messages":     totalMsg,
				}
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(out)
			}

			fmt.Printf("用户 ID   : %d\n", currentUser.ID)
			fmt.Printf("昵称      : %s\n", currentUser.Nickname)
			fmt.Printf("邮箱      : %s\n", currentUser.Email)
			fmt.Printf("会话总数  : %d\n", totalConv)
			fmt.Printf("消息总数  : %d\n", totalMsg)
			return nil
		},
	}

	cmd.Flags().BoolVar(&flagJSON, "json", false, "以 JSON 格式输出")
	return cmd
}
