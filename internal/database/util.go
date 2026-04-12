// Package database 提供数据库工具函数
package database

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ParseCID 解析单聊 CID（格式：uid1:uid2）
func ParseCID(cid string) (id1, id2 int64, ok bool) {
	parts := strings.Split(cid, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	id1, err1 := strconv.ParseInt(parts[0], 10, 64)
	id2, err2 := strconv.ParseInt(parts[1], 10, 64)
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return id1, id2, true
}

// GetOtherUserID 从单聊 CID 中获取对方用户 ID
func GetOtherUserID(cid string, currentUserID int64) (int64, error) {
	id1, id2, ok := ParseCID(cid)
	if !ok {
		return 0, fmt.Errorf("无效的 CID 格式: %s", cid)
	}
	if id1 == currentUserID {
		return id2, nil
	}
	return id1, nil
}

// FindMostFrequentID 找出出现次数最多的 ID（用于推断当前用户）
func FindMostFrequentID(idCounts map[int64]int) int64 {
	var mostFrequentID int64
	maxCount := 0
	for id, count := range idCounts {
		if count > maxCount {
			maxCount = count
			mostFrequentID = id
		}
	}
	return mostFrequentID
}

// ExtractContentText 从消息 JSON 中提取可读文本
func ExtractContentText(contentType MessageContentType, contentJson string) string {
	var content map[string]any
	if err := json.Unmarshal([]byte(contentJson), &content); err != nil {
		return ""
	}

	switch contentType {
	case MessageContentTypeText:
		if text, ok := content["text"].(string); ok {
			return text
		}
	case MessageContentTypeImage:
		if filename, ok := content["filename"].(string); ok {
			return "[图片] " + filename
		}
		return "[图片]"
	case MessageContentTypeDocument:
		if filename, ok := content["filename"].(string); ok {
			return "[文件] " + filename
		}
		return "[文件]"
	case MessageContentTypeShareLink:
		return extractAttachmentURL(content, "[链接]")
	case MessageContentTypeLink:
		return extractFromAttachments(content, "b_tl", "[链接]")
	case MessageContentTypeFileOld, MessageContentTypeFile:
		return extractFromAttachments(content, "f_name", "[文件]")
	case MessageContentTypeFolder:
		return extractFromAttachments(content, "f_name", "[文件夹]")
	case MessageContentTypeSticker:
		return "[表情]"
	case MessageContentTypeCard:
		return "[名片]"
	case MessageContentTypeVideo:
		return extractFromAttachments(content, "title", "[视频]")
	case MessageContentTypeShortVideo:
		return "[短视频]"
	case MessageContentTypeVideoCall:
		return extractFromAttachments(content, "title", "[视频通话]")
	case MessageContentTypeCalendar:
		return "[日程]"
	case MessageContentTypeVote:
		return "[投票]"
	case MessageContentTypeRobot:
		return extractI18nMessage(content, "interactiveCardLastMessage", "LastMessageI18n", "zh_CN", "[群公告]")
	case MessageContentTypeActionCard:
		return extractI18nMessage(content, "interactiveCardLastMessage", "LastMessageI18n", "zh_CN", "[互动卡片]")
	case MessageContentTypeMiniProgram:
		return extractFromAttachments(content, "desc", "[小程序]")
	}
	return ""
}

func extractAttachmentURL(content map[string]any, fallback string) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		return fallback
	}
	att, ok := attachments[0].(map[string]any)
	if !ok {
		return fallback
	}
	if url, ok := att["url"].(string); ok && url != "" {
		return url
	}
	return fallback
}

func extractFromAttachments(content map[string]any, field, fallback string) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		return fallback
	}
	att, ok := attachments[0].(map[string]any)
	if !ok {
		return fallback
	}
	extStr, ok := att["extension"].(string)
	if !ok {
		return fallback
	}
	var ext map[string]any
	if err := json.Unmarshal([]byte(extStr), &ext); err != nil {
		return fallback
	}
	if val, ok := ext[field].(string); ok && val != "" {
		return val
	}
	return fallback
}

func extractI18nMessage(content map[string]any, primaryField, i18nField, locale, fallback string) string {
	attachments, ok := content["attachments"].([]any)
	if !ok || len(attachments) == 0 {
		return fallback
	}
	att, ok := attachments[0].(map[string]any)
	if !ok {
		return fallback
	}
	extStr, ok := att["extension"].(string)
	if !ok {
		return fallback
	}
	var ext map[string]any
	if err := json.Unmarshal([]byte(extStr), &ext); err != nil {
		return fallback
	}
	if val, ok := ext[primaryField].(string); ok && val != "" {
		return val
	}
	i18nStr, ok := ext[i18nField].(string)
	if !ok {
		return fallback
	}
	var i18n map[string]any
	if err := json.Unmarshal([]byte(i18nStr), &i18n); err != nil {
		return fallback
	}
	if val, ok := i18n[locale].(string); ok && val != "" {
		return val
	}
	return fallback
}
