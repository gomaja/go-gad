package gad

import (
	"encoding/binary"
	"fmt"
)

// VelocityType identifies TS 23.032 V19.0.0 §8.6 Table 3 coding.
type VelocityType int

const (
	// VelocityHorizontal codes horizontal velocity, TS 23.032 V19.0.0 §8.6.
	VelocityHorizontal VelocityType = 0
	// VelocityHorizontalVertical codes horizontal with vertical velocity, TS 23.032 V19.0.0 §8.6.
	VelocityHorizontalVertical VelocityType = 1
	// VelocityHorizontalUncertainty codes horizontal velocity with uncertainty, TS 23.032 V19.0.0 §8.6.
	VelocityHorizontalUncertainty VelocityType = 2
	// VelocityHorizontalVerticalUncertainty codes horizontal with vertical velocity and uncertainty, TS 23.032 V19.0.0 §8.6.
	VelocityHorizontalVerticalUncertainty VelocityType = 3
)

// String returns the TS 23.032 V19.0.0 §8.6 Table 3 name.
func (v VelocityType) String() string {
	switch v {
	case VelocityHorizontal:
		return "horizontal velocity"
	case VelocityHorizontalVertical:
		return "horizontal with vertical velocity"
	case VelocityHorizontalUncertainty:
		return "horizontal velocity with uncertainty"
	case VelocityHorizontalVerticalUncertainty:
		return "horizontal with vertical velocity and uncertainty"
	default:
		return fmt.Sprintf("VelocityType(%d)", v)
	}
}

// Velocity is one of the four TS 23.032 V19.0.0 §§8.12–8.15 descriptions.
type Velocity interface {
	Type() VelocityType
	Encode() ([]byte, error)
	isVelocity()
}

// HorizontalVelocity codes TS 23.032 V19.0.0 §8.12.
type HorizontalVelocity struct {
	Bearing         Bearing
	HorizontalSpeed HorizontalSpeed
}

// Type returns TS 23.032 V19.0.0 §8.6 Table 3 coding.
func (HorizontalVelocity) Type() VelocityType { return VelocityHorizontal }

func (HorizontalVelocity) isVelocity() {}

// Encode writes TS 23.032 V19.0.0 §8.12 octets.
func (v HorizontalVelocity) Encode() ([]byte, error) {
	return velocityBytes(VelocityHorizontal, v.Bearing, v.HorizontalSpeed, 4)
}

// HorizontalWithVerticalVelocity codes TS 23.032 V19.0.0 §8.13.
type HorizontalWithVerticalVelocity struct {
	Bearing         Bearing
	HorizontalSpeed HorizontalSpeed
	Downward        bool
	VerticalSpeed   VerticalSpeed
}

// Type returns TS 23.032 V19.0.0 §8.6 Table 3 coding.
func (HorizontalWithVerticalVelocity) Type() VelocityType { return VelocityHorizontalVertical }

func (HorizontalWithVerticalVelocity) isVelocity() {}

// Encode writes TS 23.032 V19.0.0 §8.13 octets.
func (v HorizontalWithVerticalVelocity) Encode() ([]byte, error) {
	b, err := velocityBytes(VelocityHorizontalVertical, v.Bearing, v.HorizontalSpeed, 5)
	if err != nil {
		return nil, err
	}
	if v.Downward {
		b[0] |= 2
	}
	b[4] = byte(v.VerticalSpeed)
	return b, nil
}

// HorizontalVelocityWithUncertainty codes TS 23.032 V19.0.0 §8.14.
type HorizontalVelocityWithUncertainty struct {
	Bearing          Bearing
	HorizontalSpeed  HorizontalSpeed
	UncertaintySpeed UncertaintySpeed
}

// Type returns TS 23.032 V19.0.0 §8.6 Table 3 coding.
func (HorizontalVelocityWithUncertainty) Type() VelocityType { return VelocityHorizontalUncertainty }

func (HorizontalVelocityWithUncertainty) isVelocity() {}

// Encode writes TS 23.032 V19.0.0 §8.14 octets.
func (v HorizontalVelocityWithUncertainty) Encode() ([]byte, error) {
	b, err := velocityBytes(VelocityHorizontalUncertainty, v.Bearing, v.HorizontalSpeed, 5)
	if err != nil {
		return nil, err
	}
	b[4] = byte(v.UncertaintySpeed)
	return b, nil
}

// HorizontalWithVerticalVelocityAndUncertainty codes TS 23.032 V19.0.0 §8.15.
type HorizontalWithVerticalVelocityAndUncertainty struct {
	Bearing                    Bearing
	HorizontalSpeed            HorizontalSpeed
	Downward                   bool
	VerticalSpeed              VerticalSpeed
	HorizontalUncertaintySpeed UncertaintySpeed
	VerticalUncertaintySpeed   UncertaintySpeed
}

// Type returns TS 23.032 V19.0.0 §8.6 Table 3 coding.
func (HorizontalWithVerticalVelocityAndUncertainty) Type() VelocityType {
	return VelocityHorizontalVerticalUncertainty
}

func (HorizontalWithVerticalVelocityAndUncertainty) isVelocity() {}

// Encode writes TS 23.032 V19.0.0 §8.15 octets.
func (v HorizontalWithVerticalVelocityAndUncertainty) Encode() ([]byte, error) {
	b, err := velocityBytes(VelocityHorizontalVerticalUncertainty, v.Bearing, v.HorizontalSpeed, 7)
	if err != nil {
		return nil, err
	}
	if v.Downward {
		b[0] |= 2
	}
	b[4] = byte(v.VerticalSpeed)
	b[5] = byte(v.HorizontalUncertaintySpeed)
	b[6] = byte(v.VerticalUncertaintySpeed)
	return b, nil
}

func velocityBytes(typ VelocityType, bearing Bearing, speed HorizontalSpeed, n int) ([]byte, error) {
	if bearing > 359 {
		return nil, rangeError("Bearing", 1, int64(bearing), 0, 359)
	}
	b := make([]byte, n)
	b[0] = byte(typ)<<4 | byte(bearing>>8)
	b[1] = byte(bearing)
	binary.BigEndian.PutUint16(b[2:4], uint16(speed))
	return b, nil
}

// DecodeVelocity reads strict TS 23.032 V19.0.0 §§8.12–8.15 octets.
func DecodeVelocity(b []byte) (Velocity, error) {
	if len(b) == 0 {
		return nil, lengthError("Velocity", 0, 1)
	}
	typ := VelocityType(b[0] >> 4)
	if typ > 3 {
		return nil, fieldError("Velocity type", 1, ErrVelocityTypeInvalid, typ.String())
	}
	n := [4]int{4, 5, 5, 7}[typ]
	if len(b) != n {
		return nil, lengthError(typ.String(), len(b), n)
	}
	mask := byte(14)
	if typ == 1 || typ == 3 {
		mask = 12
	}
	if b[0]&mask != 0 {
		return nil, fieldError("Spare", 1, ErrSpareBit, "")
	}
	bearing := Bearing(b[0]&1)<<8 | Bearing(b[1])
	// TS 23.032 V19.0.0 §8.8: 511 is unusable too, despite the printed <511.
	if bearing > 359 {
		return nil, rangeError("Bearing", 1, int64(bearing), 0, 359)
	}
	speed := HorizontalSpeed(binary.BigEndian.Uint16(b[2:4]))
	switch typ {
	case VelocityHorizontal:
		return HorizontalVelocity{Bearing: bearing, HorizontalSpeed: speed}, nil
	case VelocityHorizontalVertical:
		return HorizontalWithVerticalVelocity{Bearing: bearing, HorizontalSpeed: speed, Downward: b[0]&2 != 0, VerticalSpeed: VerticalSpeed(b[4])}, nil
	case VelocityHorizontalUncertainty:
		return HorizontalVelocityWithUncertainty{Bearing: bearing, HorizontalSpeed: speed, UncertaintySpeed: UncertaintySpeed(b[4])}, nil
	default:
		// TS 23.032 V19.0.0 §8.6 Table 3 and Annex A give type 0011; figure 17 prints 0010.
		return HorizontalWithVerticalVelocityAndUncertainty{Bearing: bearing, HorizontalSpeed: speed, Downward: b[0]&2 != 0, VerticalSpeed: VerticalSpeed(b[4]), HorizontalUncertaintySpeed: UncertaintySpeed(b[5]), VerticalUncertaintySpeed: UncertaintySpeed(b[6])}, nil
	}
}

// ValidateVelocity returns exactly DecodeVelocity's error, TS 23.032 V19.0.0 §8.
func ValidateVelocity(b []byte) error {
	_, err := DecodeVelocity(b)
	return err
}
