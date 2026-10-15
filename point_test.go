package gad_test

import (
	"errors"
	"math"
	"math/rand/v2"
	"strings"
	"testing"

	gad "github.com/gomaja/go-gad"
)

func constructorError(t *testing.T, err, want error, field string) {
	t.Helper()
	var fe *gad.FieldError
	if !errors.Is(err, want) || !errors.As(err, &fe) || fe.Octet != 0 || fe.Field != field || !strings.HasPrefix(err.Error(), "gad: ") {
		t.Fatalf("error = %#v, want %v in %s at octet 0", err, want, field)
	}
}

func TestPointCoordinateBins(t *testing.T) {
	const latStep = 90.0 / (1 << 23)
	const lonStep = 360.0 / (1 << 24)
	for _, tc := range []struct {
		lat, lon float64
		south    bool
		n        uint32
		l        gad.Longitude
	}{
		{0, 0, false, 0, 0}, {math.Copysign(0, -1), math.Copysign(0, -1), true, 0, 0},
		{0, math.SmallestNonzeroFloat64, false, 0, 0}, {0, -math.SmallestNonzeroFloat64, false, 0, -1},
		{.75 * latStep, .75 * lonStep, false, 0, 0}, {-.75 * latStep, -.25 * lonStep, true, 0, -1},
		{math.Nextafter(latStep, 0), math.Nextafter(lonStep, 0), false, 0, 0},
		{latStep, lonStep, false, 1, 1}, {math.Nextafter(latStep, 90), math.Nextafter(lonStep, 180), false, 1, 1},
		{0, math.Nextafter(-lonStep, -180), false, 0, -2}, {0, -lonStep, false, 0, -1}, {0, math.Nextafter(-lonStep, 0), false, 0, -1},
		{45 + .75*latStep, 90 + .75*lonStep, false, 1 << 22, 1 << 22},
		{-45 - .75*latStep, -90 - .25*lonStep, true, 1 << 22, -(1 << 22) - 1},
		{90, 0, false, (1 << 23) - 1, 0}, {-90, 0, true, (1 << 23) - 1, 0},
		{math.Nextafter(90, 0), math.Nextafter(180, 0), false, (1 << 23) - 1, (1 << 23) - 1},
		{math.Nextafter(-90, 0), -180, true, (1 << 23) - 1, -(1 << 23)},
		{0, math.Nextafter(-180, 0), false, 0, -(1 << 23)},
	} {
		got, err := gad.PointFromDegrees(tc.lat, tc.lon)
		want := gad.Point{Latitude: gad.Latitude{South: tc.south, N: tc.n}, Longitude: tc.l}
		if err != nil || got != want {
			t.Fatalf("PointFromDegrees(%g,%g) = %+v,%v; want %+v", tc.lat, tc.lon, got, err, want)
		}
		if math.Signbit(got.Latitude.Degrees()) != tc.south {
			t.Fatal("lost latitude sign")
		}
	}
}

func TestPointCoordinateDomains(t *testing.T) {
	for _, ha := range []bool{false, true} {
		for _, bad := range []float64{math.NaN(), math.Inf(-1), math.Inf(1), math.Nextafter(-90, math.Inf(-1)), math.Nextafter(90, math.Inf(1))} {
			want := gad.ErrOutOfRange
			if math.IsNaN(bad) || math.IsInf(bad, 0) {
				want = gad.ErrNotFinite
			}
			var err error
			if ha {
				_, err = gad.HighAccuracyPointFromDegrees(bad, 0)
			} else {
				_, err = gad.PointFromDegrees(bad, 0)
			}
			constructorError(t, err, want, "Latitude")
		}
		for _, bad := range []float64{math.NaN(), math.Inf(-1), math.Inf(1), math.Nextafter(-180, math.Inf(-1)), 180} {
			want := gad.ErrOutOfRange
			if math.IsNaN(bad) || math.IsInf(bad, 0) {
				want = gad.ErrNotFinite
			}
			var err error
			if ha {
				_, err = gad.HighAccuracyPointFromDegrees(0, bad)
			} else {
				_, err = gad.PointFromDegrees(0, bad)
			}
			constructorError(t, err, want, "Longitude")
		}
	}
}

func TestHighAccuracyCoordinateBins(t *testing.T) {
	const step = 90.0 / (1 << 31)
	for _, tc := range []struct {
		x    float64
		want gad.HighAccuracyLatitude
	}{
		{0, 0}, {math.Copysign(0, -1), 0}, {math.SmallestNonzeroFloat64, 0}, {-math.SmallestNonzeroFloat64, -1},
		{math.Nextafter(step, 0), 0}, {step, 1}, {math.Nextafter(step, 90), 1},
		{math.Nextafter(-step, -90), -2}, {-step, -1}, {math.Nextafter(-step, 0), -1},
		{-90, math.MinInt32}, {90, math.MaxInt32}, {math.Nextafter(90, 0), math.MaxInt32},
	} {
		got, err := gad.HighAccuracyLatitudeFromDegrees(tc.x)
		if err != nil || got != tc.want {
			t.Fatalf("HA latitude(%g)=%d,%v want %d", tc.x, got, err, tc.want)
		}
		lon, err := gad.HighAccuracyLongitudeFromDegrees(tc.x * 2)
		if tc.x == 90 {
			if !errors.Is(err, gad.ErrOutOfRange) {
				t.Fatal(err)
			}
			continue
		}
		if err != nil || int32(lon) != int32(tc.want) {
			t.Fatalf("HA longitude(%g)=%d,%v want %d", tc.x*2, lon, err, tc.want)
		}
	}
	p, err := gad.HighAccuracyPointFromDegrees(90, math.Nextafter(180, 0))
	if err != nil || p.Latitude != math.MaxInt32 || p.Longitude != math.MaxInt32 {
		t.Fatalf("HA boundary = %+v,%v", p, err)
	}
}

func TestCoordinateCodeRoundTrips(t *testing.T) {
	if testing.Short() {
		t.Skip("exhaustive 24-bit coordinate code spaces")
	}
	for n := uint32(0); n < 1<<23; n++ {
		for _, south := range []bool{false, true} {
			want := gad.Latitude{South: south, N: n}
			got, err := gad.LatitudeFromDegrees(want.Degrees())
			if err != nil || got != want {
				t.Fatalf("latitude %+v: %+v,%v", want, got, err)
			}
		}
	}
	for n := int32(-1 << 23); n < 1<<23; n++ {
		want := gad.Longitude(n)
		got, err := gad.LongitudeFromDegrees(want.Degrees())
		if err != nil || got != want {
			t.Fatalf("longitude %d: %d,%v", want, got, err)
		}
	}
}

func TestHighAccuracyCoordinateCodeRoundTrips(t *testing.T) {
	r := rand.New(rand.NewPCG(23, 32))
	values := []int32{math.MinInt32, math.MinInt32 + 1, -1, 0, 1, math.MaxInt32 - 1, math.MaxInt32}
	for range 10000 {
		values = append(values, int32(r.Uint32()))
	}
	for _, n := range values {
		lat := gad.HighAccuracyLatitude(n)
		got, err := gad.HighAccuracyLatitudeFromDegrees(lat.Degrees())
		if err != nil || got != lat {
			t.Fatalf("HA latitude %d: %d,%v", n, got, err)
		}
		lon := gad.HighAccuracyLongitude(n)
		l, err := gad.HighAccuracyLongitudeFromDegrees(lon.Degrees())
		if err != nil || l != lon {
			t.Fatalf("HA longitude %d: %d,%v", n, l, err)
		}
	}
}

func FuzzPointFromDegrees(f *testing.F) {
	for _, p := range [][2]float64{{0, 0}, {math.Copysign(0, -1), 0}, {90, 0}, {-90, -180}, {45.000008046627045, 90.00001609325409}, {-45.000008046627045, -90.00000536441803}, {math.Nextafter(90, 0), math.Nextafter(180, 0)}, {math.Nextafter(-90, 0), math.Nextafter(-180, 0)}, {0, -math.SmallestNonzeroFloat64}, {math.NaN(), 0}, {0, math.Inf(1)}, {91, 181}} {
		f.Add(p[0], p[1])
	}
	f.Fuzz(func(t *testing.T, lat, lon float64) {
		p, err := gad.PointFromDegrees(lat, lon)
		invalid := math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lon) || math.IsInf(lon, 0) || lat < -90 || lat > 90 || lon < -180 || lon >= 180
		if invalid {
			if err == nil {
				t.Fatalf("invalid (%g,%g) accepted", lat, lon)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		lower := math.Abs(p.Latitude.Degrees())
		upper := lower + 90.0/(1<<23)
		if lower > math.Abs(lat) || math.Abs(lat) > upper || (math.Abs(lat) == upper && math.Abs(lat) != 90) || p.Latitude.South != math.Signbit(lat) {
			t.Fatalf("latitude %g outside %+v bin", lat, p.Latitude)
		}
		lo := p.Longitude.Degrees()
		hi := lo + 360.0/(1<<24)
		if lo > lon || lon >= hi {
			t.Fatalf("longitude %g outside [%g,%g)", lon, lo, hi)
		}
	})
}
