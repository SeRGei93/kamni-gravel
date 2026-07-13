package entity

import (
	"testing"
	"time"
)

func intp(v int) *int           { return &v }
func f64p(v float64) *float64   { return &v }
func tp(t time.Time) *time.Time { return &t }

func TestResultComputedElapsedSec(t *testing.T) {
	start := time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC)
	finish := start.Add(6*time.Hour + 15*time.Minute)

	cases := []struct {
		name   string
		start  *time.Time
		finish *time.Time
		want   *int
	}{
		{name: "both set", start: tp(start), finish: tp(finish), want: intp(6*3600 + 15*60)},
		{name: "missing start", start: nil, finish: tp(finish), want: nil},
		{name: "missing finish", start: tp(start), finish: nil, want: nil},
		{name: "neither", want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Result{StartedAt: tc.start, FinishedAt: tc.finish}
			got := r.ComputedElapsedSec()
			assertIntPtrEqual(t, got, tc.want)
		})
	}
}

func TestResultIdleTimeSec(t *testing.T) {
	cases := []struct {
		name    string
		elapsed *int
		moving  *int
		want    *int
	}{
		{name: "idle from elapsed minus moving", elapsed: intp(3600), moving: intp(3000), want: intp(600)},
		{name: "zero idle", elapsed: intp(3600), moving: intp(3600), want: intp(0)},
		{name: "missing moving", elapsed: intp(3600), moving: nil, want: nil},
		{name: "missing elapsed", elapsed: nil, moving: intp(100), want: nil},
		{name: "negative idle -> nil", elapsed: intp(100), moving: intp(200), want: nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Result{ElapsedTimeSec: tc.elapsed, MovingTimeSec: tc.moving}
			assertIntPtrEqual(t, r.IdleTimeSec(), tc.want)
		})
	}
}

func TestResultAvgSpeeds(t *testing.T) {
	// 202000 м за 36000 с (10 ч) = 20.2 км/ч; в движении 18000 с (5 ч) = 40.4 км/ч.
	r := &Result{
		DistanceMeters: intp(202000),
		ElapsedTimeSec: intp(36000),
		MovingTimeSec:  intp(18000),
	}
	assertFloatPtrApprox(t, r.AvgSpeedKmh(), f64p(20.2))
	assertFloatPtrApprox(t, r.AvgMovingSpeedKmh(), f64p(40.4))
}

func TestResultAvgSpeedsNilWhenInputMissing(t *testing.T) {
	cases := []struct {
		name string
		r    *Result
	}{
		{name: "no distance", r: &Result{ElapsedTimeSec: intp(3600), MovingTimeSec: intp(3600)}},
		{name: "zero distance", r: &Result{DistanceMeters: intp(0), ElapsedTimeSec: intp(3600)}},
		{name: "no elapsed", r: &Result{DistanceMeters: intp(202000)}},
		{name: "no moving for moving speed", r: &Result{DistanceMeters: intp(202000), ElapsedTimeSec: intp(3600)}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "no moving for moving speed" {
				if tc.r.AvgMovingSpeedKmh() != nil {
					t.Fatalf("expected nil avg moving speed, got %v", *tc.r.AvgMovingSpeedKmh())
				}
				return
			}
			if tc.r.AvgSpeedKmh() != nil {
				t.Fatalf("expected nil avg speed, got %v", *tc.r.AvgSpeedKmh())
			}
		})
	}
}

func TestResultHeartRateTimeProduct(t *testing.T) {
	tests := []struct {
		name    string
		elapsed *int
		avgHR   *int
		want    *float64
	}{
		{
			name:    "one hour at 120 bpm",
			elapsed: intp(3600),
			avgHR:   intp(120),
			want:    f64p(7200),
		},
		{
			name:    "uses seconds as fractional minutes",
			elapsed: intp(90),
			avgHR:   intp(100),
			want:    f64p(150),
		},
		{name: "missing elapsed time", avgHR: intp(120)},
		{name: "missing average heart rate", elapsed: intp(3600)},
		{name: "zero elapsed time", elapsed: intp(0), avgHR: intp(120)},
		{name: "zero average heart rate", elapsed: intp(3600), avgHR: intp(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Result{ElapsedTimeSec: tt.elapsed, AvgHeartRate: tt.avgHR}
			got := r.HeartRateTimeProduct()
			if tt.want == nil {
				if got != nil {
					t.Fatalf("HeartRateTimeProduct() = %v, want nil", *got)
				}
				return
			}
			assertFloatPtrApprox(t, got, tt.want)
		})
	}
}

func TestResultRideDate(t *testing.T) {
	loc := time.FixedZone("Minsk", 3*3600)
	start := time.Date(2025, 6, 15, 23, 45, 0, 0, loc)
	r := &Result{StartedAt: tp(start)}
	got := r.RideDate()
	if got == nil {
		t.Fatal("expected ride date, got nil")
	}
	if got.Format("2006-01-02") != "2025-06-15" {
		t.Fatalf("ride date mismatch: got %s", got.Format("2006-01-02"))
	}

	if (&Result{}).RideDate() != nil {
		t.Fatal("expected nil ride date when start missing")
	}
}

func assertIntPtrEqual(t *testing.T, got, want *int) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("nil mismatch: got %v want %v", got, want)
	case *got != *want:
		t.Fatalf("value mismatch: got %d want %d", *got, *want)
	}
}

func assertFloatPtrApprox(t *testing.T, got, want *float64) {
	t.Helper()
	if got == nil || want == nil {
		t.Fatalf("nil mismatch: got %v want %v", got, want)
	}
	if diff := *got - *want; diff > 0.001 || diff < -0.001 {
		t.Fatalf("value mismatch: got %f want %f", *got, *want)
	}
}
