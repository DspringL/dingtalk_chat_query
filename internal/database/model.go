// Package database 定义数据模型与消息类型常量
package database

// ConversationType 会话类型
type ConversationType int

// MessageContentType 消息内容类型
type MessageContentType int

const (
	ConversationTypeSingle ConversationType = 1 // 单聊
	ConversationTypeGroup  ConversationType = 2 // 群聊
)

const (
	MessageContentTypeText        MessageContentType = 1    // 文本消息
	MessageContentTypeImage       MessageContentType = 2    // 图片消息
	MessageContentTypeDocument    MessageContentType = 4    // 文档消息
	MessageContentTypeShareLink   MessageContentType = 102  // 分享链接
	MessageContentTypeLocation    MessageContentType = 202  // 位置消息
	MessageContentTypeLink        MessageContentType = 300  // 链接消息
	MessageContentTypeFileOld     MessageContentType = 500  // 文件消息（旧格式）
	MessageContentTypeFile        MessageContentType = 501  // 文件消息
	MessageContentTypeFolder      MessageContentType = 503  // 文件夹消息
	MessageContentTypeSticker     MessageContentType = 901  // 表情贴纸
	MessageContentTypeCard        MessageContentType = 1101 // 名片消息
	MessageContentTypeVideo       MessageContentType = 1200 // 视频消息
	MessageContentTypeShortVideo  MessageContentType = 1201 // 短视频
	MessageContentTypeVideoCall   MessageContentType = 1202 // 视频通话
	MessageContentTypeCalendar    MessageContentType = 1500 // 日程消息
	MessageContentTypeVote        MessageContentType = 1600 // 投票消息
	MessageContentTypeRobot       MessageContentType = 2900 // 机器人消息
	MessageContentTypeActionCard  MessageContentType = 2950 // 互动卡片
	MessageContentTypeMiniProgram MessageContentType = 3100 // 小程序消息
)

// String 返回消息类型的中文描述
func (t MessageContentType) String() string {
	switch t {
	case MessageContentTypeText:
		return "文本"
	case MessageContentTypeImage:
		return "图片"
	case MessageContentTypeDocument:
		return "文档"
	case MessageContentTypeShareLink:
		return "分享链接"
	case MessageContentTypeLocation:
		return "位置"
	case MessageContentTypeLink:
		return "链接"
	case MessageContentTypeFileOld, MessageContentTypeFile:
		return "文件"
	case MessageContentTypeFolder:
		return "文件夹"
	case MessageContentTypeSticker:
		return "表情"
	case MessageContentTypeCard:
		return "名片"
	case MessageContentTypeVideo:
		return "视频"
	case MessageContentTypeShortVideo:
		return "短视频"
	case MessageContentTypeVideoCall:
		return "视频通话"
	case MessageContentTypeCalendar:
		return "日程"
	case MessageContentTypeVote:
		return "投票"
	case MessageContentTypeRobot:
		return "机器人"
	case MessageContentTypeActionCard:
		return "互动卡片"
	case MessageContentTypeMiniProgram:
		return "小程序"
	default:
		return "未知"
	}
}

// Conversation 会话记录（对应 tbconversation）
type Conversation struct {
	ID                 int64            `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CID                string           `gorm:"column:cid;index" json:"cid"`
	Type               ConversationType `gorm:"column:type;index" json:"type"`
	Title              string           `gorm:"column:title" json:"title"`
	IsTop              bool             `gorm:"column:is_top;index" json:"is_top"`
	MessageCount       int64            `gorm:"column:message_count" json:"message_count"`
	LastMessageAt      int64            `gorm:"column:last_message_at;index" json:"last_message_at"`
	LastMessageID      int64            `gorm:"column:last_message_id" json:"last_message_id"`
	LastMessagePreview string           `gorm:"column:last_message_preview" json:"last_message_preview"`
	CreatedAt          int64            `gorm:"column:created_at" json:"created_at"`
}

// User 用户信息（对应 tbuser_profile_v2）
type User struct {
	ID       int64  `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Nickname string `gorm:"column:nickname;index" json:"nickname"`
	Email    string `gorm:"column:email;index" json:"email"`
}

// CurrentUser 当前登录用户（单例）
type CurrentUser struct {
	ID       int64  `gorm:"primaryKey;column:id" json:"id"`
	Nickname string `gorm:"column:nickname" json:"nickname"`
	Email    string `gorm:"column:email" json:"email"`
}

// Message 消息记录（对应 tbmsg_***）
type Message struct {
	ID          int64              `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	CID         string             `gorm:"column:cid;index" json:"cid"`
	OriginalCID string             `gorm:"column:original_cid" json:"original_cid"`
	SenderID    int64              `gorm:"column:sender_id;index" json:"sender_id"`
	SenderName  string             `gorm:"-" json:"sender_name"`
	ContentType MessageContentType `gorm:"column:content_type" json:"content_type"`
	ContentText string             `gorm:"column:content_text" json:"content_text"`
	ContentJson string             `gorm:"column:content_json" json:"content_json"`
	CreatedAt   int64              `gorm:"column:created_at;index" json:"created_at"`
	IsRecall    bool               `gorm:"column:is_recall" json:"is_recall"`
}
