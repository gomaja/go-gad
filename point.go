package gad

import (
	"fmt"
	"math"
)

// Latitude codes signed latitude per TS 23.032 V19.0.0 §6.1.
// South preserves the sign of the southern zero bin.
type Latitude struct {
	South bool
	N     uint32
}

// Degrees returns the lower magnitude edge of the latitude bin per TS 23.032 V19.0.0 §6.1.
func (l Latitude) Degrees() float64 {
	x := float64(l.N) * 90 / (1 << 23)
	if l.South {
		return -x
	}
	return x
}

// LatitudeFromDegrees uses floor magnitude bins per TS 23.032 V19.0.0 §6.1.
// The highest bin includes 90 degrees; southern zero retains its sign.
func LatitudeFromDegrees(x float64) (Latitude, error) {
	if err := coordinateDomain("Latitude", x, 90, true); err != nil {
		return Latitude{}, err
	}
	n := math.Floor(math.Abs(x) * (1 << 23) / 90)
	if n > (1<<23)-1 {
		n = (1 << 23) - 1
	}
	return Latitude{South: math.Signbit(x), N: uint32(n)}, nil
}

// Longitude codes 24-bit two's complement longitude per TS 23.032 V19.0.0 §6.1.
type Longitude int32

// Degrees returns the lower edge of the longitude bin per TS 23.032 V19.0.0 §6.1.
func (l Longitude) Degrees() float64 { return float64(l) * 360 / (1 << 24) }

// LongitudeFromDegrees uses floor bins on [-180,180) per TS 23.032 V19.0.0 §6.1.
func LongitudeFromDegrees(x float64) (Longitude, error) {
	if err := coordinateDomain("Longitude", x, 180, false); err != nil {
		return 0, err
	}
	// For x < 180 the scaled quotient stays below 2^23, so no clamp is needed.
	return Longitude(math.Floor(x * (1 << 24) / 360)), nil
}

// Point contains the ellipsoid coordinates of TS 23.032 V19.0.0 §6.1.
type Point struct {
	Latitude  Latitude
	Longitude Longitude
}

// PointFromDegrees codes ellipsoid coordinates per TS 23.032 V19.0.0 §6.1.
func PointFromDegrees(lat, lon float64) (Point, error) {
	latitude, err := LatitudeFromDegrees(lat)
	if err != nil {
		return Point{}, err
	}
	longitude, err := LongitudeFromDegrees(lon)
	if err != nil {
		return Point{}, err
	}
	return Point{Latitude: latitude, Longitude: longitude}, nil
}

// HighAccuracyLatitude codes 32-bit latitude per TS 23.032 V19.0.0 §6.1a.
type HighAccuracyLatitude int32

// Degrees returns the lower edge of the latitude bin per TS 23.032 V19.0.0 §6.1a.
func (l HighAccuracyLatitude) Degrees() float64 { return float64(l) * 90 / (1 << 31) }

// HighAccuracyLatitudeFromDegrees uses floor bins per TS 23.032 V19.0.0 §6.1a.
// That clause maps +90 degrees outside int32; the highest bin is extended to include it.
func HighAccuracyLatitudeFromDegrees(x float64) (HighAccuracyLatitude, error) {
	if err := coordinateDomain("Latitude", x, 90, true); err != nil {
		return 0, err
	}
	n := math.Floor(x * (1 << 31) / 90)
	if n > math.MaxInt32 {
		n = math.MaxInt32
	}
	return HighAccuracyLatitude(n), nil
}

// HighAccuracyLongitude codes 32-bit longitude per TS 23.032 V19.0.0 §6.1a.
type HighAccuracyLongitude int32

// Degrees returns the lower edge of the longitude bin per TS 23.032 V19.0.0 §6.1a.
func (l HighAccuracyLongitude) Degrees() float64 { return float64(l) * 180 / (1 << 31) }

// HighAccuracyLongitudeFromDegrees uses floor bins on [-180,180) per TS 23.032 V19.0.0 §6.1a.
func HighAccuracyLongitudeFromDegrees(x float64) (HighAccuracyLongitude, error) {
	if err := coordinateDomain("Longitude", x, 180, false); err != nil {
		return 0, err
	}
	// For x < 180 the scaled quotient stays below 2^31, so no clamp is needed.
	return HighAccuracyLongitude(math.Floor(x * (1 << 31) / 180)), nil
}

// HighAccuracyPoint contains ellipsoid coordinates per TS 23.032 V19.0.0 §6.1a.
type HighAccuracyPoint struct {
	Latitude  HighAccuracyLatitude
	Longitude HighAccuracyLongitude
}

// HighAccuracyPointFromDegrees codes ellipsoid coordinates per TS 23.032 V19.0.0 §6.1a.
func HighAccuracyPointFromDegrees(lat, lon float64) (HighAccuracyPoint, error) {
	latitude, err := HighAccuracyLatitudeFromDegrees(lat)
	if err != nil {
		return HighAccuracyPoint{}, err
	}
	longitude, err := HighAccuracyLongitudeFromDegrees(lon)
	if err != nil {
		return HighAccuracyPoint{}, err
	}
	return HighAccuracyPoint{Latitude: latitude, Longitude: longitude}, nil
}

func coordinateDomain(field string, x, limit float64, inclusive bool) error {
	if err := finiteField(field, x); err != nil {
		return err
	}
	if x < -limit || x > limit || (!inclusive && x == limit) {
		upper := ")"
		if inclusive {
			upper = "]"
		}
		return fieldError(field, 0, ErrOutOfRange, fmt.Sprintf("%g, want [-%g,%g%s", x, limit, limit, upper))
	}
	return nil
}
