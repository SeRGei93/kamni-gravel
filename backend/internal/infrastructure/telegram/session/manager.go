package session

import (
	"context"
	"sync"
	"time"
)

// SessionState представляет состояние сессии пользователя
type SessionState string

const (
	StateIdle                     SessionState = "idle"
	StateAwaitingBikeType         SessionState = "awaiting_bike_type"
	StateAwaitingGender           SessionState = "awaiting_gender"
	StateAwaitingGiftGender       SessionState = "awaiting_gift_gender"
	StateAwaitingGiftBikeType     SessionState = "awaiting_gift_bike_type"
	StateAwaitingGiftDesc         SessionState = "awaiting_gift_desc"
	StateAwaitingGiftPhoto        SessionState = "awaiting_gift_photo"
	StateAwaitingResultLink       SessionState = "awaiting_result_link"
)

// Session представляет сессию пользователя
type Session struct {
	UserID    int64
	State     SessionState
	Data      map[string]interface{} // временные данные сессии
	UpdatedAt time.Time
}

// Manager управляет сессиями пользователей
type Manager struct {
	sessions map[int64]*Session
	mu       sync.RWMutex
	timeout  time.Duration
}

// NewManager создаёт новый менеджер сессий
func NewManager(timeout time.Duration) *Manager {
	m := &Manager{
		sessions: make(map[int64]*Session),
		timeout:  timeout,
	}
	
	// Запускаем очистку устаревших сессий
	go m.cleanupLoop(context.Background())
	
	return m
}

// GetSession возвращает сессию пользователя (создаёт новую, если не существует)
func (m *Manager) GetSession(userID int64) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[userID]
	if !exists {
		session = &Session{
			UserID:    userID,
			State:     StateIdle,
			Data:      make(map[string]interface{}),
			UpdatedAt: time.Now(),
		}
		m.sessions[userID] = session
	}

	return session
}

// SetState устанавливает состояние сессии
func (m *Manager) SetState(userID int64, state SessionState) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.getOrCreateSession(userID)
	session.State = state
	session.UpdatedAt = time.Now()
}

// GetState возвращает текущее состояние сессии
func (m *Manager) GetState(userID int64) SessionState {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[userID]
	if !exists {
		return StateIdle
	}

	return session.State
}

// SetData сохраняет данные в сессии
func (m *Manager) SetData(userID int64, key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.getOrCreateSession(userID)
	session.Data[key] = value
	session.UpdatedAt = time.Now()
}

// GetData возвращает данные из сессии
func (m *Manager) GetData(userID int64, key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[userID]
	if !exists {
		return nil, false
	}

	value, ok := session.Data[key]
	return value, ok
}

// ClearSession очищает сессию пользователя
func (m *Manager) ClearSession(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, userID)
}

// ResetState сбрасывает состояние сессии в idle
func (m *Manager) ResetState(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := m.getOrCreateSession(userID)
	session.State = StateIdle
	session.Data = make(map[string]interface{})
	session.UpdatedAt = time.Now()
}

// getOrCreateSession возвращает или создаёт сессию (без блокировки)
func (m *Manager) getOrCreateSession(userID int64) *Session {
	session, exists := m.sessions[userID]
	if !exists {
		session = &Session{
			UserID:    userID,
			State:     StateIdle,
			Data:      make(map[string]interface{}),
			UpdatedAt: time.Now(),
		}
		m.sessions[userID] = session
	}
	return session
}

// cleanupLoop периодически удаляет устаревшие сессии
func (m *Manager) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.cleanup()
		}
	}
}

// cleanup удаляет устаревшие сессии
func (m *Manager) cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for userID, session := range m.sessions {
		if now.Sub(session.UpdatedAt) > m.timeout {
			delete(m.sessions, userID)
		}
	}
}
