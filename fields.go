package gad

import (
	"fmt"
	"math"
	"sort"
)

// Uncertainty codes horizontal uncertainty per TS 23.032 V19.0.0 §6.2.
type Uncertainty uint8

// Metres gives the uncertainty value per TS 23.032 V19.0.0 §6.2.
func (u Uncertainty) Metres() float64 { return 10 * (math.Pow(1.1, float64(u)) - 1) }

// UncertaintyFromMetres rounds up per TS 23.032 V19.0.0 §6.2, whose formula defines values, not bins.
// Rounding up never understates the supplied uncertainty.
func UncertaintyFromMetres(r float64) (Uncertainty, error) {
	k, e := uncertaintyCode("Uncertainty", r, 127, func(k int) float64 { return Uncertainty(k).Metres() })
	return Uncertainty(k), e
}

// HighAccuracyUncertainty codes uncertainty per TS 23.032 V19.0.0 §6.2a.
type HighAccuracyUncertainty uint8

// Metres gives the uncertainty value per TS 23.032 V19.0.0 §6.2a.
func (u HighAccuracyUncertainty) Metres() float64 { return 0.3 * (math.Pow(1.02, float64(u)) - 1) }

// HighAccuracyUncertaintyFromMetres rounds up per TS 23.032 V19.0.0 §6.2a, whose formula defines values, not bins.
// Rounding up never understates the supplied uncertainty.
func HighAccuracyUncertaintyFromMetres(r float64) (HighAccuracyUncertainty, error) {
	k, e := uncertaintyCode("High accuracy uncertainty", r, 255, func(k int) float64 { return HighAccuracyUncertainty(k).Metres() })
	return HighAccuracyUncertainty(k), e
}

// ScalableUncertainty codes a selected uncertainty range per TS 23.032 V19.0.0 §§6.2a–6.2b.
type ScalableUncertainty struct {
	Extended bool
	K        uint8
}

// Metres gives the uncertainty value per TS 23.032 V19.0.0 §§6.2a–6.2b.
// Extended code 255 denotes greater than 200 metres and returns positive infinity.
func (u ScalableUncertainty) Metres() float64 {
	if !u.Extended {
		return HighAccuracyUncertainty(u.K).Metres()
	}
	if u.K == 254 {
		return 200
	}
	if u.K == 255 {
		return math.Inf(1)
	}
	return 0.3 * (math.Pow(1.02594, float64(u.K)) - 1)
}

// ScalableUncertaintyFromMetres rounds up per TS 23.032 V19.0.0 §§6.2a–6.2b.
// These formulae define values, not bins; rounding up never understates uncertainty.
// Positive infinity is accepted only in the extended range, as code 255.
func ScalableUncertaintyFromMetres(r float64, extended bool) (ScalableUncertainty, error) {
	if !extended {
		v, e := HighAccuracyUncertaintyFromMetres(r)
		if e != nil {
			return ScalableUncertainty{}, e
		}
		return ScalableUncertainty{K: uint8(v)}, nil
	}
	if math.IsNaN(r) {
		return ScalableUncertainty{}, fieldError("High accuracy uncertainty", 0, ErrNotFinite, fmt.Sprint(r))
	}
	if r < 0 {
		return ScalableUncertainty{}, fieldError("High accuracy uncertainty", 0, ErrOutOfRange, fmt.Sprintf("%g, want >= 0", r))
	}
	k := sort.Search(255, func(k int) bool { return (ScalableUncertainty{Extended: true, K: uint8(k)}).Metres() >= r })
	return ScalableUncertainty{Extended: true, K: uint8(k)}, nil
}

// UncertaintyAltitude codes vertical uncertainty per TS 23.032 V19.0.0 §6.4.
type UncertaintyAltitude uint8

// Metres gives the uncertainty altitude value per TS 23.032 V19.0.0 §6.4.
func (u UncertaintyAltitude) Metres() float64 { return 45 * (math.Pow(1.025, float64(u)) - 1) }

// UncertaintyAltitudeFromMetres rounds up per TS 23.032 V19.0.0 §6.4, whose formula defines values, not bins.
// Rounding up never understates the supplied uncertainty.
func UncertaintyAltitudeFromMetres(r float64) (UncertaintyAltitude, error) {
	k, e := uncertaintyCode("Uncertainty altitude", r, 127, func(k int) float64 { return UncertaintyAltitude(k).Metres() })
	return UncertaintyAltitude(k), e
}

// Confidence codes a confidence percentage per TS 23.032 V19.0.0 §6.5.
// The clause's upper bound 128 exceeds the seven-bit field's maximum 127.
type Confidence uint8

// Percent interprets confidence per TS 23.032 V19.0.0 §6.5.
// Codes 0 and 101..127 convey no information and return (0,false).
// Codes 101..127 have no constructor inverse; Decode and Encode preserve them.
// Code 0 retains its inverse: ConfidenceFromPercent(0) returns Confidence(0).
func (c Confidence) Percent() (int, bool) {
	if c >= 1 && c <= 100 {
		return int(c), true
	}
	return 0, false
}

// ConfidenceFromPercent codes 0..100 percent per TS 23.032 V19.0.0 §6.5; zero means no information.
// It never produces codes 101..127, which Percent interprets as no information.
func ConfidenceFromPercent(p int) (Confidence, error) {
	if p < 0 || p > 100 {
		return 0, fieldError("Confidence", 0, ErrOutOfRange, fmt.Sprintf("%d, want 0..100", p))
	}
	return Confidence(p), nil
}

// Orientation codes whole degrees from north per TS 23.032 V19.0.0 §7.3.3.
type Orientation uint8

// Degrees gives orientation per TS 23.032 V19.0.0 §7.3.3.
func (o Orientation) Degrees() float64 { return float64(o) }

// OrientationFromDegrees uses floor bins on [0,180) per TS 23.032 V19.0.0 §7.3.3.
func OrientationFromDegrees(a float64) (Orientation, error) {
	if e := boundedField("Orientation of major axis", a, 180, false, false); e != nil {
		return 0, e
	}
	return Orientation(math.Floor(a)), nil
}

// Altitude codes height or depth in metres per TS 23.032 V19.0.0 §6.3.
type Altitude struct {
	Depth bool
	N     uint16
}

// Metres returns the lower magnitude bin edge per TS 23.032 V19.0.0 §6.3, preserving negative zero.
func (a Altitude) Metres() float64 {
	x := float64(a.N)
	if a.Depth {
		return -x
	}
	return x
}

// AltitudeFromMetres uses floor magnitude bins per TS 23.032 V19.0.0 §6.3; the highest bin saturates.
func AltitudeFromMetres(a float64) (Altitude, error) {
	if e := finiteField("Altitude", a); e != nil {
		return Altitude{}, e
	}
	n := math.Floor(math.Abs(a))
	if n > 32767 {
		n = 32767
	}
	return Altitude{Depth: math.Signbit(a), N: uint16(n)}, nil
}

// HighAccuracyAltitude codes altitude in 1/128 metre units per TS 23.032 V19.0.0 §6.3a.
type HighAccuracyAltitude int32

// Metres gives altitude per TS 23.032 V19.0.0 §6.3a.
func (a HighAccuracyAltitude) Metres() float64 { return float64(a) / 128 }

// HighAccuracyAltitudeFromMetres uses floor bins with codes -64000..1280000 per TS 23.032 V19.0.0 §6.3a.
func HighAccuracyAltitudeFromMetres(a float64) (HighAccuracyAltitude, error) {
	if e := finiteField("Altitude", a); e != nil {
		return 0, e
	}
	n := math.Floor(a * 128)
	if n < -64000 || n > 1280000 {
		return 0, fieldError("Altitude", 0, ErrOutOfRange, fmt.Sprintf("%g, want code -64000..1280000", a))
	}
	return HighAccuracyAltitude(n), nil
}

// InnerRadius codes a radius in five-metre steps per TS 23.032 V19.0.0 §6.6.
type InnerRadius uint16

// Metres returns the lower radius bin edge per TS 23.032 V19.0.0 §6.6.
func (r InnerRadius) Metres() float64 { return 5 * float64(r) }

// InnerRadiusFromMetres uses floor bins per TS 23.032 V19.0.0 §6.6; the highest bin saturates.
func InnerRadiusFromMetres(r float64) (InnerRadius, error) {
	if e := nonnegativeField("Inner radius", r); e != nil {
		return 0, e
	}
	n := math.Floor(r / 5)
	if n > 65535 {
		n = 65535
	}
	return InnerRadius(n), nil
}

// OffsetAngle codes the arc offset angle in two-degree steps per TS 23.032 V19.0.0 §6.7.
type OffsetAngle uint8

// Degrees returns the lower offset angle bin edge per TS 23.032 V19.0.0 §6.7.
func (a OffsetAngle) Degrees() float64 { return 2 * float64(a) }

// OffsetAngleFromDegrees uses floor bins on [0,360) per TS 23.032 V19.0.0 §6.7.
func OffsetAngleFromDegrees(a float64) (OffsetAngle, error) {
	if e := boundedField("Offset angle", a, 360, false, false); e != nil {
		return 0, e
	}
	return OffsetAngle(math.Floor(a / 2)), nil
}

// IncludedAngle codes the arc included angle in two-degree steps per TS 23.032 V19.0.0 §6.7.
type IncludedAngle uint8

// Degrees returns the upper included angle bin edge per TS 23.032 V19.0.0 §6.7.
func (a IncludedAngle) Degrees() float64 { return 2 * (float64(a) + 1) }

// IncludedAngleFromDegrees uses upper-inclusive bins on (0,360] per TS 23.032 V19.0.0 §6.7.
func IncludedAngleFromDegrees(a float64) (IncludedAngle, error) {
	if e := boundedField("Included angle", a, 360, true, true); e != nil {
		return 0, e
	}
	n := math.Ceil(a/2) - 1
	if n < 0 {
		n = 0
	}
	return IncludedAngle(n), nil
}

// Bearing codes whole degrees clockwise from north per TS 23.032 V19.0.0 §8.8.
// The clause omits code 511 from its unused range; it also lies outside [0,360).
type Bearing uint16

// Degrees returns the lower bearing bin edge per TS 23.032 V19.0.0 §8.8.
func (b Bearing) Degrees() float64 { return float64(b) }

// BearingFromDegrees uses floor bins on [0,360) per TS 23.032 V19.0.0 §8.8.
func BearingFromDegrees(b float64) (Bearing, error) {
	if e := boundedField("Bearing", b, 360, false, false); e != nil {
		return 0, e
	}
	return Bearing(math.Floor(b)), nil
}

// HorizontalSpeed codes speed in kilometres per hour per TS 23.032 V19.0.0 §8.7.
type HorizontalSpeed uint16

// Kmh gives speed per TS 23.032 V19.0.0 §8.7.
func (h HorizontalSpeed) Kmh() float64 { return float64(h) }

// HorizontalSpeedFromKmh rounds half up per TS 23.032 V19.0.0 §8.7; the highest bin saturates.
func HorizontalSpeedFromKmh(h float64) (HorizontalSpeed, error) {
	n, e := speedCode("Horizontal speed", h, 65535)
	return HorizontalSpeed(n), e
}

// VerticalSpeed codes speed in kilometres per hour per TS 23.032 V19.0.0 §8.9.
type VerticalSpeed uint8

// Kmh gives speed per TS 23.032 V19.0.0 §8.9.
func (v VerticalSpeed) Kmh() float64 { return float64(v) }

// VerticalSpeedFromKmh rounds half up per TS 23.032 V19.0.0 §8.9; the highest bin saturates.
func VerticalSpeedFromKmh(v float64) (VerticalSpeed, error) {
	n, e := speedCode("Vertical speed", v, 255)
	return VerticalSpeed(n), e
}

// UncertaintySpeed codes speed uncertainty per TS 23.032 V19.0.0 §8.11.
type UncertaintySpeed uint8

// UncertaintySpeedUnspecified denotes unspecified speed uncertainty per TS 23.032 V19.0.0 §8.11.
const UncertaintySpeedUnspecified UncertaintySpeed = 255

// Kmh interprets speed uncertainty per TS 23.032 V19.0.0 §8.11.
// Codes 0..254 have a constructor inverse. Code 255 returns (0,false), has no
// constructor inverse, and is preserved by DecodeVelocity and Encode.
func (s UncertaintySpeed) Kmh() (float64, bool) {
	if s == UncertaintySpeedUnspecified {
		return 0, false
	}
	return float64(s), true
}

// UncertaintySpeedFromKmh rounds up per TS 23.032 V19.0.0 §8.11, which defines values, not bins.
// Rounding up never understates the supplied uncertainty. The largest result is
// 254; this constructor never produces UncertaintySpeedUnspecified (255).
func UncertaintySpeedFromKmh(s float64) (UncertaintySpeed, error) {
	if e := nonnegativeField("Uncertainty speed", s); e != nil {
		return 0, e
	}
	n := math.Ceil(s)
	if n > 254 {
		return 0, fieldError("Uncertainty speed", 0, ErrOutOfRange, fmt.Sprintf("%g, want <= 254", s))
	}
	return UncertaintySpeed(n), nil
}

func finiteField(field string, x float64) error {
	if math.IsNaN(x) || math.IsInf(x, 0) {
		return fieldError(field, 0, ErrNotFinite, fmt.Sprint(x))
	}
	return nil
}

func nonnegativeField(field string, x float64) error {
	if e := finiteField(field, x); e != nil {
		return e
	}
	if x < 0 {
		return fieldError(field, 0, ErrOutOfRange, fmt.Sprintf("%g, want >= 0", x))
	}
	return nil
}

func boundedField(field string, x, max float64, lowerOpen, upperClosed bool) error {
	if e := nonnegativeField(field, x); e != nil {
		return e
	}
	if (lowerOpen && x == 0) || x > max || (!upperClosed && x == max) {
		lower, upper := "[", ")"
		if lowerOpen {
			lower = "("
		}
		if upperClosed {
			upper = "]"
		}
		return fieldError(field, 0, ErrOutOfRange, fmt.Sprintf("%g, want %s0,%g%s", x, lower, max, upper))
	}
	return nil
}

func uncertaintyCode(field string, r float64, max int, metres func(int) float64) (int, error) {
	if e := nonnegativeField(field, r); e != nil {
		return 0, e
	}
	k := sort.Search(max+1, func(k int) bool { return metres(k) >= r })
	if k > max {
		return 0, fieldError(field, 0, ErrOutOfRange, fmt.Sprintf("%g, want <= %g", r, metres(max)))
	}
	return k, nil
}

func speedCode(field string, h, max float64) (float64, error) {
	if e := nonnegativeField(field, h); e != nil {
		return 0, e
	}
	if h >= max-0.5 {
		return max, nil
	}
	n := math.Floor(h)
	if h-n >= 0.5 {
		n++
	}
	return n, nil
}
