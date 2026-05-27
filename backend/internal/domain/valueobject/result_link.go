package valueobject

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// ResultLink представляет ссылку на результат Strava.
type ResultLink struct {
	URL      string
	Platform Platform
}

// Platform представляет платформу результата
type Platform string

const (
	PlatformStrava Platform = "strava"
	PlatformNone   Platform = ""
)

var (
	stravaActivityIDRe = regexp.MustCompile(`^\d+$`)
	stravaAppTokenRe   = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
)

// NewResultLink создаёт и валидирует ссылку на результат
func NewResultLink(rawURL string) (*ResultLink, error) {
	rawURL = strings.TrimSpace(rawURL)

	if rawURL == "" {
		return nil, fmt.Errorf("result link cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid Strava URL format. Example: https://www.strava.com/activities/14758223172")
	}

	parsedURL.Scheme = strings.ToLower(parsedURL.Scheme)
	host := strings.ToLower(parsedURL.Hostname())

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("invalid Strava URL format. Example: https://www.strava.com/activities/14758223172")
	}
	if host == "" || parsedURL.Port() != "" || parsedURL.User != nil {
		return nil, fmt.Errorf("invalid Strava URL format. Example: https://www.strava.com/activities/14758223172")
	}

	parsedURL.Host = host

	if isStravaActivityURL(host, parsedURL.Path) || isStravaAppURL(host, parsedURL.Path) {
		return &ResultLink{
			URL:      parsedURL.String(),
			Platform: PlatformStrava,
		}, nil
	}

	if strings.Contains(host, "strava") {
		return nil, fmt.Errorf("invalid Strava URL format. Example: https://www.strava.com/activities/14758223172")
	}

	return nil, fmt.Errorf("unsupported platform. Only Strava links are accepted")
}

func isStravaActivityURL(host string, path string) bool {
	if host != "strava.com" && host != "www.strava.com" {
		return false
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 2 && parts[0] == "activities" && stravaActivityIDRe.MatchString(parts[1])
}

func isStravaAppURL(host string, path string) bool {
	if host != "strava.app.link" && host != "www.strava.app.link" {
		return false
	}

	parts := strings.Split(strings.Trim(path, "/"), "/")
	return len(parts) == 1 && stravaAppTokenRe.MatchString(parts[0])
}

// IsStrava проверяет, является ли ссылка Strava
func (rl *ResultLink) IsStrava() bool {
	return rl.Platform == PlatformStrava
}

// String возвращает URL
func (rl *ResultLink) String() string {
	return rl.URL
}
