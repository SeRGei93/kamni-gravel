package valueobject

import (
	"fmt"
	"regexp"
	"strings"
)

// ResultLink представляет ссылку на результат (Strava/Komoot)
type ResultLink struct {
	URL      string
	Platform Platform
}

// Platform представляет платформу результата
type Platform string

const (
	PlatformStrava Platform = "strava"
	PlatformKomoot Platform = "komoot"
	PlatformNone   Platform = ""
)

var (
	stravaRe    = regexp.MustCompile(`^https?://(www\.)?strava\.com/activities/\d+$`)
	stravaAppRe = regexp.MustCompile(`^https?://(www\.)?strava\.app\.link/[A-Za-z0-9]+$`)
	komootRe    = regexp.MustCompile(`^https?://(www\.)?komoot\.com/tour/\d+$`)
)

// NewResultLink создаёт и валидирует ссылку на результат
func NewResultLink(url string) (*ResultLink, error) {
	url = strings.TrimSpace(strings.ToLower(url))
	
	if url == "" {
		return nil, fmt.Errorf("result link cannot be empty")
	}
	
	// Проверяем Strava
	if stravaRe.MatchString(url) || stravaAppRe.MatchString(url) {
		return &ResultLink{
			URL:      url,
			Platform: PlatformStrava,
		}, nil
	}
	
	// Проверяем Komoot
	if komootRe.MatchString(url) {
		return &ResultLink{
			URL:      url,
			Platform: PlatformKomoot,
		}, nil
	}
	
	// Если содержит strava или komoot, но формат неверный
	if strings.Contains(url, "strava.com") {
		return nil, fmt.Errorf("invalid Strava URL format. Example: https://www.strava.com/activities/14758223172")
	}
	if strings.Contains(url, "komoot.com") {
		return nil, fmt.Errorf("invalid Komoot URL format. Example: https://www.komoot.com/tour/2308024419")
	}
	
	return nil, fmt.Errorf("unsupported platform. Only Strava and Komoot links are accepted")
}

// IsStrava проверяет, является ли ссылка Strava
func (rl *ResultLink) IsStrava() bool {
	return rl.Platform == PlatformStrava
}

// IsKomoot проверяет, является ли ссылка Komoot
func (rl *ResultLink) IsKomoot() bool {
	return rl.Platform == PlatformKomoot
}

// String возвращает URL
func (rl *ResultLink) String() string {
	return rl.URL
}
