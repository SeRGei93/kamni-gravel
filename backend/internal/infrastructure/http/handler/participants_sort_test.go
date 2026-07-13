package handler

import (
	"testing"
	"time"

	"gravel_bot/internal/application/dto"
)

func sortIntPtr(v int) *int              { return &v }
func sortStrPtr(v string) *string        { return &v }
func sortTimePtr(v time.Time) *time.Time { return &v }
func sortFloatPtr(v float64) *float64    { return &v }

// idsOf возвращает порядок id после сортировки — удобно сравнивать.
func idsOf(items []*dto.ParticipantDTO) []uint {
	ids := make([]uint, len(items))
	for i, it := range items {
		ids[i] = it.ID
	}
	return ids
}

func equalIDs(a, b []uint) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestSortParticipantDTOsNumericAscDesc(t *testing.T) {
	build := func() []*dto.ParticipantDTO {
		return []*dto.ParticipantDTO{
			{ID: 1, ElapsedTimeSec: sortIntPtr(300)},
			{ID: 2, ElapsedTimeSec: sortIntPtr(100)},
			{ID: 3, ElapsedTimeSec: sortIntPtr(200)},
		}
	}

	asc := build()
	sortParticipantDTOs(asc, "elapsed_time", "asc")
	if got := idsOf(asc); !equalIDs(got, []uint{2, 3, 1}) {
		t.Fatalf("asc order mismatch: %v", got)
	}

	desc := build()
	sortParticipantDTOs(desc, "elapsed_time", "desc")
	if got := idsOf(desc); !equalIDs(got, []uint{1, 3, 2}) {
		t.Fatalf("desc order mismatch: %v", got)
	}
}

func TestSortParticipantDTOsMissingValuesLast(t *testing.T) {
	build := func() []*dto.ParticipantDTO {
		return []*dto.ParticipantDTO{
			{ID: 1, DistanceMeters: nil},
			{ID: 2, DistanceMeters: sortIntPtr(50000)},
			{ID: 3, DistanceMeters: nil},
			{ID: 4, DistanceMeters: sortIntPtr(20000)},
		}
	}

	asc := build()
	sortParticipantDTOs(asc, "distance_km", "asc")
	// Непустые по возрастанию (4, 2), затем пустые в исходном порядке (1, 3).
	if got := idsOf(asc); !equalIDs(got, []uint{4, 2, 1, 3}) {
		t.Fatalf("asc missing-last mismatch: %v", got)
	}

	desc := build()
	sortParticipantDTOs(desc, "distance_km", "desc")
	// Непустые по убыванию (2, 4), затем пустые остаются в конце.
	if got := idsOf(desc); !equalIDs(got, []uint{2, 4, 1, 3}) {
		t.Fatalf("desc missing-last mismatch: %v", got)
	}
}

func TestSortParticipantDTOsPlaceZeroTreatedAsMissing(t *testing.T) {
	items := []*dto.ParticipantDTO{
		{ID: 1, Place: 2},
		{ID: 2, Place: 0}, // нет места — в конец
		{ID: 3, Place: 1},
	}
	sortParticipantDTOs(items, "place", "asc")
	if got := idsOf(items); !equalIDs(got, []uint{3, 1, 2}) {
		t.Fatalf("place zero-last mismatch: %v", got)
	}
}

func TestSortParticipantDTOsUnknownKeyKeepsOrder(t *testing.T) {
	items := []*dto.ParticipantDTO{{ID: 3}, {ID: 1}, {ID: 2}}
	sortParticipantDTOs(items, "definitely_not_a_column", "asc")
	if got := idsOf(items); !equalIDs(got, []uint{3, 1, 2}) {
		t.Fatalf("unknown key should keep order, got: %v", got)
	}

	// Пустой ключ тоже сохраняет порядок.
	sortParticipantDTOs(items, "", "asc")
	if got := idsOf(items); !equalIDs(got, []uint{3, 1, 2}) {
		t.Fatalf("empty key should keep order, got: %v", got)
	}
}

func TestSortParticipantDTOsTimestampAndDate(t *testing.T) {
	t1 := time.Date(2026, 6, 20, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 22, 9, 0, 0, 0, time.UTC)

	byStart := []*dto.ParticipantDTO{
		{ID: 1, StartedAt: sortTimePtr(t2)},
		{ID: 2, StartedAt: sortTimePtr(t1)},
		{ID: 3, StartedAt: nil},
	}
	sortParticipantDTOs(byStart, "started_at", "asc")
	if got := idsOf(byStart); !equalIDs(got, []uint{2, 1, 3}) {
		t.Fatalf("started_at asc mismatch: %v", got)
	}

	byDate := []*dto.ParticipantDTO{
		{ID: 1, RideDate: sortStrPtr("2026-06-23")},
		{ID: 2, RideDate: sortStrPtr("2026-06-21")},
		{ID: 3, RideDate: sortStrPtr("")}, // пусто — в конец
	}
	sortParticipantDTOs(byDate, "ride_date", "asc")
	if got := idsOf(byDate); !equalIDs(got, []uint{2, 1, 3}) {
		t.Fatalf("ride_date asc mismatch: %v", got)
	}
}

// Дельта к прошлому году может быть отрицательной (медленнее) — порядок
// учитывает знак, отсутствующие значения уходят в конец.
func TestSortParticipantDTOsPrevElapsedDelta(t *testing.T) {
	build := func() []*dto.ParticipantDTO {
		return []*dto.ParticipantDTO{
			{ID: 1, PrevElapsedDeltaSec: sortIntPtr(-120)}, // медленнее на 2 мин
			{ID: 2, PrevElapsedDeltaSec: sortIntPtr(3300)}, // быстрее на 55 мин
			{ID: 3, PrevElapsedDeltaSec: nil},
			{ID: 4, PrevElapsedDeltaSec: sortIntPtr(0)},
		}
	}

	asc := build()
	sortParticipantDTOs(asc, "prev_elapsed_delta", "asc")
	if got := idsOf(asc); !equalIDs(got, []uint{1, 4, 2, 3}) {
		t.Fatalf("prev_elapsed_delta asc mismatch: %v", got)
	}

	desc := build()
	sortParticipantDTOs(desc, "prev_elapsed_delta", "desc")
	if got := idsOf(desc); !equalIDs(got, []uint{2, 4, 1, 3}) {
		t.Fatalf("prev_elapsed_delta desc mismatch: %v", got)
	}
}

func TestSortParticipantDTOsPeakAvgSpeedDelta(t *testing.T) {
	build := func() []*dto.ParticipantDTO {
		return []*dto.ParticipantDTO{
			{ID: 1, PeakAvgSpeedDeltaKmh: sortFloatPtr(30.9)},
			{ID: 2, PeakAvgSpeedDeltaKmh: nil},
			{ID: 3, PeakAvgSpeedDeltaKmh: sortFloatPtr(12.4)},
		}
	}

	asc := build()
	sortParticipantDTOs(asc, "peak_avg_speed_delta_kmh", "asc")
	if got := idsOf(asc); !equalIDs(got, []uint{3, 1, 2}) {
		t.Fatalf("peak_avg_speed_delta_kmh asc mismatch: %v", got)
	}

	desc := build()
	sortParticipantDTOs(desc, "peak_avg_speed_delta_kmh", "desc")
	if got := idsOf(desc); !equalIDs(got, []uint{1, 3, 2}) {
		t.Fatalf("peak_avg_speed_delta_kmh desc mismatch: %v", got)
	}
}

func TestSortParticipantDTOsHeartRateTimeProduct(t *testing.T) {
	build := func() []*dto.ParticipantDTO {
		return []*dto.ParticipantDTO{
			{ID: 1, HeartRateTimeProduct: sortFloatPtr(7200)},
			{ID: 2, HeartRateTimeProduct: nil},
			{ID: 3, HeartRateTimeProduct: sortFloatPtr(6900)},
		}
	}

	asc := build()
	sortParticipantDTOs(asc, "heart_rate_time_product", "asc")
	if got := idsOf(asc); !equalIDs(got, []uint{3, 1, 2}) {
		t.Fatalf("heart_rate_time_product asc mismatch: %v", got)
	}

	desc := build()
	sortParticipantDTOs(desc, "heart_rate_time_product", "desc")
	if got := idsOf(desc); !equalIDs(got, []uint{1, 3, 2}) {
		t.Fatalf("heart_rate_time_product desc mismatch: %v", got)
	}
}

// Сортировка стабильна: при равных значениях исходный порядок сохраняется —
// это гарантирует консистентную нарезку страниц (сортировка идёт до пагинации).
func TestSortParticipantDTOsStableForEqualValues(t *testing.T) {
	items := []*dto.ParticipantDTO{
		{ID: 1, PrizesCount: 0},
		{ID: 2, PrizesCount: 0},
		{ID: 3, PrizesCount: 0},
	}
	sortParticipantDTOs(items, "prizes_count", "asc")
	if got := idsOf(items); !equalIDs(got, []uint{1, 2, 3}) {
		t.Fatalf("stable order mismatch: %v", got)
	}
}
