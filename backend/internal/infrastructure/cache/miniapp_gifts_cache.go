// Package cache содержит файловые кеши инфраструктурного слоя.
package cache

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gravel_bot/internal/application/dto"
)

// DefaultMiniappGiftsCacheTTL — TTL по умолчанию для кеша каталога подарков мини-приложения.
// Это страховочный предел: основная инвалидация выполняется по событию (при одобрении подарка).
const DefaultMiniappGiftsCacheTTL = 1 * time.Hour

// MiniappGiftsCache — файловый кеш каталога одобренных подарков для первого экрана
// Telegram Mini App. Каждая комбинация ключа (eventID, gender, bikeType) хранится
// отдельным JSON-файлом на диске, поэтому кеш переживает перезапуск бэкенда.
// Инвалидация выполняется по событию (InvalidateEvent) при изменении одобренного каталога.
type MiniappGiftsCache struct {
	dir string
	ttl time.Duration
	mu  sync.RWMutex
}

// miniappGiftsCacheEntry — формат записи на диске. cached_at хранится отдельно,
// чтобы TTL не зависел от mtime файла.
type miniappGiftsCacheEntry struct {
	CachedAt int64          `json:"cached_at"` // unix nanoseconds
	Gifts    []*dto.GiftDTO `json:"gifts"`
}

// NewMiniappGiftsCache создаёт файловый кеш в каталоге dir, создавая каталог при необходимости.
// ttl <= 0 отключает истечение по времени (инвалидация выполняется только по событию).
func NewMiniappGiftsCache(dir string, ttl time.Duration) (*MiniappGiftsCache, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, fmt.Errorf("miniapp gifts cache dir is empty")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create miniapp gifts cache dir %q: %w", dir, err)
	}
	return &MiniappGiftsCache{dir: dir, ttl: ttl}, nil
}

// Get возвращает закешированный каталог для ключа. Второй результат false означает промах:
// файла нет, файл повреждён или истёк TTL.
func (c *MiniappGiftsCache) Get(eventID uint, gender, bikeType string) ([]*dto.GiftDTO, bool) {
	if c == nil {
		return nil, false
	}

	path := c.filePath(eventID, gender, bikeType)

	c.mu.RLock()
	data, err := os.ReadFile(path)
	c.mu.RUnlock()
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("WARN Miniapp gifts cache read failed: event_id=%d gender=%q bike_type=%q error=%v", eventID, gender, bikeType, err)
		}
		return nil, false
	}

	var entry miniappGiftsCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		// Повреждённый файл считаем промахом и по возможности удаляем.
		log.Printf("WARN Miniapp gifts cache entry corrupt, dropping: path=%s error=%v", path, err)
		c.removeFile(path)
		return nil, false
	}

	if c.ttl > 0 {
		expiresAt := time.Unix(0, entry.CachedAt).Add(c.ttl)
		if time.Now().After(expiresAt) {
			return nil, false
		}
	}

	return entry.Gifts, true
}

// Set сохраняет каталог для ключа на диск атомарной записью (temp-файл + rename).
// Ошибки записи деградируют до «нет кеша» и никогда не прерывают вызывающий запрос.
func (c *MiniappGiftsCache) Set(eventID uint, gender, bikeType string, gifts []*dto.GiftDTO) {
	if c == nil {
		return
	}

	entry := miniappGiftsCacheEntry{
		CachedAt: time.Now().UnixNano(),
		Gifts:    gifts,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("WARN Miniapp gifts cache marshal failed: event_id=%d gender=%q bike_type=%q error=%v", eventID, gender, bikeType, err)
		return
	}

	path := c.filePath(eventID, gender, bikeType)
	tmp := path + ".tmp"

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		log.Printf("WARN Miniapp gifts cache write failed: path=%s error=%v", tmp, err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		log.Printf("WARN Miniapp gifts cache rename failed: path=%s error=%v", path, err)
		c.removeFileLocked(tmp)
	}
}

// InvalidateEvent удаляет все закешированные комбинации фильтров для события.
// Вызывается при изменении одобренного каталога (одобрение/правка/удаление подарка).
func (c *MiniappGiftsCache) InvalidateEvent(eventID uint) {
	if c == nil {
		return
	}

	pattern := filepath.Join(c.dir, fmt.Sprintf("gift_%d__*.json", eventID))

	c.mu.Lock()
	defer c.mu.Unlock()

	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("WARN Miniapp gifts cache invalidate glob failed: event_id=%d error=%v", eventID, err)
		return
	}
	for _, path := range matches {
		c.removeFileLocked(path)
	}
}

// InvalidateAll удаляет весь кеш каталога подарков.
func (c *MiniappGiftsCache) InvalidateAll() {
	if c == nil {
		return
	}

	pattern := filepath.Join(c.dir, "gift_*.json")

	c.mu.Lock()
	defer c.mu.Unlock()

	matches, err := filepath.Glob(pattern)
	if err != nil {
		log.Printf("WARN Miniapp gifts cache invalidate-all glob failed: error=%v", err)
		return
	}
	for _, path := range matches {
		c.removeFileLocked(path)
	}
}

func (c *MiniappGiftsCache) filePath(eventID uint, gender, bikeType string) string {
	name := fmt.Sprintf("gift_%d__%s__%s.json", eventID, sanitizeFilterToken(gender), sanitizeFilterToken(bikeType))
	return filepath.Join(c.dir, name)
}

func (c *MiniappGiftsCache) removeFile(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeFileLocked(path)
}

func (c *MiniappGiftsCache) removeFileLocked(path string) {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		log.Printf("WARN Miniapp gifts cache remove failed: path=%s error=%v", path, err)
	}
}

// sanitizeFilterToken приводит значение фильтра к безопасному для имени файла виду.
// Пустое значение трактуется как "all" (та же семантика, что и в query-слое),
// поэтому пустой и явный "all" фильтр указывают на один и тот же файл.
func sanitizeFilterToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "all"
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	return b.String()
}
