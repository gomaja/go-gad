// Package gad codes the geographical area descriptions and the velocities of
// 3GPP TS 23.032 V19.0.0 (Universal Geographical Area Description) in octets.
//
// Decode returns one of the eleven shape types, from EllipsoidPoint to
// HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid; a type
// switch selects the shape. DecodeVelocity returns one of the four velocity
// types. Decoding is strict: the length must be exact, every spare bit zero
// and every field within its coded range. Validate and ValidateVelocity
// return the same errors without a value. Decoding is also lossless: Encode
// of a decoded value reproduces the input octets.
//
// A value holds the coded fields, the N and K codes and flags of §6 and §8;
// methods convert them to degrees, metres, km/h and percent. Constructors
// such as PointFromDegrees and UncertaintyFromMetres choose the codes for
// physical values by the bin rules of §6 and §8. Uncertainty constructors
// round up, so a coded uncertainty is never smaller than the one given.
//
// Every error is a *FieldError naming the field and octet; it wraps one of
// the sentinel errors, for use with errors.Is.
//
// The polygon geometry rules of §5.4 are not checked.
package gad
