# dtchat - 钉钉聊天记录 CLI 工具

跨平台命令行工具，自动发现并读取本机钉钉客户端的聊天记录数据库。

## 功能

- **自动发现**：无需手动指定路径，自动定位钉钉数据库
- **自动解密**：支持 v2 加密数据库（AES-ECB）
- **查看会话**：列出单聊、群聊、置顶会话
- **读取消息**：查看指定会话的聊天记录
- **全局搜索**：跨会话关键词搜索
- **导出数据**：导出为 JSON 或纯文本

## 安装

```bash
go build -o dtchat .
```

或使用构建脚本生成多平台二进制：

```bash
./build.sh
```

## 使用

```bash
# 查看当前用户信息
dtchat info

# 列出所有会话
dtchat list

# 只列单聊 / 群聊 / 置顶
dtchat list --type single
dtchat list --type group
dtchat list --type top

# 读取指定会话消息（最新 50 条）
dtchat messages <cid>
dtchat messages <cid> --limit 100

# 全局搜索
dtchat search 关键词
dtchat search 关键词 --cid <cid>   # 限定会话内搜索

# 导出会话
dtchat export <cid>                    # JSON 格式
dtchat export <cid> --format text      # 纯文本
dtchat export <cid> -o chat.json       # 保存到文件

# 手动指定数据库路径（加密文件需要 -k 指定 UID）
dtchat info -d /path/to/dingtalk.db
dtchat info -d /path/to/encrypted.db -k 505256109
```

## 数据库路径

| 平台 | 路径 |
|------|------|
| macOS | `~/Library/Containers/5ZSL2CJU2T.com.dingtalk.mac/Data/Library/Application Support/DingTalkMac/{uid}_v2/DBFiles/dingtalk.db` |
| Windows | `%APPDATA%\DingTalk\{uid}_{version}\DBFiles\dingtalk.db` |

## 支持平台

- macOS (amd64 / arm64)
- Linux (amd64 / arm64)
- Windows (amd64)

## 发布流程

### 配置推送仓库

将 GitHub 仓库追加为 origin 的推送地址（只推不拉）：

```bash
git remote set-url --add --push origin git@github.com:DspringL/dingtalk_chat_query.git
```

配置后 `git push origin` 会同时推送到 GitLab 和 GitHub，`git pull` 仍只从 GitLab 拉取。

### 发布新版本

打 tag 并推送，自动触发 GitHub Actions 构建所有平台二进制并发布到 Releases：

```bash
git tag v1.0.0
git push origin v1.0.0
```

### 删除 tag

```bash
# 删除本地 tag
git tag -d v1.0.0

# 删除远程 tag
git push origin :refs/tags/v1.0.0
```
