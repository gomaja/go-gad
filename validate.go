package gad

import "fmt"

// Validate checks every shape in TS 23.032 V19.0.0 §7.2 Table 2a against §7.3.
// Header spare bits are unconstrained; confidence codes 101..127 are accepted (§6.5).
func Validate(b []byte) error {
	if len(b) == 0 {
		return ErrInformationTooShort
	}
	shape := Shape(b[0] >> 4)
	var length int
	var zeroBits, angles []int
	switch shape {
	case ShapeEllipsoidPoint: // §7.3.1, figure 5.
		length = 7
	case ShapeEllipsoidPointUncertainty: // §7.3.2, figure 6.
		length, zeroBits = 8, []int{7}
	case ShapeEllipsoidPointUncertaintyEllipse: // §7.3.3, figure 6a.
		length, zeroBits, angles = 11, []int{7, 8, 10}, []int{9}
	case ShapePolygon: // §7.3.4, figure 7.
		points := int(b[0] & 0xf)
		if points < 3 || points > 15 {
			return fmt.Errorf("gad: polygon requires 3..15 points, got %d (TS 23.032 V19.0.0 §7.3.4)", points)
		}
		length = 1 + 6*points
	case ShapeEllipsoidPointAltitude: // §7.3.5, figure 8.
		length = 9
	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid: // §7.3.6, figure 9.
		length, zeroBits, angles = 14, []int{9, 10, 12, 13}, []int{11}
	case ShapeEllipsoidArc: // §7.3.7, figure 10; angles follow §6.7.
		length, zeroBits, angles = 13, []int{9, 12}, []int{10, 11}
	case ShapeHighAccuracyEllipsoidPointUncertaintyEllipse: // §7.3.3a, figure 7.3.3a-1.
		length, zeroBits, angles = 13, []int{12}, []int{11}
	case ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse: // §7.3.3b, figure 7.3.3b-1.
		length, angles = 13, []int{11}
	case ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid: // §7.3.6a, figure 7.3.6a-1.
		length, zeroBits, angles = 18, []int{15, 17}, []int{14}
	case ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid: // §7.3.6b, figure 7.3.6b-1.
		length, angles = 18, []int{14}
	default:
		return fmt.Errorf("%w: unsupported shape type: %d", ErrShapeTypeInvalid, shape)
	}
	if len(b) != length {
		return fmt.Errorf("gad: shape %d requires exactly %d octets, got %d (TS 23.032 V19.0.0 §7.3)", shape, length, len(b))
	}
	for _, i := range zeroBits {
		if b[i]&0x80 != 0 {
			return fmt.Errorf("%w: shape %d octet %d requires bit 8 zero (TS 23.032 V19.0.0 §7.3)", ErrCodeOutOfRange, shape, i+1)
		}
	}
	for _, i := range angles {
		if b[i] > 179 {
			return fmt.Errorf("%w: shape %d octet %d requires 0..179, got %d (TS 23.032 V19.0.0 §§6.7, 7.3)", ErrAngleOutOfRange, shape, i+1, b[i])
		}
	}
	if shape == ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid || shape == ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid {
		// TS 23.032 V19.0.0 §§7.3.6a, 7.3.6b: two explicit zero bits.
		if b[9]&0xc0 != 0 {
			return fmt.Errorf("%w: shape %d octet 10 requires bits 8..7 zero (TS 23.032 V19.0.0 §§7.3.6a, 7.3.6b)", ErrCodeOutOfRange, shape)
		}
		// TS 23.032 V19.0.0 §6.3a: signed 22-bit altitude code limits.
		altitude := int32(b[9]&0x3f)<<16 | int32(b[10])<<8 | int32(b[11])
		if altitude&0x200000 != 0 {
			altitude -= 1 << 22
		}
		if altitude < -64000 || altitude > 1280000 {
			return fmt.Errorf("%w: high accuracy altitude code requires [-64000, 1280000], got %d (TS 23.032 V19.0.0 §6.3a)", ErrAltitudeOutOfRange, altitude)
		}
	}
	return nil
}
