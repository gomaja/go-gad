package gad

import (
	"errors"
	"testing"
)

// TS 23.032 V19.0.0 §§6–8: locate errors without losing their sentinel.
func TestFieldErrorMessage(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value FieldError
		want  string
	}{
		{"field octet and detail", FieldError{Field: "Orientation of major axis", Octet: 10, Err: ErrOutOfRange, Detail: "180, want 0..179"}, "gad: Orientation of major axis (octet 10): value out of range: 180, want 0..179"},
		{"constructor", FieldError{Field: "Latitude", Err: ErrNotFinite, Detail: "NaN"}, "gad: Latitude: value not finite: NaN"},
		{"no detail", FieldError{Field: "Spare", Octet: 1, Err: ErrSpareBit}, "gad: Spare (octet 1): spare bit set"},
		{"nil sentinel", FieldError{Field: "Latitude", Octet: 2}, "gad: Latitude (octet 2)"},
		{"nil sentinel with detail", FieldError{Field: "Latitude", Detail: "no cause"}, "gad: Latitude: no cause"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.value.Error(); got != tc.want {
				t.Fatalf("Error()=%q, want %q", got, tc.want)
			}
			if tc.value.Unwrap() != tc.value.Err {
				t.Fatalf("Unwrap()=%v, want %v", tc.value.Unwrap(), tc.value.Err)
			}
			if tc.value.Err != nil && !errors.Is(&tc.value, tc.value.Err) {
				t.Fatal("lost sentinel")
			}
		})
	}
}

// TS 23.032 V19.0.0 §§6–8: real failures carry their field, octet and bounds.
func TestCodingErrorMessages(t *testing.T) {
	highAltitude := func(code uint32, shape Shape) []byte {
		b := make([]byte, 18)
		b[0] = byte(shape) << 4
		b[9], b[10], b[11] = byte(code>>16)&63, byte(code>>8), byte(code)
		return b
	}
	const ha, scalable = ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid, ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid
	errorOf := func(_ any, err error) error { return err }
	for _, tc := range []struct {
		name string
		err  error
		want string
	}{
		{"decode altitude above", errorOf(Decode(highAltitude(1280001, ha))), "gad: Altitude (octet 10): value out of range: 1280001, want -64000..1280000"},
		{"decode altitude below", errorOf(Decode(highAltitude(1<<22-64001, ha))), "gad: Altitude (octet 10): value out of range: -64001, want -64000..1280000"},
		{"decode scalable altitude below", errorOf(Decode(highAltitude(1<<22-64001, scalable))), "gad: Altitude (octet 10): value out of range: -64001, want -64000..1280000"},
		{"encode altitude below", errorOf(HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{Altitude: -64001}.Encode()), "gad: Altitude (octet 10): value out of range: -64001, want -64000..1280000"},
		{"encode longitude", errorOf(EllipsoidPoint{Point: Point{Longitude: 1 << 23}}.Encode()), "gad: Longitude (octet 5): value out of range: 8388608, want -8388608..8388607"},
		{"length", errorOf(Decode(make([]byte, 12))), "gad: ellipsoid point (octet 8): invalid length: 12 octets, want 7"},
		{"arc length", errorOf(Decode(append([]byte{0xa0}, make([]byte, 11)...))), "gad: ellipsoid arc (octet 13): invalid length: 12 octets, want 13"},
		{"header spare", errorOf(Decode([]byte{0x01, 0, 0, 0, 0, 0, 0})), "gad: Spare (octet 1): spare bit set"},
		{"reserved shape", errorOf(Decode([]byte{0x20})), "gad: Type of shape (octet 1): reserved type of shape: Shape(2)"},
		{"bearing", errorOf(DecodeVelocity([]byte{0x01, 0x68, 0, 0})), "gad: Bearing (octet 1): value out of range: 360, want 0..359"},
		{"constructor", errorOf(LatitudeFromDegrees(91)), "gad: Latitude: value out of range: 91, want [-90,90]"},
	} {
		if tc.err == nil || tc.err.Error() != tc.want {
			t.Errorf("%s: %v, want %q", tc.name, tc.err, tc.want)
		}
	}
}
