package valueobject

import (
	"fmt"
	"strings"
	"time"
)

const minskUTCOffsetSeconds = 3 * 60 * 60

var minskLocation = time.FixedZone("Europe/Minsk", minskUTCOffsetSeconds)

// MinskLocation возвращает фиксированную временную зону Минска UTC+3.
func MinskLocation() *time.Location {
	return minskLocation
}

// ParseMinskDateTime разбирает дату/время события как Минск UTC+3.
func ParseMinskDateTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("date value cannot be empty")
	}

	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed.In(MinskLocation()), nil
	}

	for _, layout := range []string{
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	} {
		if parsed, err := time.ParseInLocation(layout, value, MinskLocation()); err == nil {
			return parsed, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse Minsk date time: %s", value)
}

// FormatMinskDateTime форматирует момент времени в фиксированной зоне Минска UTC+3.
func FormatMinskDateTime(value time.Time) string {
	return value.In(MinskLocation()).Format("02.01.2006 15:04")
}
