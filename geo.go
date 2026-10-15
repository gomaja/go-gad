package gad

import (
	"fmt"
	"math"
)

// Shape identifies a geographical shape in TS 23.032 V19.0.0 §7.2.
type Shape int

// Shape codes are defined by TS 23.032 V19.0.0 §7.2 Table 2a.
const (
	ShapeEllipsoidPoint                                                 Shape = 0
	ShapeEllipsoidPointUncertainty                                      Shape = 1
	ShapeEllipsoidPointUncertaintyEllipse                               Shape = 3
	ShapePolygon                                                        Shape = 5
	ShapeEllipsoidPointAltitude                                         Shape = 8
	ShapeEllipsoidPointAltitudeUncertaintyEllipsoid                     Shape = 9
	ShapeEllipsoidArc                                                   Shape = 10
	ShapeHighAccuracyEllipsoidPointUncertaintyEllipse                   Shape = 11
	ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid         Shape = 12
	ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse           Shape = 13
	ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid Shape = 14
)

// Description holds the five shapes decoded by Decode (TS 23.032 V19.0.0 §7.3).
// Optional fields hold coded octet values, except UncertaintyMeters.
// AngleMajorAxis uses degrees; arc angles use two-degree bins (§6.7).
type Description struct {
	ShapeType            Shape    `json:"ShapeType"`
	Latitude             float64  `json:"Latitude"`
	Longitude            float64  `json:"Longitude"`
	UncertaintyCode      *uint8   `json:"UncertaintyCode,omitempty"`
	UncertaintyMeters    *float64 `json:"UncertaintyMeters,omitempty"`
	UncertaintySemiMajor *uint8   `json:"UncertaintySemiMajor,omitempty"`
	UncertaintySemiMinor *uint8   `json:"UncertaintySemiMinor,omitempty"`
	AngleMajorAxis       *uint8   `json:"AngleMajorAxis,omitempty"`
	Confidence           *uint8   `json:"Confidence,omitempty"`
	Altitude             *int16   `json:"Altitude,omitempty"`
	UncertaintyAltitude  *uint8   `json:"UncertaintyAltitude,omitempty"`
	InnerRadius          *uint16  `json:"InnerRadius,omitempty"`
	UncertaintyRadius    *uint8   `json:"UncertaintyRadius,omitempty"`
	OffsetAngle          *uint8   `json:"OffsetAngle,omitempty"`
	IncludedAngle        *uint8   `json:"IncludedAngle,omitempty"`
}

// uncertaintyToMeters applies C(K) = 10 * (1.1^K - 1), TS 23.032 V19.0.0 §6.2.
func uncertaintyToMeters(k uint8) float64 {
	return 10 * (math.Pow(1.1, float64(k)) - 1)
}

// check7Bit checks seven-bit fields in TS 23.032 V19.0.0 §7.3.
func check7Bit(name string, v uint8) error {
	if v > 127 {
		return fmt.Errorf("%w: %s out of range [0, 127]: %d", ErrCodeOutOfRange, name, v)
	}
	return nil
}

// checkAngle checks angular codes 0..179, TS 23.032 V19.0.0 §§6.7, 7.3.3.
func checkAngle(name string, v uint8) error {
	if v > 179 {
		return fmt.Errorf("%w: %s out of range [0, 179]: %d", ErrAngleOutOfRange, name, v)
	}
	return nil
}

// Decode reads five shapes from TS 23.032 V19.0.0 §§7.3.1–7.3.3, 7.3.6, 7.3.7.
// It accepts trailing octets and does not validate spare bits or angular values.
// Use Validate to check the complete octet coding.
func Decode(data []byte) (*Description, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("%w: geographical information too short", ErrInformationTooShort)
	}

	shapeType := Shape(data[0] >> 4)
	gi := &Description{ShapeType: shapeType}

	switch shapeType {
	case ShapeEllipsoidPoint:
		if len(data) < 7 {
			return nil, fmt.Errorf("%w: ellipsoid point requires 7 octets, got %d", ErrPointInvalidLength, len(data))
		}
		gi.Latitude, gi.Longitude = decodeLatLon(data[1:7])

	case ShapeEllipsoidPointUncertainty:
		if len(data) < 8 {
			return nil, fmt.Errorf("%w: ellipsoid point with uncertainty requires 8 octets, got %d", ErrPointUncertaintyInvalidLength, len(data))
		}
		gi.Latitude, gi.Longitude = decodeLatLon(data[1:7])
		uc := data[7] & 0x7F
		gi.UncertaintyCode = &uc
		meters := uncertaintyToMeters(uc)
		gi.UncertaintyMeters = &meters

	case ShapeEllipsoidPointUncertaintyEllipse:
		if len(data) < 11 {
			return nil, fmt.Errorf("%w: ellipsoid point with uncertainty ellipse requires 11 octets, got %d", ErrPointUncertaintyEllipseInvalidLength, len(data))
		}
		gi.Latitude, gi.Longitude = decodeLatLon(data[1:7])
		semiMajor := data[7] & 0x7F
		semiMinor := data[8] & 0x7F
		angle := data[9]
		confidence := data[10] & 0x7F
		gi.UncertaintySemiMajor = &semiMajor
		gi.UncertaintySemiMinor = &semiMinor
		gi.AngleMajorAxis = &angle
		gi.Confidence = &confidence

	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid:
		if len(data) < 14 {
			return nil, fmt.Errorf("%w: ellipsoid point with altitude requires 14 octets, got %d", ErrPointAltitudeInvalidLength, len(data))
		}
		gi.Latitude, gi.Longitude = decodeLatLon(data[1:7])
		altSign := data[7] >> 7
		altVal := int16(data[7]&0x7F)<<8 | int16(data[8])
		if altSign == 1 {
			altVal = -altVal
		}
		gi.Altitude = &altVal
		semiMajor := data[9] & 0x7F
		semiMinor := data[10] & 0x7F
		angle := data[11]
		uncAlt := data[12] & 0x7F
		confidence := data[13] & 0x7F
		gi.UncertaintySemiMajor = &semiMajor
		gi.UncertaintySemiMinor = &semiMinor
		gi.AngleMajorAxis = &angle
		gi.UncertaintyAltitude = &uncAlt
		gi.Confidence = &confidence

	case ShapeEllipsoidArc:
		if len(data) < 13 {
			return nil, fmt.Errorf("%w: ellipsoid arc requires 13 octets, got %d", ErrArcInvalidLength, len(data))
		}
		gi.Latitude, gi.Longitude = decodeLatLon(data[1:7])
		innerRadius := uint16(data[7])<<8 | uint16(data[8])
		uncRadius := data[9] & 0x7F
		offsetAngle := data[10]
		includedAngle := data[11]
		confidence := data[12] & 0x7F
		gi.InnerRadius = &innerRadius
		gi.UncertaintyRadius = &uncRadius
		gi.OffsetAngle = &offsetAngle
		gi.IncludedAngle = &includedAngle
		gi.Confidence = &confidence

	default:
		return nil, fmt.Errorf("%w: unsupported shape type: %d", ErrShapeTypeInvalid, shapeType)
	}

	return gi, nil
}

// Encode codes a description per TS 23.032 V19.0.0 §§6.1–6.7, 7.3.
// Coordinates use floor bins; the highest latitude bin includes the poles (§6.1).
// Arc code N represents 2N <= offset < 2(N+1) and 2N < included <= 2(N+1) (§6.7).
func (gi *Description) Encode() ([]byte, error) {
	if math.IsNaN(gi.Latitude) || math.IsInf(gi.Latitude, 0) {
		return nil, fmt.Errorf("%w: latitude is not a finite number: %v", ErrLatitudeNotFinite, gi.Latitude)
	}
	if math.IsNaN(gi.Longitude) || math.IsInf(gi.Longitude, 0) {
		return nil, fmt.Errorf("%w: longitude is not a finite number: %v", ErrLongitudeNotFinite, gi.Longitude)
	}
	if gi.Latitude < -90 || gi.Latitude > 90 {
		return nil, fmt.Errorf("%w: latitude out of range [-90, 90]: %v", ErrLatitudeOutOfRange, gi.Latitude)
	}
	if gi.Longitude < -180 || gi.Longitude >= 180 {
		return nil, fmt.Errorf("%w: longitude out of range [-180, 180): %v", ErrLongitudeOutOfRange, gi.Longitude)
	}
	latBytes, lonBytes := encodeLatLon(gi.Latitude, gi.Longitude)

	switch gi.ShapeType {
	case ShapeEllipsoidPoint:
		data := make([]byte, 7)
		data[0] = byte(gi.ShapeType) << 4
		copy(data[1:4], latBytes[:])
		copy(data[4:7], lonBytes[:])
		return data, nil

	case ShapeEllipsoidPointUncertainty:
		if gi.UncertaintyCode == nil {
			return nil, fmt.Errorf("%w: EllipsoidPointUncertainty requires UncertaintyCode", ErrPointUncertaintyFieldsMissing)
		}
		if err := check7Bit("UncertaintyCode", *gi.UncertaintyCode); err != nil {
			return nil, err
		}
		data := make([]byte, 8)
		data[0] = byte(gi.ShapeType) << 4
		copy(data[1:4], latBytes[:])
		copy(data[4:7], lonBytes[:])
		data[7] = *gi.UncertaintyCode
		return data, nil

	case ShapeEllipsoidPointUncertaintyEllipse:
		if gi.UncertaintySemiMajor == nil || gi.UncertaintySemiMinor == nil ||
			gi.AngleMajorAxis == nil || gi.Confidence == nil {
			return nil, fmt.Errorf("%w: EllipsoidPointUncertaintyEllipse requires UncertaintySemiMajor, UncertaintySemiMinor, AngleMajorAxis, Confidence", ErrPointUncertaintyEllipseFieldsMissing)
		}
		if err := check7Bit("UncertaintySemiMajor", *gi.UncertaintySemiMajor); err != nil {
			return nil, err
		}
		if err := check7Bit("UncertaintySemiMinor", *gi.UncertaintySemiMinor); err != nil {
			return nil, err
		}
		if err := checkAngle("AngleMajorAxis", *gi.AngleMajorAxis); err != nil {
			return nil, err
		}
		if err := check7Bit("Confidence", *gi.Confidence); err != nil {
			return nil, err
		}
		data := make([]byte, 11)
		data[0] = byte(gi.ShapeType) << 4
		copy(data[1:4], latBytes[:])
		copy(data[4:7], lonBytes[:])
		data[7] = *gi.UncertaintySemiMajor
		data[8] = *gi.UncertaintySemiMinor
		data[9] = *gi.AngleMajorAxis
		data[10] = *gi.Confidence
		return data, nil

	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid:
		if gi.Altitude == nil || gi.UncertaintySemiMajor == nil ||
			gi.UncertaintySemiMinor == nil || gi.AngleMajorAxis == nil ||
			gi.UncertaintyAltitude == nil || gi.Confidence == nil {
			return nil, fmt.Errorf("%w: EllipsoidPointAltitude requires Altitude, UncertaintySemiMajor, UncertaintySemiMinor, AngleMajorAxis, UncertaintyAltitude, Confidence", ErrPointAltitudeFieldsMissing)
		}
		alt := *gi.Altitude
		if alt < -32767 {
			return nil, fmt.Errorf("%w: altitude out of range [-32767, 32767]: %d", ErrAltitudeOutOfRange, alt)
		}
		if err := check7Bit("UncertaintySemiMajor", *gi.UncertaintySemiMajor); err != nil {
			return nil, err
		}
		if err := check7Bit("UncertaintySemiMinor", *gi.UncertaintySemiMinor); err != nil {
			return nil, err
		}
		if err := checkAngle("AngleMajorAxis", *gi.AngleMajorAxis); err != nil {
			return nil, err
		}
		if err := check7Bit("UncertaintyAltitude", *gi.UncertaintyAltitude); err != nil {
			return nil, err
		}
		if err := check7Bit("Confidence", *gi.Confidence); err != nil {
			return nil, err
		}
		data := make([]byte, 14)
		data[0] = byte(gi.ShapeType) << 4
		copy(data[1:4], latBytes[:])
		copy(data[4:7], lonBytes[:])
		if alt < 0 {
			data[7] = 0x80 | byte((-alt)>>8)
			data[8] = byte(-alt)
		} else {
			data[7] = byte(alt >> 8)
			data[8] = byte(alt)
		}
		data[9] = *gi.UncertaintySemiMajor
		data[10] = *gi.UncertaintySemiMinor
		data[11] = *gi.AngleMajorAxis
		data[12] = *gi.UncertaintyAltitude
		data[13] = *gi.Confidence
		return data, nil

	case ShapeEllipsoidArc:
		if gi.InnerRadius == nil || gi.UncertaintyRadius == nil ||
			gi.OffsetAngle == nil || gi.IncludedAngle == nil || gi.Confidence == nil {
			return nil, fmt.Errorf("%w: EllipsoidArc requires InnerRadius, UncertaintyRadius, OffsetAngle, IncludedAngle, Confidence", ErrArcFieldsMissing)
		}
		if err := check7Bit("UncertaintyRadius", *gi.UncertaintyRadius); err != nil {
			return nil, err
		}
		if err := checkAngle("OffsetAngle", *gi.OffsetAngle); err != nil {
			return nil, err
		}
		if err := checkAngle("IncludedAngle", *gi.IncludedAngle); err != nil {
			return nil, err
		}
		if err := check7Bit("Confidence", *gi.Confidence); err != nil {
			return nil, err
		}
		data := make([]byte, 13)
		data[0] = byte(gi.ShapeType) << 4
		copy(data[1:4], latBytes[:])
		copy(data[4:7], lonBytes[:])
		data[7] = byte(*gi.InnerRadius >> 8)
		data[8] = byte(*gi.InnerRadius)
		data[9] = *gi.UncertaintyRadius
		data[10] = *gi.OffsetAngle
		data[11] = *gi.IncludedAngle
		data[12] = *gi.Confidence
		return data, nil

	default:
		return nil, fmt.Errorf("%w: unsupported shape type: %d", ErrShapeTypeInvalid, gi.ShapeType)
	}
}

// decodeLatLon returns bin edges per TS 23.032 V19.0.0 §6.1.
func decodeLatLon(data []byte) (lat, lon float64) {
	sign := (data[0] >> 7) & 0x01
	latN := uint32(data[0]&0x7F)<<16 | uint32(data[1])<<8 | uint32(data[2])
	lat = float64(latN) * 90.0 / float64(1<<23)
	if sign == 1 {
		lat = -lat
	}

	lonN := int32(data[3])<<16 | int32(data[4])<<8 | int32(data[5])
	if lonN&0x800000 != 0 {
		lonN |= -1 << 24
	}
	lon = float64(lonN) * 360.0 / float64(1<<24)

	return lat, lon
}

// encodeLatLon applies floor bins and the extended polar bin, TS 23.032 V19.0.0 §6.1.
// Scaling before division preserves decoded edges and subnormal longitude inputs.
func encodeLatLon(lat, lon float64) (latBytes [3]byte, lonBytes [3]byte) {
	var sign byte
	// Preserve the southern bin zero sign (TS 23.032 V19.0.0 §6.1).
	if math.Signbit(lat) {
		sign = 1
		lat = -lat
	}
	latN := uint32(math.Floor(lat * float64(1<<23) / 90.0))
	if latN > 0x7FFFFF {
		latN = 0x7FFFFF
	}
	latBytes[0] = sign<<7 | byte((latN>>16)&0x7F)
	latBytes[1] = byte(latN >> 8)
	latBytes[2] = byte(latN)

	lonN := int32(math.Floor(lon * float64(1<<24) / 360.0))
	if lonN > 0x7FFFFF {
		lonN = 0x7FFFFF
	}
	lonBytes[0] = byte(lonN >> 16)
	lonBytes[1] = byte(lonN >> 8)
	lonBytes[2] = byte(lonN)

	return latBytes, lonBytes
}
