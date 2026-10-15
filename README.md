# go-gad

The octet coding of 3GPP TS 23.032, Universal Geographical Area Description
(GAD), for Go: the geographical shapes and the velocity descriptions that MAP,
CAP, Diameter location services and other 3GPP protocols carry.

## Status

The package decodes, encodes and validates every shape TS 23.032 V19.0.0 codes
in octets, eleven in all, and the four velocity types, as
[issue #1](https://github.com/gomaja/go-gad/issues/1) specifies. Decoding is
strict and lossless: every value it accepts encodes back to the same octets.
The decoded values are cross-checked against Wireshark and libosmocore; see
[testdata/crosscheck](testdata/crosscheck/README.md).

## Install

The module is used from its main branch; there are no tagged releases.

```
go get github.com/gomaja/go-gad@main
```

## Usage

```go
d, err := gad.Decode(b) // strict decoding of any of the eleven shapes
if err != nil {
    return err
}
if circle, ok := d.(gad.EllipsoidPointUncertaintyCircle); ok {
    fmt.Println(circle.Latitude.Degrees(), circle.Longitude.Degrees(),
        circle.Uncertainty.Metres())
}

point, err := gad.PointFromDegrees(48.8566, 2.3522)
if err != nil {
    return err
}
radius, err := gad.UncertaintyFromMetres(20)
if err != nil {
    return err
}
encoded, err := (gad.EllipsoidPointUncertaintyCircle{
    Point: point, Uncertainty: radius,
}).Encode()
if err != nil {
    return err
}
fmt.Printf("%x\n", encoded)

velocity, err := gad.DecodeVelocity(v) // all four velocity types
if err != nil {
    return err
}
fmt.Println(velocity.Type())

err = gad.Validate(b)         // exactly Decode's validation
err = gad.ValidateVelocity(v) // exactly DecodeVelocity's validation
```

Coded fields preserve accepted octets, including signed zero, confidence codes
without information, and unspecified speed uncertainty. Constructors convert
physical units to the specification's bins; uncertainty constructors round up.
Every error is a `*gad.FieldError`, with a field and octet, wrapping a sentinel
such as `gad.ErrLength` or `gad.ErrOutOfRange` for use with `errors.Is`.
Polygon geometry constraints in TS 23.032 §5.4 are not checked.

## Specifications

[STANDARDS.md](STANDARDS.md) lists the documents and versions the code follows.

## License

MIT, see [LICENSE](LICENSE).
