package handlers

import (
	"TGFaqBot/database"
	"fmt"
	"sync"
	"time"
)

// OperationType 操作类型
type OperationType string

const (
	OpAdd         OperationType = "add"
	OpUpdate      OperationType = "update"
	OpDelete      OperationType = "delete"
	OpBatchDelete OperationType = "batch_delete"
)

// HistoryEntry 历史记录条目
type HistoryEntry struct {
	ID        string         `json:"id"`
	UserID    int64          `json:"user_id"`
	ChatID    int64          `json:"chat_id"`
	Operation OperationType  `json:"operation"`
	Timestamp time.Time      `json:"timestamp"`
	Details   HistoryDetails `json:"details"`
}

// HistoryDetails 操作详情
type HistoryDetails struct {
	Key       string `json:"key,omitempty"`
	OldValue  string `json:"old_value,omitempty"`
	NewValue  string `json:"new_value,omitempty"`
	MatchType int    `json:"match_type,omitempty"`
	Count     int    `json:"count,omitempty"` // 用于批量操作
}

// HistoryManager 历史管理器
type HistoryManager struct {
	history map[int64][]HistoryEntry // chatID -> 历史记录
	mutex   sync.RWMutex
	maxSize int // 每个聊天的最大历史记录数
}

// NewHistoryManager 创建历史管理器
func NewHistoryManager() *HistoryManager {
	return &HistoryManager{
		history: make(map[int64][]HistoryEntry),
		maxSize: 10, // 保留最近10个操作
	}
}

// AddEntry 添加历史记录
func (h *HistoryManager) AddEntry(userID, chatID int64, operation OperationType, details HistoryDetails) string {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	entryID := generateID()
	entry := HistoryEntry{
		ID:        entryID,
		UserID:    userID,
		ChatID:    chatID,
		Operation: operation,
		Timestamp: time.Now(),
		Details:   details,
	}

	chatHistory := h.history[chatID]
	chatHistory = append(chatHistory, entry)

	// 限制历史记录大小
	if len(chatHistory) > h.maxSize {
		chatHistory = chatHistory[len(chatHistory)-h.maxSize:]
	}

	h.history[chatID] = chatHistory
	return entryID
}

// GetHistory 获取聊天的历史记录
func (h *HistoryManager) GetHistory(chatID int64) []HistoryEntry {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	chatHistory := h.history[chatID]
	result := make([]HistoryEntry, len(chatHistory))
	copy(result, chatHistory)
	return result
}

// GetLastEntry 获取最后一个操作
func (h *HistoryManager) GetLastEntry(chatID int64) *HistoryEntry {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	chatHistory := h.history[chatID]
	if len(chatHistory) == 0 {
		return nil
	}

	// 返回副本
	entry := chatHistory[len(chatHistory)-1]
	return &entry
}

// FindEntry 根据ID查找记录
func (h *HistoryManager) FindEntry(chatID int64, entryID string) *HistoryEntry {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	chatHistory := h.history[chatID]
	for i := len(chatHistory) - 1; i >= 0; i-- {
		if chatHistory[i].ID == entryID {
			entry := chatHistory[i]
			return &entry
		}
	}
	return nil
}

// generateID 生成简单的ID
func generateID() string {
	return time.Now().Format("20060102150405") + "_" + time.Now().Format("000")
}

// CanUndo 检查是否可以撤销
func (h *HistoryManager) CanUndo(chatID int64, entryID string) bool {
	entry := h.FindEntry(chatID, entryID)
	if entry == nil {
		return false
	}

	// 只允许撤销最近5分钟内的操作
	return time.Since(entry.Timestamp) <= 5*time.Minute
}

// UndoOperation 撤销操作
func (h *HistoryManager) UndoOperation(db database.Database, chatID int64, entryID string) error {
	entry := h.FindEntry(chatID, entryID)
	if entry == nil {
		return fmt.Errorf("历史记录不存在")
	}

	if !h.CanUndo(chatID, entryID) {
		return fmt.Errorf("操作超时，无法撤销")
	}

	switch entry.Operation {
	case OpAdd:
		// 撤销添加 = 删除
		matchTypeValue, err := database.MatchTypeFromInt(entry.Details.MatchType)
		if err != nil {
			return fmt.Errorf("匹配类型转换错误: %v", err)
		}
		return db.DeleteEntry(entry.Details.Key, matchTypeValue)

	case OpUpdate:
		// 撤销更新 = 恢复旧值
		matchTypeValue, err := database.MatchTypeFromInt(entry.Details.MatchType)
		if err != nil {
			return fmt.Errorf("匹配类型转换错误: %v", err)
		}
		return db.UpdateEntry(entry.Details.Key, matchTypeValue, matchTypeValue, entry.Details.OldValue)

	case OpDelete:
		// 撤销删除 = 重新添加
		matchTypeValue, err := database.MatchTypeFromInt(entry.Details.MatchType)
		if err != nil {
			return fmt.Errorf("匹配类型转换错误: %v", err)
		}
		return db.AddEntry(entry.Details.Key, matchTypeValue, entry.Details.OldValue)

	case OpBatchDelete:
		// 批量删除无法撤销
		return fmt.Errorf("批量删除操作无法撤销")

	default:
		return fmt.Errorf("未知操作类型")
	}
}
