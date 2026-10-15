package gad

import (
	"bytes"
	"math/rand"
	"reflect"
	"testing"
)

func TestDescriptionLossless(t *testing.T) {
	r := rand.New(rand.NewSource(230032))
	for iteration := 0; iteration < 256; iteration++ {
		for _, original := range randomDescriptions(r, iteration) {
			encoded, err := original.Encode()
			if err != nil {
				t.Fatalf("iteration %d, %T: Encode(%#v): %v", iteration, original, original, err)
			}
			if !referenceShapeAccepts(encoded) {
				t.Fatalf("iteration %d, %T: encoding %x violates reference", iteration, original, encoded)
			}
			decoded, err := Decode(encoded)
			if err != nil {
				t.Fatalf("iteration %d, %T: Decode(%x): %v", iteration, original, encoded, err)
			}
			if !reflect.DeepEqual(decoded, original) {
				t.Fatalf("iteration %d: decoded %#v, want %#v", iteration, decoded, original)
			}
			again, err := decoded.Encode()
			if err != nil || !bytes.Equal(again, encoded) {
				t.Fatalf("iteration %d, %T: second encoding %x, %v; want %x", iteration, original, again, err, encoded)
			}
			// §7.3 descriptions own their values after Decode returns.
			for i := range encoded {
				encoded[i] ^= 0xff
			}
			if !reflect.DeepEqual(decoded, original) {
				t.Fatalf("iteration %d, %T: Decode retained input storage", iteration, original)
			}
		}
	}
}

func TestVelocityLossless(t *testing.T) {
	r := rand.New(rand.NewSource(230033))
	for iteration := 0; iteration < 256; iteration++ {
		for _, original := range randomVelocities(r, iteration) {
			encoded, err := original.Encode()
			if err != nil {
				t.Fatalf("iteration %d, %T: Encode(%#v): %v", iteration, original, original, err)
			}
			if !referenceVelocityAccepts(encoded) {
				t.Fatalf("iteration %d, %T: encoding %x violates reference", iteration, original, encoded)
			}
			decoded, err := DecodeVelocity(encoded)
			if err != nil {
				t.Fatalf("iteration %d, %T: DecodeVelocity(%x): %v", iteration, original, encoded, err)
			}
			if !reflect.DeepEqual(decoded, original) {
				t.Fatalf("iteration %d: decoded %#v, want %#v", iteration, decoded, original)
			}
			again, err := decoded.Encode()
			if err != nil || !bytes.Equal(again, encoded) {
				t.Fatalf("iteration %d, %T: second encoding %x, %v; want %x", iteration, original, again, err, encoded)
			}
			for i := range encoded {
				encoded[i] ^= 0xff
			}
			if !reflect.DeepEqual(decoded, original) {
				t.Fatalf("iteration %d, %T: DecodeVelocity retained input storage", iteration, original)
			}
		}
	}
}

// The first three iterations force zero, maximum, and signed-zero codes;
// subsequent iterations sample independently selected valid coded fields.
func propertyCode(r *rand.Rand, iteration, maximum int) int {
	switch iteration {
	case 0, 2:
		return 0
	case 1:
		return maximum
	default:
		return r.Intn(maximum + 1)
	}
}

func randomDescriptions(r *rand.Rand, iteration int) []Description {
	code := func(maximum int) int { return propertyCode(r, iteration, maximum) }
	point := func() Point {
		return Point{
			Latitude:  Latitude{South: iteration&1 != 0 || iteration == 2, N: uint32(code(0x7fffff))},
			Longitude: Longitude(code(0xffffff) - 0x800000),
		}
	}
	var high HighAccuracyPoint
	switch iteration {
	case 0:
		high = HighAccuracyPoint{Latitude: -1 << 31, Longitude: -1 << 31}
	case 1:
		high = HighAccuracyPoint{Latitude: 1<<31 - 1, Longitude: 1<<31 - 1}
	case 2:
	default:
		high = HighAccuracyPoint{Latitude: HighAccuracyLatitude(r.Uint32()), Longitude: HighAccuracyLongitude(r.Uint32())}
	}
	altitude := Altitude{Depth: iteration&2 != 0 || iteration == 1, N: uint16(code(32767))}
	highAltitude := HighAccuracyAltitude(code(1344000) - 64000)
	major, minor := Uncertainty(code(127)), Uncertainty(code(127))
	highMajor, highMinor := HighAccuracyUncertainty(code(255)), HighAccuracyUncertainty(code(255))
	orientation, confidence := Orientation(code(179)), Confidence(code(127))
	horizontalExtended, verticalExtended := iteration&1 != 0, iteration&2 != 0
	scalableMajor := ScalableUncertainty{Extended: horizontalExtended, K: uint8(highMajor)}
	scalableMinor := ScalableUncertainty{Extended: horizontalExtended, K: uint8(highMinor)}
	points := make([]Point, 3+code(12))
	for i := range points {
		points[i] = point()
		// A polygon must also preserve independently signed vertices.
		points[i].Latitude.South = (iteration+i)&1 != 0
	}
	return []Description{
		EllipsoidPoint{Point: point()},
		EllipsoidPointUncertaintyCircle{Point: point(), Uncertainty: major},
		EllipsoidPointUncertaintyEllipse{Point: point(), SemiMajor: major, SemiMinor: minor, Orientation: orientation, Confidence: confidence},
		Polygon{Points: points},
		EllipsoidPointAltitude{Point: point(), Altitude: altitude},
		EllipsoidPointAltitudeUncertaintyEllipsoid{
			Point: point(), Altitude: altitude, SemiMajor: major, SemiMinor: minor,
			Orientation: orientation, UncertaintyAltitude: UncertaintyAltitude(code(127)), Confidence: confidence,
		},
		EllipsoidArc{
			Point: point(), InnerRadius: InnerRadius(code(65535)), UncertaintyRadius: major,
			OffsetAngle: OffsetAngle(code(179)), IncludedAngle: IncludedAngle(code(179)), Confidence: confidence,
		},
		HighAccuracyEllipsoidPointUncertaintyEllipse{
			HighAccuracyPoint: high, SemiMajor: highMajor, SemiMinor: highMinor,
			Orientation: orientation, Confidence: confidence,
		},
		HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{
			HighAccuracyPoint: high, Altitude: highAltitude, SemiMajor: highMajor, SemiMinor: highMinor,
			Orientation: orientation, HorizontalConfidence: confidence,
			UncertaintyAltitude: HighAccuracyUncertainty(code(255)), VerticalConfidence: Confidence(code(127)),
		},
		HighAccuracyEllipsoidPointScalableUncertaintyEllipse{
			HighAccuracyPoint: high, SemiMajor: scalableMajor, SemiMinor: scalableMinor,
			Orientation: orientation, Confidence: confidence,
		},
		HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{
			HighAccuracyPoint: high, Altitude: highAltitude, SemiMajor: scalableMajor, SemiMinor: scalableMinor,
			Orientation: orientation, HorizontalConfidence: confidence,
			UncertaintyAltitude: ScalableUncertainty{Extended: verticalExtended, K: uint8(code(255))},
			VerticalConfidence:  Confidence(code(127)),
		},
	}
}

func randomVelocities(r *rand.Rand, iteration int) []Velocity {
	code := func(maximum int) int { return propertyCode(r, iteration, maximum) }
	bearing, horizontal := Bearing(code(359)), HorizontalSpeed(code(65535))
	vertical, downward := VerticalSpeed(code(255)), iteration&1 != 0
	return []Velocity{
		HorizontalVelocity{Bearing: bearing, HorizontalSpeed: horizontal},
		HorizontalWithVerticalVelocity{Bearing: bearing, HorizontalSpeed: horizontal, VerticalSpeed: vertical, Downward: downward},
		HorizontalVelocityWithUncertainty{Bearing: bearing, HorizontalSpeed: horizontal, UncertaintySpeed: UncertaintySpeed(code(255))},
		HorizontalWithVerticalVelocityAndUncertainty{
			Bearing: bearing, HorizontalSpeed: horizontal, VerticalSpeed: vertical, Downward: downward,
			HorizontalUncertaintySpeed: UncertaintySpeed(code(255)), VerticalUncertaintySpeed: UncertaintySpeed(code(255)),
		},
	}
}
