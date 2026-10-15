package gad

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"testing"
)

// TS 23.032 V19.0.0 §6.1.
func TestGeoEncode_RejectsOutOfRangeLatitude(t *testing.T) {
	for _, lat := range []float64{-90.01, 90.01, 200, -500} {
		gi := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: lat, Longitude: 0}
		if _, err := gi.Encode(); err == nil {
			t.Errorf("Latitude=%v: expected error, got nil", lat)
		}
	}
}

// TS 23.032 V19.0.0 §6.1.
func TestGeoEncode_RejectsOutOfRangeLongitude(t *testing.T) {
	for _, lon := range []float64{-180.01, 180.01, 500, -500, 180} {
		gi := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: 0, Longitude: lon}
		if _, err := gi.Encode(); err == nil {
			t.Errorf("Longitude=%v: expected error, got nil", lon)
		}
	}
}

// TS 23.032 V19.0.0 §6.1.
func TestGeoEncode_AcceptsBoundaryLatLon(t *testing.T) {
	cases := []struct {
		lat, lon float64
	}{
		{0, 0},
		{90, 0},
		{-90, 0},
		{89.9999, 179.9999},
		{-89.9999, -180},
	}
	for _, tc := range cases {
		gi := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: tc.lat, Longitude: tc.lon}
		if _, err := gi.Encode(); err != nil {
			t.Errorf("Encode(lat=%v, lon=%v): unexpected error: %v", tc.lat, tc.lon, err)
		}
	}
}

// TS 23.032 V19.0.0 §6.1.
func TestGeoEncode_AcceptsULPInsideBoundary(t *testing.T) {
	for _, tc := range []struct {
		lat, lon float64
		want     []byte
	}{
		{math.Nextafter(90, 0), 0, []byte{0, 0x7f, 0xff, 0xff, 0, 0, 0}},
		{math.Nextafter(-90, 0), 0, []byte{0, 0xff, 0xff, 0xff, 0, 0, 0}},
		{0, math.Nextafter(180, 0), []byte{0, 0, 0, 0, 0x7f, 0xff, 0xff}},
		{0, math.Nextafter(-180, 0), []byte{0, 0, 0, 0, 0x80, 0, 0}},
	} {
		gi := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: tc.lat, Longitude: tc.lon}
		got, err := gi.Encode()
		if err != nil {
			t.Fatalf("Encode(%v, %v): %v", tc.lat, tc.lon, err)
		}
		if !bytes.Equal(got, tc.want) {
			t.Errorf("Encode(%v, %v) = %x, want %x", tc.lat, tc.lon, got, tc.want)
		}
	}
}

// TS 23.032 V19.0.0 §6.1.
func TestGeoEncode_LonNegative180RoundTrips(t *testing.T) {
	gi := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: 0, Longitude: -180}
	data, err := gi.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	got, err := Decode(data)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if got.Longitude != -180 {
		t.Errorf("Longitude round-trip: got %v, want -180", got.Longitude)
	}
}

// TS 23.032 V19.0.0 §§6.3, 7.3.6.
func TestGeoEncode_AltitudeRangeCheck(t *testing.T) {
	cases := []struct {
		name    string
		alt     int16
		wantErr bool
	}{
		{"zero", 0, false},
		{"max valid", 32767, false},
		{"min valid", -32767, false},
		{"MinInt16 rejected", math.MinInt16, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			alt := tc.alt
			gi := &Description{
				ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
				Latitude:  0, Longitude: 0,
				Altitude:             &alt,
				UncertaintySemiMajor: u8(1),
				UncertaintySemiMinor: u8(1),
				AngleMajorAxis:       u8(0),
				UncertaintyAltitude:  u8(1),
				Confidence:           u8(50),
			}
			_, err := gi.Encode()
			if tc.wantErr && err == nil {
				t.Errorf("alt=%d: expected error, got nil", tc.alt)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("alt=%d: unexpected error: %v", tc.alt, err)
			}
		})
	}
}

// TS 23.032 V19.0.0 §§6.3, 7.3.6.
func TestGeoEncode_AltitudeBoundaryRoundTrip(t *testing.T) {
	for _, alt := range []int16{32767, -32767, 0, 1, -1, 100, -100} {
		a := alt
		gi := &Description{
			ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
			Latitude:  0, Longitude: 0,
			Altitude:             &a,
			UncertaintySemiMajor: u8(1),
			UncertaintySemiMinor: u8(1),
			AngleMajorAxis:       u8(0),
			UncertaintyAltitude:  u8(1),
			Confidence:           u8(50),
		}
		data, err := gi.Encode()
		if err != nil {
			t.Fatalf("alt=%d: Encode error: %v", alt, err)
		}
		got, err := Decode(data)
		if err != nil {
			t.Fatalf("alt=%d: Decode error: %v", alt, err)
		}
		if got.Altitude == nil || *got.Altitude != alt {
			t.Errorf("alt=%d: round-trip got %v", alt, got.Altitude)
		}
	}
}

// TS 23.032 V19.0.0 §7.3.
func TestGeoEncode_RejectsMissingRequiredFields(t *testing.T) {
	cases := []struct {
		name string
		gi   *Description
	}{
		{
			"EllipsoidPointUncertainty missing UncertaintyCode",
			&Description{ShapeType: ShapeEllipsoidPointUncertainty, Latitude: 0, Longitude: 0},
		},
		{
			"EllipsoidPointUncertaintyEllipse missing SemiMajor",
			&Description{
				ShapeType: ShapeEllipsoidPointUncertaintyEllipse,
				Latitude:  0, Longitude: 0,
				UncertaintySemiMinor: u8(1), AngleMajorAxis: u8(0), Confidence: u8(50),
			},
		},
		{
			"EllipsoidPointAltitude missing Altitude",
			&Description{
				ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
				Latitude:  0, Longitude: 0,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(0), UncertaintyAltitude: u8(1), Confidence: u8(50),
			},
		},
		{
			"EllipsoidArc missing InnerRadius",
			&Description{
				ShapeType: ShapeEllipsoidArc,
				Latitude:  0, Longitude: 0,
				UncertaintyRadius: u8(1), OffsetAngle: u8(0),
				IncludedAngle: u8(90), Confidence: u8(50),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.gi.Encode(); err == nil {
				t.Errorf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

// TS 23.032 V19.0.0 §7.3.
func TestGeoEncode_ValidShapesRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		gi   *Description
	}{
		{
			"EllipsoidPoint",
			&Description{ShapeType: ShapeEllipsoidPoint, Latitude: 22.63, Longitude: 113.03},
		},
		{
			"EllipsoidPointUncertainty",
			&Description{
				ShapeType: ShapeEllipsoidPointUncertainty,
				Latitude:  22.63, Longitude: 113.03,
				UncertaintyCode: u8(10),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.gi.Encode()
			if err != nil {
				t.Fatalf("Encode: %v", err)
			}
			got, err := Decode(data)
			if err != nil {
				t.Fatalf("Decode: %v", err)
			}
			if got.ShapeType != tc.gi.ShapeType {
				t.Errorf("ShapeType: got %d, want %d", got.ShapeType, tc.gi.ShapeType)
			}
			if math.Abs(got.Latitude-tc.gi.Latitude) > 0.001 {
				t.Errorf("Latitude: got %v, want %v", got.Latitude, tc.gi.Latitude)
			}
			if math.Abs(got.Longitude-tc.gi.Longitude) > 0.001 {
				t.Errorf("Longitude: got %v, want %v", got.Longitude, tc.gi.Longitude)
			}
		})
	}
}

// TS 23.032 V19.0.0 §7.3.
func TestGeoEncode_SevenBitFieldsExactBytes(t *testing.T) {
	gi := &Description{
		ShapeType: ShapeEllipsoidPointUncertainty,
		Latitude:  0, Longitude: 0,
		UncertaintyCode: u8(0x7F),
	}
	data, err := gi.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	want := []byte{0x10, 0, 0, 0, 0, 0, 0, 0x7F}
	if !bytes.Equal(data, want) {
		t.Errorf("Encode bytes: got %x, want %x", data, want)
	}
}

// TS 23.032 V19.0.0 §7.3.
func TestGeoEncode_Rejects7BitFieldsOverflow(t *testing.T) {
	cases := []struct {
		name string
		gi   *Description
	}{
		{
			"UncertaintyCode=200",
			&Description{ShapeType: ShapeEllipsoidPointUncertainty, UncertaintyCode: u8(200)},
		},
		{
			"UncertaintyCode=128",
			&Description{ShapeType: ShapeEllipsoidPointUncertainty, UncertaintyCode: u8(128)},
		},
		{
			"Ellipse: SemiMajor=200",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(200), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(0), Confidence: u8(50),
			},
		},
		{
			"Ellipse: Confidence=255",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(0), Confidence: u8(255),
			},
		},
		{
			"Altitude shape: UncertaintyAltitude=200",
			func() *Description {
				alt := int16(100)
				return &Description{
					ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
					Altitude:  &alt, UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
					AngleMajorAxis: u8(0), UncertaintyAltitude: u8(200), Confidence: u8(50),
				}
			}(),
		},
		{
			"Arc: UncertaintyRadius=200",
			&Description{
				ShapeType:         ShapeEllipsoidArc,
				InnerRadius:       u16(1000),
				UncertaintyRadius: u8(200), OffsetAngle: u8(0),
				IncludedAngle: u8(90), Confidence: u8(50),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.gi.Encode(); err == nil {
				t.Errorf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

// TS 23.032 V19.0.0 §§6.7, 7.3.3, 7.3.6, 7.3.7.
func TestGeoEncode_RejectsAngularFieldsOutOfRange(t *testing.T) {
	alt := int16(100)
	cases := []struct {
		name string
		gi   *Description
	}{
		{
			"Ellipse: AngleMajorAxis=180",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(180), Confidence: u8(50),
			},
		},
		{
			"Ellipse: AngleMajorAxis=255",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(255), Confidence: u8(50),
			},
		},
		{
			"Altitude: AngleMajorAxis=200",
			&Description{
				ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
				Altitude:  &alt, UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(200), UncertaintyAltitude: u8(1), Confidence: u8(50),
			},
		},
		{
			"Arc: OffsetAngle=180",
			&Description{
				ShapeType:   ShapeEllipsoidArc,
				InnerRadius: u16(1000), UncertaintyRadius: u8(1),
				OffsetAngle: u8(180), IncludedAngle: u8(90), Confidence: u8(50),
			},
		},
		{
			"Arc: IncludedAngle=200",
			&Description{
				ShapeType:   ShapeEllipsoidArc,
				InnerRadius: u16(1000), UncertaintyRadius: u8(1),
				OffsetAngle: u8(0), IncludedAngle: u8(200), Confidence: u8(50),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.gi.Encode(); err == nil {
				t.Errorf("%s: expected error, got nil", tc.name)
			}
		})
	}
}

// TS 23.032 V19.0.0 §§6.7, 7.3.3, 7.3.6, 7.3.7.
func TestGeoEncode_AcceptsAngularFieldsAtBoundary(t *testing.T) {
	alt := int16(100)
	cases := []struct {
		name string
		gi   *Description
	}{
		{
			"Ellipse: AngleMajorAxis=0",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(0), Confidence: u8(50),
			},
		},
		{
			"Ellipse: AngleMajorAxis=179",
			&Description{
				ShapeType:            ShapeEllipsoidPointUncertaintyEllipse,
				UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(179), Confidence: u8(50),
			},
		},
		{
			"Altitude: AngleMajorAxis=179",
			&Description{
				ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid,
				Altitude:  &alt, UncertaintySemiMajor: u8(1), UncertaintySemiMinor: u8(1),
				AngleMajorAxis: u8(179), UncertaintyAltitude: u8(1), Confidence: u8(50),
			},
		},
		{
			"Arc: OffsetAngle=179, IncludedAngle=179",
			&Description{
				ShapeType:   ShapeEllipsoidArc,
				InnerRadius: u16(1000), UncertaintyRadius: u8(1),
				OffsetAngle: u8(179), IncludedAngle: u8(179), Confidence: u8(50),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := tc.gi.Encode(); err != nil {
				t.Errorf("%s: unexpected error: %v", tc.name, err)
			}
		})
	}
}

// TS 23.032 V19.0.0 §§6.7, 7.3.3.
func TestGeoDecodeThenEncode_SpecInvalidAngleFailsOnReEncode(t *testing.T) {
	wire := []byte{
		0x30,             // ShapeType=3 (UncertaintyEllipse)<<4
		0x00, 0x00, 0x00, // lat = 0
		0x00, 0x00, 0x00, // lon = 0
		0x01, // SemiMajor = 1
		0x01, // SemiMinor = 1
		200,  // AngleMajorAxis = 200 (SPEC-INVALID)
		0x32, // Confidence = 50
	}
	gi, err := Decode(wire)
	if err != nil {
		t.Fatalf("Decode of spec-invalid AngleMajorAxis=200: unexpected error: %v", err)
	}
	if gi.AngleMajorAxis == nil || *gi.AngleMajorAxis != 200 {
		t.Fatalf("AngleMajorAxis after decode: got %v, want 200", gi.AngleMajorAxis)
	}
	if _, err := gi.Encode(); err == nil {
		t.Error("Encode of spec-invalid AngleMajorAxis=200: expected error, got nil")
	}
}

// TS 23.032 V19.0.0 §6.1.
func u8(v uint8) *uint8 { return &v }

// TS 23.032 V19.0.0 §6.1.
func u16(v uint16) *uint16 { return &v }

// TS 23.032 V19.0.0 §§6.2, 7.3.2.
func TestGeoUncertainty(t *testing.T) {
	for _, tc := range []struct {
		k      byte
		meters float64
	}{
		{0, 0}, {1, 1.0000000000000009}, {2, 2.100000000000002},
	} {
		t.Run(fmt.Sprint(tc.k), func(t *testing.T) {
			got, err := Decode([]byte{0x10, 0, 0, 0, 0, 0, 0, tc.k})
			if err != nil {
				t.Fatal(err)
			}
			if got.UncertaintyMeters == nil || got.UncertaintyCode == nil {
				t.Fatal("missing uncertainty")
			}
			wantEqual(t, "code", tc.k, *got.UncertaintyCode)
			wantEqual(t, "metres", tc.meters, *got.UncertaintyMeters)
		})
	}
}

// TS 23.032 V19.0.0 §7.3.
func TestGeoDecodeLengths(t *testing.T) {
	for _, tc := range []struct {
		name    string
		data    []byte
		want    error
		message string
	}{
		{"empty", nil, ErrInformationTooShort, "geographical information too short"},
		{"point header", []byte{0}, ErrPointInvalidLength, "ellipsoid point requires 7 octets, got 1"},
		{"arc truncated", []byte{0xa0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, ErrArcInvalidLength, "ellipsoid arc requires 13 octets, got 12"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Decode(tc.data)
			if got != nil || !errors.Is(err, tc.want) {
				t.Fatalf("Decode = %v, %v; want nil, %v", got, err, tc.want)
			}
			wantEqual(t, "error", fmt.Sprintf("%s: %s", tc.want, tc.message), err.Error())
		})
	}
	got, err := Decode([]byte{0xa0, 0, 0, 0, 0, 0, 0, 0x12, 0x34, 3, 4, 5, 99})
	if err != nil {
		t.Fatal(err)
	}
	if got.InnerRadius == nil || got.UncertaintyRadius == nil || got.OffsetAngle == nil || got.IncludedAngle == nil || got.Confidence == nil {
		t.Fatal("missing arc fields")
	}
	wantEqual(t, "shape", ShapeEllipsoidArc, got.ShapeType)
	wantEqual(t, "radius", uint16(0x1234), *got.InnerRadius)
	wantEqual(t, "uncertainty", uint8(3), *got.UncertaintyRadius)
	wantEqual(t, "offset", uint8(4), *got.OffsetAngle)
	wantEqual(t, "included", uint8(5), *got.IncludedAngle)
	wantEqual(t, "confidence", uint8(99), *got.Confidence)
}

// TS 23.032 V19.0.0 §6.1.
func TestGeoCoordinates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		lat, lon float64
		want     []byte
		wantErr  error
	}{
		{"south pole", -90, 0, []byte{0, 0xff, 0xff, 0xff, 0, 0, 0}, nil},
		{"north pole", 90, 0, []byte{0, 0x7f, 0xff, 0xff, 0, 0, 0}, nil},
		{"west boundary", 0, -180, []byte{0, 0, 0, 0, 0x80, 0, 0}, nil},
		{"east boundary", 0, 180, nil, ErrLongitudeOutOfRange},
		{"maximum quantum", 8388607.0 * 90 / 8388608, 8388607.0 * 360 / 16777216, []byte{0, 0x7f, 0xff, 0xff, 0x7f, 0xff, 0xff}, nil},
		{"southern hemisphere", -45, -90, []byte{0, 0xc0, 0, 0, 0xc0, 0, 0}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			in := &Description{ShapeType: ShapeEllipsoidPoint, Latitude: tc.lat, Longitude: tc.lon}
			data, err := in.Encode()
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Encode = %x, %v; want %v", data, err, tc.wantErr)
			}
			wantEqual(t, "octets", tc.want, data)
			if err != nil {
				return
			}
			got, err := Decode(tc.want)
			if err != nil {
				t.Fatal(err)
			}
			wantLat := tc.lat
			if tc.lat == 90 || tc.lat == -90 {
				wantLat = math.Copysign(8388607.0*90/8388608, tc.lat)
			}
			wantEqual(t, "latitude", wantLat, got.Latitude)
			wantEqual(t, "longitude", tc.lon, got.Longitude)
		})
	}
}

// TS 23.032 V19.0.0 §§6.3, 7.3.6.
func TestGeoZeroAltitude(t *testing.T) {
	alt := int16(0)
	zero := uint8(0)
	in := &Description{ShapeType: ShapeEllipsoidPointAltitudeUncertaintyEllipsoid, Altitude: &alt,
		UncertaintySemiMajor: &zero, UncertaintySemiMinor: &zero, AngleMajorAxis: &zero,
		UncertaintyAltitude: &zero, Confidence: &zero}
	got, err := in.Encode()
	if err != nil {
		t.Fatal(err)
	}
	wantEqual(t, "octets", []byte{0x90, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}, got)
}

// TS 23.032 V19.0.0 §§6.5, 7.3.7.
func TestGeoArcConfidence(t *testing.T) {
	for _, confidence := range []uint8{99, 128} {
		t.Run(fmt.Sprint(confidence), func(t *testing.T) {
			radius := uint16(0x1234)
			uncertainty, offset, included := uint8(3), uint8(4), uint8(5)
			in := &Description{ShapeType: ShapeEllipsoidArc, InnerRadius: &radius,
				UncertaintyRadius: &uncertainty, OffsetAngle: &offset, IncludedAngle: &included, Confidence: &confidence}
			got, err := in.Encode()
			if confidence == 128 {
				if got != nil || !errors.Is(err, ErrCodeOutOfRange) {
					t.Fatalf("Encode = %x, %v; want confidence range error", got, err)
				}
				wantEqual(t, "error", fmt.Sprintf("%s: Confidence out of range [0, 127]: 128", ErrCodeOutOfRange), err.Error())
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			wantEqual(t, "octets", []byte{0xa0, 0, 0, 0, 0, 0, 0, 0x12, 0x34, 3, 4, 5, 99}, got)
		})
	}
}

// TS 23.032 V19.0.0 §7.2: Decode and Encode support five shape codes.
func TestUnsupportedShapes(t *testing.T) {
	for _, shape := range []Shape{
		ShapePolygon, ShapeEllipsoidPointAltitude,
		ShapeHighAccuracyEllipsoidPointUncertaintyEllipse,
		ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid,
		ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse,
		ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid,
		2, 4, 6, 7, 15,
	} {
		if d, err := Decode([]byte{byte(shape) << 4}); d != nil || !errors.Is(err, ErrShapeTypeInvalid) {
			t.Errorf("Decode shape %d = %v, %v", shape, d, err)
		}
		if b, err := (&Description{ShapeType: shape}).Encode(); b != nil || !errors.Is(err, ErrShapeTypeInvalid) {
			t.Errorf("Encode shape %d = %x, %v", shape, b, err)
		}
	}
}

// TS 23.032 V19.0.0 §7.3: Decode accepts trailing octets and masks seven-bit fields.
func TestDecodePermissiveCoding(t *testing.T) {
	for _, tc := range []struct {
		header      byte
		n           int
		lengthError error
		masked      []int
	}{
		{0, 7, ErrPointInvalidLength, nil},
		{0x10, 8, ErrPointUncertaintyInvalidLength, []int{7}},
		{0x30, 11, ErrPointUncertaintyEllipseInvalidLength, []int{7, 8, 10}},
		{0x90, 14, ErrPointAltitudeInvalidLength, []int{9, 10, 12, 13}},
		{0xa0, 13, ErrArcInvalidLength, []int{9, 12}},
	} {
		t.Run(fmt.Sprintf("shape=%x", tc.header), func(t *testing.T) {
			for n := 1; n < tc.n; n++ {
				b := make([]byte, n)
				b[0] = tc.header
				if d, err := Decode(b); d != nil || !errors.Is(err, tc.lengthError) {
					t.Fatalf("length %d = %v, %v", n, d, err)
				}
			}
			b := make([]byte, tc.n+1)
			b[0] = tc.header | 0xf
			b[tc.n] = 0xff
			for _, i := range tc.masked {
				b[i] = 0xff
			}
			d, err := Decode(b)
			if err != nil {
				t.Fatal(err)
			}
			got, err := d.Encode()
			if err != nil {
				t.Fatal(err)
			}
			want := bytes.Clone(b[:tc.n])
			want[0] = tc.header
			for _, i := range tc.masked {
				want[i] = 0x7f
			}
			if !bytes.Equal(got, want) {
				t.Fatalf("canonical bytes = %x, want %x", got, want)
			}
		})
	}
}
