# go-gad

The octet coding of 3GPP TS 23.032, Universal Geographical Area Description
(GAD), for Go: the geographical shapes and the velocity descriptions that MAP,
CAP, Diameter location services and other 3GPP protocols carry.

## Status

The package holds the TS 23.032 code that `github.com/gomaja/go-asn1-gsmmap`
used internally, moved here. It decodes and encodes five shapes (ellipsoid
point, point with uncertainty circle, point with uncertainty ellipse, point
with altitude and uncertainty ellipsoid, ellipsoid arc) and validates every
shape the specification codes, eleven in all, and the four velocity types.
[Issue #1](https://github.com/gomaja/go-gad/issues/1) describes the complete
scope this module is to implement.

## Install

The module is used from its main branch; there are no tagged releases.

```
go get github.com/gomaja/go-gad@main
```

## Usage

```go
d, err := gad.Decode(b) // point with uncertainty circle, for example
if err != nil {
	return err
}
fmt.Println(d.ShapeType, d.Latitude, d.Longitude, *d.UncertaintyMeters)

err = gad.Validate(b)         // any of the eleven coded shapes
err = gad.ValidateVelocity(v) // velocity types 0 to 3
```

## Specifications

[STANDARDS.md](STANDARDS.md) lists the documents and versions the code follows.

## License

MIT, see [LICENSE](LICENSE).
