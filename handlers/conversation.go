package handlers

import (
	"sync"
	"time"
)

// Conversation 对话状态结构
type Conversation struct {
	Stage     string
	EntryID   int
	NewType   int
	OldType   int
	MessageID int
	CreatedAt time.Time // 添加创建时间用于超时检查
}

// State 对话状态管理器
type State struct {
	states map[int64]*Conversation
	mutex  sync.RWMutex
}

// NewState 创建新的对话状态管理器
func NewState() *State {
	return &State{
		states: make(map[int64]*Conversation),
	}
}

// Get 获取对话状态
func (s *State) Get(chatID int64) (*Conversation, bool) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	state, exists := s.states[chatID]
	return state, exists
}

// Set 设置对话状态
func (s *State) Set(chatID int64, conv *Conversation) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	conv.CreatedAt = time.Now() // 设置创建时间
	s.states[chatID] = conv
}

// CleanupExpired 清理过期的对话状态
func (s *State) CleanupExpired(timeout time.Duration) []int64 {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var expiredChats []int64
	now := time.Now()

	for chatID, conv := range s.states {
		if now.Sub(conv.CreatedAt) > timeout {
			expiredChats = append(expiredChats, chatID)
			delete(s.states, chatID)
		}
	}

	return expiredChats
}

// Delete 删除对话状态
func (s *State) Delete(chatID int64) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	delete(s.states, chatID)
}

// GetAll 获取所有对话状态（用于调试）
func (s *State) GetAll() map[int64]*Conversation {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	result := make(map[int64]*Conversation)
	for k, v := range s.states {
		result[k] = v
	}
	return result
}
