package gad

import "errors"

// Coding errors for TS 23.032 V19.0.0 §§6–8.
var (
	ErrAltitudeOutOfRange                   = errors.New("gad: geographical altitude is out of range")
	ErrAngleOutOfRange                      = errors.New("gad: geographical angle is out of range")
	ErrArcFieldsMissing                     = errors.New("gad: ellipsoid arc has missing fields")
	ErrArcInvalidLength                     = errors.New("gad: ellipsoid arc has invalid length")
	ErrCodeOutOfRange                       = errors.New("gad: geographical code is out of range")
	ErrInformationTooShort                  = errors.New("gad: geographical information is too short")
	ErrLatitudeNotFinite                    = errors.New("gad: latitude is not finite")
	ErrLatitudeOutOfRange                   = errors.New("gad: latitude is out of range")
	ErrLongitudeNotFinite                   = errors.New("gad: longitude is not finite")
	ErrLongitudeOutOfRange                  = errors.New("gad: longitude is out of range")
	ErrPointAltitudeFieldsMissing           = errors.New("gad: ellipsoid point with altitude has missing fields")
	ErrPointAltitudeInvalidLength           = errors.New("gad: ellipsoid point with altitude has invalid length")
	ErrPointInvalidLength                   = errors.New("gad: ellipsoid point has invalid length")
	ErrPointUncertaintyEllipseFieldsMissing = errors.New("gad: ellipsoid point with uncertainty ellipse has missing fields")
	ErrPointUncertaintyEllipseInvalidLength = errors.New("gad: ellipsoid point with uncertainty ellipse has invalid length")
	ErrPointUncertaintyFieldsMissing        = errors.New("gad: ellipsoid point with uncertainty has missing fields")
	ErrPointUncertaintyInvalidLength        = errors.New("gad: ellipsoid point with uncertainty has invalid length")
	ErrShapeTypeInvalid                     = errors.New("gad: geographical shape type is unsupported")
	ErrVelocityTypeInvalid                  = errors.New("gad: velocity type must be 0..3 (TS 23.032 V19.0.0 §8.6)")
	ErrVelocityLengthInvalid                = errors.New("gad: velocity must be 4, 5, 5 or 7 octets for types 0..3 (TS 23.032 V19.0.0 §§8.12–8.15)")
)
