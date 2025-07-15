package handlers

import (
	"sync"
	"time"
)

// ChatModelPreference 存储聊天的模型偏好
type ChatModelPreference struct {
	ModelID   string    `json:"model_id"`
	Provider  string    `json:"provider"`
	ModelName string    `json:"model_name"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PreferenceManager 管理聊天的模型偏好
type PreferenceManager struct {
	preferences map[int64]*ChatModelPreference // chatID -> preference
	mutex       sync.RWMutex
}

// NewPreferenceManager 创建新的偏好管理器
func NewPreferenceManager() *PreferenceManager {
	return &PreferenceManager{
		preferences: make(map[int64]*ChatModelPreference),
	}
}

// SetChatPreference 设置聊天的模型偏好
func (pm *PreferenceManager) SetChatPreference(chatID int64, modelID, provider, modelName string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.preferences[chatID] = &ChatModelPreference{
		ModelID:   modelID,
		Provider:  provider,
		ModelName: modelName,
		UpdatedAt: time.Now(),
	}
}

// GetChatPreference 获取聊天的模型偏好
func (pm *PreferenceManager) GetChatPreference(chatID int64) *ChatModelPreference {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	pref, exists := pm.preferences[chatID]
	if !exists {
		return nil
	}

	// 返回副本以避免并发问题
	return &ChatModelPreference{
		ModelID:   pref.ModelID,
		Provider:  pref.Provider,
		ModelName: pref.ModelName,
		UpdatedAt: pref.UpdatedAt,
	}
}

// ClearChatPreference 清除聊天的模型偏好
func (pm *PreferenceManager) ClearChatPreference(chatID int64) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	delete(pm.preferences, chatID)
}

// HasChatPreference 检查聊天是否有模型偏好
func (pm *PreferenceManager) HasChatPreference(chatID int64) bool {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	_, exists := pm.preferences[chatID]
	return exists
}

// GetAllPreferences 获取所有偏好（用于调试）
func (pm *PreferenceManager) GetAllPreferences() map[int64]*ChatModelPreference {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result := make(map[int64]*ChatModelPreference)
	for chatID, pref := range pm.preferences {
		result[chatID] = &ChatModelPreference{
			ModelID:   pref.ModelID,
			Provider:  pref.Provider,
			ModelName: pref.ModelName,
			UpdatedAt: pref.UpdatedAt,
		}
	}
	return result
}
