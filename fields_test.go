package gad_test

import (
	"math"
	"testing"

	gad "github.com/gomaja/go-gad"
)

func TestUncertaintyExampleTables(t *testing.T) {
	// The tolerance is one unit in the last digit printed in each normative table.
	for _, tc := range []struct {
		name    string
		convert func(uint8) float64
		rows    [][3]float64
	}{
		{"6.2 Table 1", func(k uint8) float64 { return gad.Uncertainty(k).Metres() }, [][3]float64{{0, 0, 1}, {1, 1, 1}, {2, 2.1, .1}, {20, 57.3, .1}, {40, 443, 1}, {60, 3000, 1000}, {80, 20000, 1000}, {100, 138000, 1000}, {120, 927000, 1000}, {127, 1800000, 100000}}},
		{"6.4 Table 2", func(k uint8) float64 { return gad.UncertaintyAltitude(k).Metres() }, [][3]float64{{0, 0, 1}, {1, 1.13, .01}, {2, 2.28, .01}, {20, 28.7, .1}, {40, 75.8, .1}, {60, 153, .1}, {80, 279.4, .1}, {100, 486.6, .1}, {120, 826.1, .1}, {127, 990.5, .1}}},
		{"6.2a Table 6.2a-1", func(k uint8) float64 { return gad.HighAccuracyUncertainty(k).Metres() }, [][3]float64{{0, 0, 1}, {1, .006, .001}, {2, .01212, .00001}, {20, .14578, .00001}, {40, .36241, .00001}, {60, .68430, .00001}, {80, 1.16263, .00001}, {100, 1.87339, .00001}, {120, 2.92954, .00001}, {127, 3.40973, .00001}, {255, 46.49129, .00001}}},
		{"6.2b Table 6.2b-1", func(k uint8) float64 { return (gad.ScalableUncertainty{Extended: true, K: k}).Metres() }, [][3]float64{{0, 0, 1}, {1, .00778, .00001}, {2, .01577, .00001}, {20, .20068, .00001}, {40, .53560, .00001}, {60, 1.09457, .00001}, {80, 2.02744, .00001}, {100, 3.58434, .00001}, {120, 6.18271, .00001}, {127, 7.45551, .00001}, {253, 195.12396, .00001}, {254, 200, 1}, {255, math.Inf(1), 0}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, row := range tc.rows {
				got := tc.convert(uint8(row[0]))
				if got == row[1] {
					continue
				}
				if math.Abs(got-row[1]) > row[2] {
					t.Errorf("K=%g got %.12g want %g within %g", row[0], got, row[1], row[2])
				}
			}
		})
	}
}

func TestUncertaintyConstructors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		max    int
		metres func(int) float64
		from   func(float64) (int, error)
	}{
		{"Uncertainty", 127, func(k int) float64 { return gad.Uncertainty(k).Metres() }, func(r float64) (int, error) { v, e := gad.UncertaintyFromMetres(r); return int(v), e }},
		{"High accuracy uncertainty", 255, func(k int) float64 { return gad.HighAccuracyUncertainty(k).Metres() }, func(r float64) (int, error) { v, e := gad.HighAccuracyUncertaintyFromMetres(r); return int(v), e }},
		{"Uncertainty altitude", 127, func(k int) float64 { return gad.UncertaintyAltitude(k).Metres() }, func(r float64) (int, error) { v, e := gad.UncertaintyAltitudeFromMetres(r); return int(v), e }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for k := 0; k <= tc.max; k++ {
				x := tc.metres(k)
				got, e := tc.from(x)
				if e != nil || got != k {
					t.Fatalf("code %d round trip = %d,%v", k, got, e)
				}
				if k > 0 {
					prev := tc.metres(k - 1)
					if x <= prev {
						t.Fatal("not strictly increasing")
					}
					got, e = tc.from(math.Nextafter(prev, math.Inf(1)))
					if e != nil || got != k {
						t.Fatalf("above code %d = %d,%v want %d", k-1, got, e, k)
					}
				}
			}
			for _, bad := range []float64{-1, math.Nextafter(tc.metres(tc.max), math.Inf(1))} {
				_, e := tc.from(bad)
				constructorError(t, e, gad.ErrOutOfRange, tc.name)
			}
			for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
				_, e := tc.from(bad)
				constructorError(t, e, gad.ErrNotFinite, tc.name)
			}
		})
	}
	for _, extended := range []bool{false, true} {
		for k := 0; k <= 255; k++ {
			want := gad.ScalableUncertainty{Extended: extended, K: uint8(k)}
			got, e := gad.ScalableUncertaintyFromMetres(want.Metres(), extended)
			if e != nil || got != want {
				t.Fatalf("scalable %+v round trip %+v,%v", want, got, e)
			}
			if k > 0 && want.Metres() <= (gad.ScalableUncertainty{Extended: extended, K: uint8(k - 1)}).Metres() {
				t.Fatal("scalable not increasing")
			}
		}
	}
	for _, tc := range []struct {
		x float64
		k uint8
	}{{math.Nextafter((gad.ScalableUncertainty{Extended: true, K: 253}).Metres(), math.Inf(1)), 254}, {200, 254}, {math.Nextafter(200, math.Inf(1)), 255}, {math.Inf(1), 255}} {
		got, e := gad.ScalableUncertaintyFromMetres(tc.x, true)
		if e != nil || got.K != tc.k || !got.Extended {
			t.Fatalf("scalable %g = %+v,%v", tc.x, got, e)
		}
	}
}

func TestFieldConstructorBins(t *testing.T) {
	for _, tc := range []struct {
		name   string
		from   func(float64) (float64, error)
		values [][2]float64
		bad    []float64
	}{
		{"Orientation of major axis", func(x float64) (float64, error) { v, e := gad.OrientationFromDegrees(x); return v.Degrees(), e }, [][2]float64{{0, 0}, {.999, 0}, {1, 1}, {179, 179}, {math.Nextafter(180, 0), 179}}, []float64{-1, 180}},
		{"Altitude", func(x float64) (float64, error) { v, e := gad.AltitudeFromMetres(x); return v.Metres(), e }, [][2]float64{{0, 0}, {math.Copysign(0, -1), math.Copysign(0, -1)}, {.99, 0}, {-1.99, -1}, {32767, 32767}, {1e100, 32767}, {-1e100, -32767}}, nil},
		{"Altitude", func(x float64) (float64, error) { v, e := gad.HighAccuracyAltitudeFromMetres(x); return v.Metres(), e }, [][2]float64{{-500, -500}, {10000, 10000}, {1.0 / 128, 1.0 / 128}, {math.Nextafter(1.0/128, 0), 0}, {-.0001, -1.0 / 128}}, []float64{-500.01, 10000.01}},
		{"Inner radius", func(x float64) (float64, error) { v, e := gad.InnerRadiusFromMetres(x); return v.Metres(), e }, [][2]float64{{0, 0}, {4.99, 0}, {5, 5}, {327675, 327675}, {1e100, 327675}}, []float64{-1}},
		{"Offset angle", func(x float64) (float64, error) { v, e := gad.OffsetAngleFromDegrees(x); return v.Degrees(), e }, [][2]float64{{0, 0}, {1.999, 0}, {2, 2}, {359.999, 358}}, []float64{-1, 360}},
		{"Included angle", func(x float64) (float64, error) { v, e := gad.IncludedAngleFromDegrees(x); return v.Degrees(), e }, [][2]float64{{math.SmallestNonzeroFloat64, 2}, {.0001, 2}, {2, 2}, {2.0001, 4}, {360, 360}}, []float64{0, -1, math.Nextafter(360, math.Inf(1))}},
		{"Bearing", func(x float64) (float64, error) { v, e := gad.BearingFromDegrees(x); return v.Degrees(), e }, [][2]float64{{0, 0}, {.999, 0}, {1, 1}, {359.999, 359}}, []float64{-1, 360}},
		{"Horizontal speed", func(x float64) (float64, error) { v, e := gad.HorizontalSpeedFromKmh(x); return v.Kmh(), e }, [][2]float64{{0, 0}, {.49, 0}, {math.Nextafter(.5, 0), 0}, {.5, 1}, {1.49, 1}, {math.Nextafter(1.5, 0), 1}, {1.5, 2}, {65534.5, 65535}, {1e100, 65535}}, []float64{-1}},
		{"Vertical speed", func(x float64) (float64, error) { v, e := gad.VerticalSpeedFromKmh(x); return v.Kmh(), e }, [][2]float64{{0, 0}, {.49, 0}, {math.Nextafter(.5, 0), 0}, {.5, 1}, {1.49, 1}, {math.Nextafter(1.5, 0), 1}, {1.5, 2}, {254.5, 255}, {1e100, 255}}, []float64{-1}},
		{"Uncertainty speed", func(x float64) (float64, error) { v, e := gad.UncertaintySpeedFromKmh(x); y, _ := v.Kmh(); return y, e }, [][2]float64{{0, 0}, {.0001, 1}, {1, 1}, {1.0001, 2}, {254, 254}}, []float64{-1, math.Nextafter(254, math.Inf(1)), 255}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, pair := range tc.values {
				got, e := tc.from(pair[0])
				if e != nil || got != pair[1] || math.Signbit(got) != math.Signbit(pair[1]) {
					t.Fatalf("from(%g)=%g,%v want %g", pair[0], got, e, pair[1])
				}
			}
			for _, x := range tc.bad {
				_, e := tc.from(x)
				constructorError(t, e, gad.ErrOutOfRange, tc.name)
			}
			for _, x := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
				_, e := tc.from(x)
				constructorError(t, e, gad.ErrNotFinite, tc.name)
			}
		})
	}
}

func TestScalarCodeRoundTrips(t *testing.T) {
	for n := 0; n <= 65535; n++ {
		hs := gad.HorizontalSpeed(n)
		h, e := gad.HorizontalSpeedFromKmh(hs.Kmh())
		if e != nil || h != hs {
			t.Fatalf("horizontal %d", n)
		}
		ir := gad.InnerRadius(n)
		r, e := gad.InnerRadiusFromMetres(ir.Metres())
		if e != nil || r != ir {
			t.Fatalf("radius %d", n)
		}
		if n <= 32767 {
			for _, depth := range []bool{false, true} {
				a := gad.Altitude{Depth: depth, N: uint16(n)}
				got, e := gad.AltitudeFromMetres(a.Metres())
				if e != nil || got != a {
					t.Fatalf("altitude %+v: %+v,%v", a, got, e)
				}
			}
		}
		if n <= 255 {
			v := gad.VerticalSpeed(n)
			got, e := gad.VerticalSpeedFromKmh(v.Kmh())
			if e != nil || got != v {
				t.Fatalf("vertical %d", n)
			}
		}
		if n <= 254 {
			s := gad.UncertaintySpeed(n)
			x, ok := s.Kmh()
			got, e := gad.UncertaintySpeedFromKmh(x)
			if !ok || e != nil || got != s {
				t.Fatalf("uncertainty speed %d", n)
			}
		}
		if n <= 359 {
			b := gad.Bearing(n)
			got, e := gad.BearingFromDegrees(b.Degrees())
			if e != nil || got != b {
				t.Fatalf("bearing %d", n)
			}
		}
		if n <= 179 {
			o := gad.Orientation(n)
			got, e := gad.OrientationFromDegrees(o.Degrees())
			if e != nil || got != o {
				t.Fatalf("orientation %d", n)
			}
			a := gad.OffsetAngle(n)
			ag, e := gad.OffsetAngleFromDegrees(a.Degrees())
			if e != nil || ag != a {
				t.Fatalf("offset %d", n)
			}
			i := gad.IncludedAngle(n)
			ig, e := gad.IncludedAngleFromDegrees(i.Degrees())
			if e != nil || ig != i {
				t.Fatalf("included %d", n)
			}
		}
	}
	if x, ok := gad.UncertaintySpeedUnspecified.Kmh(); x != 0 || ok {
		t.Fatalf("unspecified = %g,%t", x, ok)
	}
	for n := 0; n <= 127; n++ {
		c := gad.Confidence(n)
		p, ok := c.Percent()
		want := n >= 1 && n <= 100
		if ok != want || (want && p != n) || (!want && p != 0) {
			t.Fatalf("confidence %d = %d,%t", n, p, ok)
		}
		if n <= 100 {
			got, e := gad.ConfidenceFromPercent(n)
			if e != nil || got != c {
				t.Fatalf("confidence constructor %d", n)
			}
		}
	}
	for _, p := range []int{-1, 101, 128} {
		_, e := gad.ConfidenceFromPercent(p)
		constructorError(t, e, gad.ErrOutOfRange, "Confidence")
	}
	for n := int32(-64000); n <= 1280000; n++ {
		a := gad.HighAccuracyAltitude(n)
		got, e := gad.HighAccuracyAltitudeFromMetres(a.Metres())
		if e != nil || got != a {
			t.Fatalf("HA altitude %d = %d,%v", n, got, e)
		}
	}
}

func TestScalableUncertaintyDomains(t *testing.T) {
	for _, extended := range []bool{false, true} {
		_, e := gad.ScalableUncertaintyFromMetres(math.NaN(), extended)
		constructorError(t, e, gad.ErrNotFinite, "High accuracy uncertainty")
		_, e = gad.ScalableUncertaintyFromMetres(-1, extended)
		constructorError(t, e, gad.ErrOutOfRange, "High accuracy uncertainty")
		_, e = gad.ScalableUncertaintyFromMetres(math.Inf(-1), extended)
		want := gad.ErrNotFinite
		if extended {
			want = gad.ErrOutOfRange
		}
		constructorError(t, e, want, "High accuracy uncertainty")
		if !extended {
			_, e = gad.ScalableUncertaintyFromMetres(math.Inf(1), extended)
			constructorError(t, e, gad.ErrNotFinite, "High accuracy uncertainty")
		}
	}
	for k := 0; k < 255; k++ {
		lower := (gad.ScalableUncertainty{Extended: true, K: uint8(k)}).Metres()
		got, e := gad.ScalableUncertaintyFromMetres(math.Nextafter(lower, math.Inf(1)), true)
		if e != nil || got.K != uint8(k+1) {
			t.Fatalf("above extended code %d: %+v,%v", k, got, e)
		}
	}
}

// TS 23.032 V19.0.0 §§6.5, 8.11: codes without values are interpretation-only.
func TestNoInformationCodes(t *testing.T) {
	for n := 101; n <= 127; n++ {
		if percent, ok := gad.Confidence(n).Percent(); percent != 0 || ok {
			t.Fatalf("confidence %d = %d,%t; want 0,false", n, percent, ok)
		}
		_, err := gad.ConfidenceFromPercent(n)
		constructorError(t, err, gad.ErrOutOfRange, "Confidence")
	}
	// Zero conveys no information too, but retains its constructor inverse.
	zero, err := gad.ConfidenceFromPercent(0)
	if err != nil || zero != 0 {
		t.Fatalf("zero confidence = %d,%v", zero, err)
	}
	if percent, ok := zero.Percent(); percent != 0 || ok {
		t.Fatalf("zero confidence = %d,%t", percent, ok)
	}
	if gad.UncertaintySpeedUnspecified != 255 {
		t.Fatalf("unspecified code = %d, want 255", gad.UncertaintySpeedUnspecified)
	}
	if speed, ok := gad.UncertaintySpeed(255).Kmh(); speed != 0 || ok {
		t.Fatalf("unspecified speed = %g,%t; want 0,false", speed, ok)
	}
	for _, speed := range []float64{math.Nextafter(254, math.Inf(1)), 254.5, 255, math.MaxFloat64} {
		_, err := gad.UncertaintySpeedFromKmh(speed)
		constructorError(t, err, gad.ErrOutOfRange, "Uncertainty speed")
	}
	highest, err := gad.UncertaintySpeedFromKmh(254)
	if err != nil || highest != 254 {
		t.Fatalf("highest speed = %d,%v", highest, err)
	}
}
