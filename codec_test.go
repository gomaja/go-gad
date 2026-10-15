package gad

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func assertFieldError(t *testing.T, err, errorKind error, field string, octet int) {
	t.Helper()
	var fe *FieldError
	if !errors.Is(err, errorKind) || !errors.As(err, &fe) {
		t.Fatalf("error %v, want *FieldError wrapping %v", err, errorKind)
	}
	if fe.Field != field || fe.Octet != octet || !strings.HasPrefix(err.Error(), "gad: ") {
		t.Fatalf("error = %#v (%v), want %q octet %d", fe, err, field, octet)
	}
	if !strings.Contains(err.Error(), strings.TrimPrefix(errorKind.Error(), "gad: ")) || !strings.Contains(err.Error(), field) {
		t.Fatalf("incomplete message: %v", err)
	}
	if octet > 0 && !strings.Contains(err.Error(), fmt.Sprintf("octet %d", octet)) {
		t.Fatalf("missing octet: %v", err)
	}
}

type shapeCase struct {
	shape  Shape
	n      int
	spares map[int]byte
	angles []int
}

// An independently transcribed layout, TS 23.032 V19.0.0 §7.3 figures.
var shapeCases = []shapeCase{
	{0, 7, map[int]byte{1: 15}, nil},
	{1, 8, map[int]byte{1: 15, 8: 128}, nil},
	{3, 11, map[int]byte{1: 15, 8: 128, 9: 128, 11: 128}, []int{10}},
	{5, 19, nil, nil},
	{8, 9, map[int]byte{1: 15}, nil},
	{9, 14, map[int]byte{1: 15, 10: 128, 11: 128, 13: 128, 14: 128}, []int{12}},
	{10, 13, map[int]byte{1: 15, 10: 128, 13: 128}, []int{11, 12}},
	{11, 13, map[int]byte{1: 15, 13: 128}, []int{12}},
	{12, 18, map[int]byte{1: 15, 10: 192, 16: 128, 18: 128}, []int{15}},
	{13, 13, map[int]byte{1: 15}, []int{12}},
	{14, 18, map[int]byte{1: 15, 10: 192}, []int{15}},
}

func zeroWire(c shapeCase) []byte {
	b := make([]byte, c.n)
	b[0] = byte(c.shape) << 4
	if c.shape == 5 {
		b[0] |= 3
	}
	return b
}

// TS 23.032 V19.0.0 §7.3: exact lengths, spare bits and angular codes.
func TestStrictShapes(t *testing.T) {
	for _, c := range shapeCases {
		t.Run(c.shape.String(), func(t *testing.T) {
			b := zeroWire(c)
			v, err := Decode(b)
			if err != nil {
				t.Fatal(err)
			}
			got, err := v.Encode()
			if err != nil || !bytes.Equal(got, b) {
				t.Fatalf("Encode=%x,%v", got, err)
			}
			if v.Shape() != c.shape {
				t.Fatalf("Shape=%v", v.Shape())
			}
			for _, n := range []int{c.n - 1, c.n + 1} {
				wire := make([]byte, n)
				copy(wire, b)
				_, err := Decode(wire)
				assertFieldError(t, err, ErrLength, c.shape.String(), min(n, c.n)+1)
				if Validate(wire).Error() != err.Error() {
					t.Fatal("Validate differs")
				}
			}
			for o, mask := range c.spares {
				for bit := byte(1); bit != 0; bit <<= 1 {
					if mask&bit == 0 {
						continue
					}
					wire := bytes.Clone(b)
					wire[o-1] |= bit
					_, err := Decode(wire)
					var fe *FieldError
					if !errors.Is(err, ErrSpareBit) || !errors.As(err, &fe) || fe.Octet != o {
						t.Fatalf("octet %d mask %x: %v", o, bit, err)
					}
					field := "Spare"
					switch c.shape {
					case 1:
						if o == 8 {
							field = "Uncertainty"
						}
					case 3:
						field = map[int]string{1: "Spare", 8: "Uncertainty semi-major", 9: "Uncertainty semi-minor", 11: "Confidence"}[o]
					case 9:
						field = map[int]string{1: "Spare", 10: "Uncertainty semi-major", 11: "Uncertainty semi-minor", 13: "Uncertainty altitude", 14: "Confidence"}[o]
					case 10:
						field = map[int]string{1: "Spare", 10: "Uncertainty radius", 13: "Confidence"}[o]
					case 11:
						if o == 13 {
							field = "Confidence"
						}
					case 12:
						if o == 16 {
							field = "Horizontal confidence"
						}
						if o == 18 {
							field = "Vertical confidence"
						}
					}
					assertFieldError(t, err, ErrSpareBit, field, o)
				}
			}
			for _, o := range c.angles {
				for _, n := range []byte{179, 180, 255} {
					wire := bytes.Clone(b)
					wire[o-1] = n
					v, err := Decode(wire)
					if n == 179 {
						if err != nil {
							t.Fatal(err)
						}
						got, err := v.Encode()
						if err != nil || !bytes.Equal(got, wire) {
							t.Fatalf("angle roundtrip %x %v", got, err)
						}
					} else {
						name := "Orientation of major axis"
						if c.shape == 10 {
							name = "Offset angle"
							if o == 12 {
								name = "Included angle"
							}
						}
						assertFieldError(t, err, ErrOutOfRange, name, o)
					}
				}
			}
		})
	}
	for _, n := range []byte{2, 4, 6, 7, 15} {
		_, err := Decode([]byte{n << 4})
		assertFieldError(t, err, ErrShapeTypeInvalid, "Type of shape", 1)
	}
	_, err := Decode(nil)
	assertFieldError(t, err, ErrLength, "Geographical information", 1)
}

// TS 23.032 V19.0.0 §§6.3a,7.3.4,7.3.6a,7.3.6b.
func TestPolygonAndHighAltitudeBoundaries(t *testing.T) {
	for _, n := range []int{2, 3, 15, 16} {
		p := Polygon{Points: make([]Point, n)}
		b, err := p.Encode()
		if n == 2 || n == 16 {
			assertFieldError(t, err, ErrOutOfRange, "Number of points", 1)
		} else {
			got, e := Decode(b)
			if e != nil || !reflect.DeepEqual(got, p) {
				t.Fatalf("polygon %d: %v %v", n, got, e)
			}
		}
		if n < 16 {
			wire := make([]byte, 1+6*n)
			wire[0] = 0x50 | byte(n)
			_, err := Decode(wire)
			if n == 2 {
				assertFieldError(t, err, ErrOutOfRange, "Number of points", 1)
			} else if err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, shape := range []Shape{12, 14} {
		for _, a := range []int32{-64001, -64000, 1280000, 1280001} {
			b := make([]byte, 18)
			b[0] = byte(shape) << 4
			b[9] = byte(uint32(a)>>16) & 63
			b[10] = byte(a >> 8)
			b[11] = byte(a)
			_, err := Decode(b)
			var v Description = HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{Altitude: HighAccuracyAltitude(a)}
			if shape == 14 {
				v = HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{Altitude: HighAccuracyAltitude(a)}
			}
			encoded, e := v.Encode()
			if a < -64000 || a > 1280000 {
				assertFieldError(t, err, ErrOutOfRange, "Altitude", 10)
				assertFieldError(t, e, ErrOutOfRange, "Altitude", 10)
			} else if err != nil || e != nil || !bytes.Equal(encoded, b) {
				t.Fatalf("%d %d: %v %v %x", shape, a, err, e, encoded)
			}
		}
	}
}

// TS 23.032 V19.0.0 §6.5: every seven-bit confidence is lossless.
func TestConfidenceWireCodes(t *testing.T) {
	for _, c := range shapeCases {
		positions := map[Shape][]int{3: {11}, 9: {14}, 10: {13}, 11: {13}, 12: {16, 18}, 13: {13}, 14: {16, 18}}[c.shape]
		for _, o := range positions {
			for n := byte(0); n <= 127; n++ {
				b := zeroWire(c)
				b[o-1] = n
				v, err := Decode(b)
				if err != nil {
					t.Fatal(err)
				}
				got, err := v.Encode()
				if err != nil || !bytes.Equal(got, b) {
					t.Fatalf("confidence %d: %x %v", n, got, err)
				}
			}
		}
	}
}

// TS 23.032 V19.0.0 §§6–7: coded values must fit their fields.
func TestEncodeInvalidFields(t *testing.T) {
	invalidPoint := Point{Latitude: Latitude{N: 1 << 23}}
	for _, tc := range []struct {
		v        Description
		field    string
		o        int
		sentinel error
	}{
		{EllipsoidPoint{Point: invalidPoint}, "Latitude", 2, ErrOutOfRange},
		{EllipsoidPoint{Point: Point{Longitude: 1 << 23}}, "Longitude", 5, ErrOutOfRange},
		{EllipsoidPoint{Point: Point{Longitude: -(1 << 23) - 1}}, "Longitude", 5, ErrOutOfRange},
		{Polygon{Points: []Point{{}, {}, invalidPoint}}, "Latitude", 14, ErrOutOfRange},
		{Polygon{Points: []Point{{}, {}, {Longitude: 1 << 23}}}, "Longitude", 17, ErrOutOfRange},
		{EllipsoidPointUncertaintyCircle{Uncertainty: 128}, "Uncertainty", 8, ErrOutOfRange},
		{EllipsoidPointUncertaintyEllipse{SemiMajor: 128}, "Uncertainty semi-major", 8, ErrOutOfRange},
		{EllipsoidPointUncertaintyEllipse{SemiMinor: 128}, "Uncertainty semi-minor", 9, ErrOutOfRange},
		{EllipsoidPointUncertaintyEllipse{Orientation: 180}, "Orientation of major axis", 10, ErrOutOfRange},
		{EllipsoidPointUncertaintyEllipse{Confidence: 128}, "Confidence", 11, ErrOutOfRange},
		{EllipsoidPointAltitude{Altitude: Altitude{N: 32768}}, "Altitude", 8, ErrOutOfRange},
		{EllipsoidPointAltitudeUncertaintyEllipsoid{SemiMajor: 128}, "Uncertainty semi-major", 10, ErrOutOfRange},
		{EllipsoidPointAltitudeUncertaintyEllipsoid{SemiMinor: 128}, "Uncertainty semi-minor", 11, ErrOutOfRange},
		{EllipsoidPointAltitudeUncertaintyEllipsoid{Orientation: 180}, "Orientation of major axis", 12, ErrOutOfRange},
		{EllipsoidPointAltitudeUncertaintyEllipsoid{UncertaintyAltitude: 128}, "Uncertainty altitude", 13, ErrOutOfRange},
		{EllipsoidPointAltitudeUncertaintyEllipsoid{Confidence: 128}, "Confidence", 14, ErrOutOfRange},
		{EllipsoidArc{UncertaintyRadius: 128}, "Uncertainty radius", 10, ErrOutOfRange},
		{EllipsoidArc{OffsetAngle: 180}, "Offset angle", 11, ErrOutOfRange},
		{EllipsoidArc{IncludedAngle: 180}, "Included angle", 12, ErrOutOfRange},
		{EllipsoidArc{Confidence: 128}, "Confidence", 13, ErrOutOfRange},
		{HighAccuracyEllipsoidPointUncertaintyEllipse{Orientation: 180}, "Orientation of major axis", 12, ErrOutOfRange},
		{HighAccuracyEllipsoidPointUncertaintyEllipse{Confidence: 128}, "Confidence", 13, ErrOutOfRange},
		{HighAccuracyEllipsoidPointScalableUncertaintyEllipse{SemiMinor: ScalableUncertainty{Extended: true}}, "Uncertainty semi-minor", 11, ErrUncertaintyRange},
		{HighAccuracyEllipsoidPointScalableUncertaintyEllipse{Orientation: 180}, "Orientation of major axis", 12, ErrOutOfRange},
		{HighAccuracyEllipsoidPointScalableUncertaintyEllipse{Confidence: 128}, "Confidence", 13, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{Orientation: 180}, "Orientation of major axis", 15, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{HorizontalConfidence: 128}, "Horizontal confidence", 16, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{VerticalConfidence: 128}, "Vertical confidence", 18, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{SemiMinor: ScalableUncertainty{Extended: true}}, "Uncertainty semi-minor", 14, ErrUncertaintyRange},
		{HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{Orientation: 180}, "Orientation of major axis", 15, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{HorizontalConfidence: 128}, "Horizontal confidence", 16, ErrOutOfRange},
		{HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{VerticalConfidence: 128}, "Vertical confidence", 18, ErrOutOfRange},
	} {
		t.Run(fmt.Sprintf("%T/%s", tc.v, tc.field), func(t *testing.T) {
			b, err := tc.v.Encode()
			if b != nil {
				t.Fatalf("invalid encode %x", b)
			}
			assertFieldError(t, err, tc.sentinel, tc.field, tc.o)
		})
	}
}

// TS 23.032 V19.0.0 §7.3: checks follow octet order, spare before value.
func TestErrorOrder(t *testing.T) {
	for _, tc := range []struct {
		b        []byte
		sentinel error
		field    string
		o        int
	}{
		{[]byte{0xff}, ErrShapeTypeInvalid, "Type of shape", 1},
		{[]byte{0x52}, ErrOutOfRange, "Number of points", 1},
		{[]byte{1}, ErrLength, "ellipsoid point", 2},
		{[]byte{0x31, 0, 0, 0, 0, 0, 0, 128, 128, 180, 128}, ErrSpareBit, "Spare", 1},
		{[]byte{0x30, 0, 0, 0, 0, 0, 0, 0, 0, 180, 128}, ErrOutOfRange, "Orientation of major axis", 10},
		{[]byte{0xc0, 0, 0, 0, 0, 0, 0, 0, 0, 0xc0, 0, 0, 0, 0, 180, 128, 0, 128}, ErrSpareBit, "Spare", 10},
	} {
		_, err := Decode(tc.b)
		assertFieldError(t, err, tc.sentinel, tc.field, tc.o)
	}
}

// TS 23.032 V19.0.0 §7.2 Table 2a and §8.6 Table 3.
func TestTypeStrings(t *testing.T) {
	for shape, want := range map[Shape]string{
		0:  "ellipsoid point",
		1:  "ellipsoid point with uncertainty circle",
		3:  "ellipsoid point with uncertainty ellipse",
		5:  "polygon",
		8:  "ellipsoid point with altitude",
		9:  "ellipsoid point with altitude and uncertainty ellipsoid",
		10: "ellipsoid arc",
		11: "high accuracy ellipsoid point with uncertainty ellipse",
		12: "high accuracy ellipsoid point with altitude and uncertainty ellipsoid",
		13: "high accuracy ellipsoid point with scalable uncertainty ellipse",
		14: "high accuracy ellipsoid point with altitude and scalable uncertainty ellipsoid",
	} {
		if got := shape.String(); got != want {
			t.Errorf("Shape(%d).String()=%q, want %q", shape, got, want)
		}
	}
	for _, n := range []int{-1, 2, 4, 6, 7, 15, 16} {
		if Shape(n).String() != fmt.Sprintf("Shape(%d)", n) {
			t.Fatal(n)
		}
	}
	for typ, want := range []string{
		"horizontal velocity",
		"horizontal with vertical velocity",
		"horizontal velocity with uncertainty",
		"horizontal with vertical velocity and uncertainty",
	} {
		if got := VelocityType(typ).String(); got != want {
			t.Errorf("VelocityType(%d).String()=%q, want %q", typ, got, want)
		}
	}
	for _, n := range []int{-1, 4, 15, 16} {
		if VelocityType(n).String() != fmt.Sprintf("VelocityType(%d)", n) {
			t.Fatal(n)
		}
	}
}
