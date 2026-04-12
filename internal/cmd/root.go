// Package cmd 定义 CLI 根命令和全局标志
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

var (
	// 全局标志
	flagDBPath  string // 数据库路径（可选，不指定则自动发现）
	flagUserID  string // 用户 UID（用于解密）
	flagVerbose bool   // 详细输出

	// 全局数据库实例（由 PersistentPreRunE 初始化）
	globalDB *gorm.DB
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "dtchat",
	Short: "钉钉聊天记录读取工具",
	Long: `dtchat - 钉钉聊天记录 CLI 工具

自动发现并读取本机钉钉客户端的聊天记录数据库，
支持查看会话列表、读取消息、搜索内容和导出数据。

示例:
  dtchat info                        # 查看当前用户信息
  dtchat list                        # 列出所有会话
  dtchat list --type single          # 只列出单聊
  dtchat messages <cid>              # 读取指定会话消息
  dtchat search <关键词>              # 全局搜索消息
  dtchat export <cid>                # 导出会话为 JSON
  dtchat export <cid> --format text  # 导出会话为纯文本`,
}

// Execute 执行根命令
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&flagDBPath, "db", "d", "", "钉钉数据库路径（不指定则自动发现）")
	rootCmd.PersistentFlags().StringVarP(&flagUserID, "key", "k", "", "用户 UID（加密数据库解密密钥）")
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "显示详细日志")

	rootCmd.AddCommand(
		newInfoCmd(),
		newListCmd(),
		newMessagesCmd(),
		newSearchCmd(),
		newExportCmd(),
	)
}
