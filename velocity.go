package gad

// VelocityType identifies velocity coding in TS 23.032 V19.0.0 §8.6 Table 3.
type VelocityType int

const (
	VelocityHorizontal                    VelocityType = 0
	VelocityHorizontalVertical            VelocityType = 1
	VelocityHorizontalUncertainty         VelocityType = 2
	VelocityHorizontalVerticalUncertainty VelocityType = 3
)

// TS 23.032 V19.0.0 §§8.12–8.15, figures 14–17.
var velocityEstimateLength = [4]int{4, 5, 5, 7}

// ValidateVelocity checks type and exact length per TS 23.032 V19.0.0 §§8.5–8.6, 8.12–8.15.
func ValidateVelocity(b []byte) error {
	if len(b) == 0 {
		return ErrVelocityLengthInvalid
	}
	typ := VelocityType(b[0] >> 4)
	if typ > VelocityHorizontalVerticalUncertainty {
		return ErrVelocityTypeInvalid
	}
	if len(b) != velocityEstimateLength[typ] {
		return ErrVelocityLengthInvalid
	}
	return nil
}
