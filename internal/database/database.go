// Package database 提供钉钉数据库迁移与查询功能
package database

import (
	"database/sql"
	"fmt"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// MigrateToMemory 将钉钉原始数据库迁移到内存数据库，便于查询
func MigrateToMemory(dbPath string) (*gorm.DB, error) {
	db, err := openRawDB(dbPath)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	memDB, err := createMemoryDB()
	if err != nil {
		return nil, err
	}

	if err := migrateUsers(db, memDB); err != nil {
		return nil, fmt.Errorf("迁移用户数据失败: %w", err)
	}
	if err := migrateConversations(db, memDB); err != nil {
		return nil, fmt.Errorf("迁移会话数据失败: %w", err)
	}
	if err := migrateMessages(db, memDB); err != nil {
		return nil, fmt.Errorf("迁移消息数据失败: %w", err)
	}
	if err := updateContentText(memDB); err != nil {
		return nil, fmt.Errorf("更新消息文本失败: %w", err)
	}
	if err := updateSingleChatTitles(memDB); err != nil {
		return nil, fmt.Errorf("更新单聊标题失败: %w", err)
	}
	if err := saveCurrentUser(memDB); err != nil {
		return nil, fmt.Errorf("保存当前用户失败: %w", err)
	}
	if err := updateConversationStats(memDB); err != nil {
		return nil, fmt.Errorf("更新会话统计失败: %w", err)
	}

	return memDB, nil
}

// ValidateDB 验证数据库文件是否可以正常打开
func ValidateDB(dbPath string) error {
	db, err := openRawDB(dbPath)
	if err != nil {
		return err
	}
	db.Close()
	return nil
}

func openRawDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("打开数据库失败: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("数据库连接失败: %w", err)
	}
	return db, nil
}

func createMemoryDB() (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("创建内存数据库失败: %w", err)
	}
	if err := db.AutoMigrate(&Conversation{}, &User{}, &CurrentUser{}, &Message{}); err != nil {
		return nil, fmt.Errorf("初始化数据库结构失败: %w", err)
	}
	return db, nil
}

func migrateUsers(srcDB *sql.DB, destDB *gorm.DB) error {
	rows, err := srcDB.Query("SELECT uid, nick, email FROM tbuser_profile_v2")
	if err != nil {
		return err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.ID, &u.Nickname, &u.Email); err != nil {
			return err
		}
		users = append(users, u)
	}
	if len(users) == 0 {
		return nil
	}
	return destDB.Create(&users).Error
}

func migrateConversations(srcDB *sql.DB, destDB *gorm.DB) error {
	rows, err := srcDB.Query("SELECT cid, type, title, top, lastMid, createAt FROM tbconversation")
	if err != nil {
		return err
	}
	defer rows.Close()

	var conversations []Conversation
	for rows.Next() {
		var conv Conversation
		var top int
		if err := rows.Scan(&conv.CID, &conv.Type, &conv.Title, &top, &conv.LastMessageID, &conv.CreatedAt); err != nil {
			return err
		}
		conv.IsTop = top > 0
		conversations = append(conversations, conv)
	}
	if len(conversations) == 0 {
		return nil
	}
	return destDB.Create(&conversations).Error
}

func migrateMessages(srcDB *sql.DB, destDB *gorm.DB) error {
	rows, err := srcDB.Query("SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'tbmsg%'")
	if err != nil {
		return err
	}
	defer rows.Close()

	var tableNames []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return err
		}
		tableNames = append(tableNames, t)
	}

	const batchSize = 1000
	for _, tableName := range tableNames {
		offset := 0
		for {
			query := fmt.Sprintf(
				"SELECT mid, cid, senderId, contentType, content, createdAt, recallStatus FROM %s LIMIT %d OFFSET %d",
				tableName, batchSize, offset,
			)
			msgRows, err := srcDB.Query(query)
			if err != nil {
				return err
			}

			var batch []Message
			for msgRows.Next() {
				var msg Message
				var recallStatus int
				if err := msgRows.Scan(&msg.ID, &msg.CID, &msg.SenderID, &msg.ContentType, &msg.ContentJson, &msg.CreatedAt, &recallStatus); err != nil {
					_ = msgRows.Close()
					return err
				}
				msg.IsRecall = recallStatus > 0
				batch = append(batch, msg)
			}
			_ = msgRows.Close()

			if len(batch) == 0 {
				break
			}
			if err := destDB.Create(&batch).Error; err != nil {
				return err
			}
			offset += batchSize
		}
	}
	return nil
}

func updateContentText(db *gorm.DB) error {
	const batchSize = 500
	offset := 0
	for {
		var messages []Message
		if err := db.Limit(batchSize).Offset(offset).Find(&messages).Error; err != nil {
			return err
		}
		if len(messages) == 0 {
			break
		}
		for i := range messages {
			messages[i].ContentText = ExtractContentText(messages[i].ContentType, messages[i].ContentJson)
		}
		if err := db.Save(&messages).Error; err != nil {
			return err
		}
		offset += batchSize
	}
	return nil
}

func updateSingleChatTitles(db *gorm.DB) error {
	currentUserID, err := GetCurrentUserID(db)
	if err != nil {
		return err
	}

	var conversations []Conversation
	if err := db.Where("type = ?", ConversationTypeSingle).Find(&conversations).Error; err != nil {
		return err
	}

	for i := range conversations {
		otherUserID, err := GetOtherUserID(conversations[i].CID, currentUserID)
		if err != nil {
			continue
		}
		var user User
		if err := db.First(&user, otherUserID).Error; err == nil {
			conversations[i].Title = user.Nickname
		}
	}
	return db.Save(&conversations).Error
}

// GetCurrentUserID 通过单聊 CID 频率推断当前用户 ID
func GetCurrentUserID(db *gorm.DB) (int64, error) {
	var conversations []Conversation
	if err := db.Where("type = ?", ConversationTypeSingle).Find(&conversations).Error; err != nil {
		return 0, err
	}

	idCount := make(map[int64]int)
	for _, conv := range conversations {
		id1, id2, ok := ParseCID(conv.CID)
		if ok {
			idCount[id1]++
			idCount[id2]++
		}
	}
	return FindMostFrequentID(idCount), nil
}

func saveCurrentUser(db *gorm.DB) error {
	currentUserID, err := GetCurrentUserID(db)
	if err != nil {
		return err
	}
	var user User
	if err := db.First(&user, currentUserID).Error; err != nil {
		return err
	}
	return db.Create(&CurrentUser{
		ID:       user.ID,
		Nickname: user.Nickname,
		Email:    user.Email,
	}).Error
}

func updateConversationStats(db *gorm.DB) error {
	var conversations []Conversation
	if err := db.Find(&conversations).Error; err != nil {
		return err
	}

	for i := range conversations {
		var count int64
		db.Model(&Message{}).Where("cid = ?", conversations[i].CID).Count(&count)
		conversations[i].MessageCount = count

		var lastMsg Message
		if err := db.Where("cid = ?", conversations[i].CID).Order("created_at DESC").First(&lastMsg).Error; err == nil {
			conversations[i].LastMessageAt = lastMsg.CreatedAt
			conversations[i].LastMessagePreview = lastMsg.ContentText
		}
	}
	return db.Save(&conversations).Error
}
