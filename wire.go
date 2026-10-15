package gad

import "encoding/binary"

func checkShape(b []byte) error {
	if len(b) == 0 {
		return lengthError("Geographical information", 0, 1)
	}
	shape := Shape(b[0] >> 4)
	length := 0
	switch shape {
	case ShapeEllipsoidPoint:
		length = 7
	case ShapeEllipsoidPointUncertaintyCircle:
		length = 8
	case ShapeEllipsoidPointUncertaintyEllipse:
		length = 11
	case ShapePolygon:
		n := int(b[0] & 15)
		if n < 3 {
			return rangeError("Number of points", 1, int64(n), 3, 15)
		}
		length = 1 + 6*n
	case ShapeEllipsoidPointAltitude:
		length = 9
	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid:
		length = 14
	case ShapeEllipsoidArc:
		length = 13
	case ShapeHighAccuracyEllipsoidPointUncertaintyEllipse:
		length = 13
	case ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid:
		length = 18
	case ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse:
		length = 13
	case ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid:
		length = 18
	default:
		return fieldError("Type of shape", 1, ErrShapeTypeInvalid, shape.String())
	}
	if len(b) != length {
		return lengthError(shape.String(), len(b), length)
	}
	if shape != ShapePolygon && b[0]&15 != 0 {
		return fieldError("Spare", 1, ErrSpareBit, "")
	}
	switch shape {
	case ShapeEllipsoidPoint: // TS 23.032 V19.0.0 §7.3.1.
	case ShapeEllipsoidPointUncertaintyCircle: // TS 23.032 V19.0.0 §7.3.2.
		if b[7]&128 != 0 {
			return fieldError("Uncertainty", 8, ErrSpareBit, "")
		}
	case ShapeEllipsoidPointUncertaintyEllipse: // TS 23.032 V19.0.0 §7.3.3.
		if b[7]&128 != 0 {
			return fieldError("Uncertainty semi-major", 8, ErrSpareBit, "")
		}
		if b[8]&128 != 0 {
			return fieldError("Uncertainty semi-minor", 9, ErrSpareBit, "")
		}
		if b[9] > 179 {
			return rangeError("Orientation of major axis", 10, int64(b[9]), 0, 179)
		}
		if b[10]&128 != 0 {
			return fieldError("Confidence", 11, ErrSpareBit, "")
		}
	case ShapePolygon: // TS 23.032 V19.0.0 §7.3.4.
	case ShapeEllipsoidPointAltitude: // TS 23.032 V19.0.0 §7.3.5.
	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid: // TS 23.032 V19.0.0 §7.3.6.
		if b[9]&128 != 0 {
			return fieldError("Uncertainty semi-major", 10, ErrSpareBit, "")
		}
		if b[10]&128 != 0 {
			return fieldError("Uncertainty semi-minor", 11, ErrSpareBit, "")
		}
		if b[11] > 179 {
			return rangeError("Orientation of major axis", 12, int64(b[11]), 0, 179)
		}
		if b[12]&128 != 0 {
			return fieldError("Uncertainty altitude", 13, ErrSpareBit, "")
		}
		if b[13]&128 != 0 {
			return fieldError("Confidence", 14, ErrSpareBit, "")
		}
	case ShapeEllipsoidArc: // TS 23.032 V19.0.0 §7.3.7.
		if b[9]&128 != 0 {
			return fieldError("Uncertainty radius", 10, ErrSpareBit, "")
		}
		if b[10] > 179 {
			return rangeError("Offset angle", 11, int64(b[10]), 0, 179)
		}
		if b[11] > 179 {
			return rangeError("Included angle", 12, int64(b[11]), 0, 179)
		}
		if b[12]&128 != 0 {
			return fieldError("Confidence", 13, ErrSpareBit, "")
		}
	case ShapeHighAccuracyEllipsoidPointUncertaintyEllipse: // TS 23.032 V19.0.0 §7.3.3a.
		if b[11] > 179 {
			return rangeError("Orientation of major axis", 12, int64(b[11]), 0, 179)
		}
		if b[12]&128 != 0 {
			return fieldError("Confidence", 13, ErrSpareBit, "")
		}
	case ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid: // TS 23.032 V19.0.0 §7.3.6a.
		if b[9]&0xc0 != 0 {
			return fieldError("Spare", 10, ErrSpareBit, "")
		}
		a := readHighAltitude(b)
		if a < -64000 || a > 1280000 {
			return rangeError("Altitude", 10, int64(a), -64000, 1280000)
		}
		if b[14] > 179 {
			return rangeError("Orientation of major axis", 15, int64(b[14]), 0, 179)
		}
		if b[15]&128 != 0 {
			return fieldError("Horizontal confidence", 16, ErrSpareBit, "")
		}
		if b[17]&128 != 0 {
			return fieldError("Vertical confidence", 18, ErrSpareBit, "")
		}
	case ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse: // TS 23.032 V19.0.0 §7.3.3b.
		if b[11] > 179 {
			return rangeError("Orientation of major axis", 12, int64(b[11]), 0, 179)
		}
	case ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid: // TS 23.032 V19.0.0 §7.3.6b.
		if b[9]&0xc0 != 0 {
			return fieldError("Spare", 10, ErrSpareBit, "")
		}
		a := readHighAltitude(b)
		if a < -64000 || a > 1280000 {
			return rangeError("Altitude", 10, int64(a), -64000, 1280000)
		}
		if b[14] > 179 {
			return rangeError("Orientation of major axis", 15, int64(b[14]), 0, 179)
		}
	}
	return nil
}

// TS 23.032 V19.0.0 §7.1 uses big-endian octets.
func read16(b []byte, o int) uint16 { return binary.BigEndian.Uint16(b[o-1 : o+1]) }

// TS 23.032 V19.0.0 §6.1: latitude sign-magnitude, longitude signed 24-bit.
func readPoint(b []byte, o int) Point {
	i := o - 1
	n := int32(b[i+3])<<16 | int32(b[i+4])<<8 | int32(b[i+5])
	if n&0x800000 != 0 {
		n -= 1 << 24
	}
	return Point{Latitude: Latitude{South: b[i]&128 != 0, N: uint32(b[i]&127)<<16 | uint32(b[i+1])<<8 | uint32(b[i+2])}, Longitude: Longitude(n)}
}

// TS 23.032 V19.0.0 §6.1a: signed 32-bit coordinates.
func readHighPoint(b []byte) HighAccuracyPoint {
	return HighAccuracyPoint{Latitude: HighAccuracyLatitude(int32(binary.BigEndian.Uint32(b[1:5]))), Longitude: HighAccuracyLongitude(int32(binary.BigEndian.Uint32(b[5:9])))}
}

// TS 23.032 V19.0.0 §6.3a: sign-extend the 22-bit altitude code.
func readHighAltitude(b []byte) int32 {
	a := int32(b[9]&63)<<16 | int32(b[10])<<8 | int32(b[11])
	if a&0x200000 != 0 {
		a -= 1 << 22
	}
	return a
}

type shapeWriter struct {
	b   []byte
	err error
}

func newShapeWriter(s Shape, n int) *shapeWriter {
	b := make([]byte, n)
	b[0] = byte(s) << 4
	return &shapeWriter{b: b}
}

func (w *shapeWriter) fail(err error) {
	if w.err == nil {
		w.err = err
	}
}

func (w *shapeWriter) result() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}
	return w.b, nil
}

func (w *shapeWriter) code(name string, o int, n, maxCode int64) {
	if n < 0 || n > maxCode {
		w.fail(rangeError(name, o, n, 0, maxCode))
	}
	w.b[o-1] = byte(n)
}

func (w *shapeWriter) uint16(n uint16, o int) { binary.BigEndian.PutUint16(w.b[o-1:o+1], n) }

func (w *shapeWriter) point(p Point, o int) {
	n := p.Latitude.N
	if n > 0x7fffff {
		w.fail(rangeError("Latitude", o, int64(n), 0, 0x7fffff))
	}
	w.b[o-1] = byte(n >> 16)
	w.b[o] = byte(n >> 8)
	w.b[o+1] = byte(n)
	if p.Latitude.South {
		w.b[o-1] |= 128
	}
	l := int32(p.Longitude)
	if l < -0x800000 || l > 0x7fffff {
		w.fail(rangeError("Longitude", o+3, int64(l), -0x800000, 0x7fffff))
	}
	w.b[o+2] = byte(l >> 16)
	w.b[o+3] = byte(l >> 8)
	w.b[o+4] = byte(l)
}

func (w *shapeWriter) highPoint(p HighAccuracyPoint) {
	binary.BigEndian.PutUint32(w.b[1:5], uint32(p.Latitude))
	binary.BigEndian.PutUint32(w.b[5:9], uint32(p.Longitude))
}

func (w *shapeWriter) altitude(a Altitude, o int) {
	if a.N > 32767 {
		w.fail(rangeError("Altitude", o, int64(a.N), 0, 32767))
	}
	w.uint16(a.N, o)
	if a.Depth {
		w.b[o-1] |= 128
	}
}

func (w *shapeWriter) highAltitude(a HighAccuracyAltitude, o int) {
	if a < -64000 || a > 1280000 {
		w.fail(rangeError("Altitude", o, int64(a), -64000, 1280000))
	}
	w.b[o-1] = byte(uint32(a)>>16) & 63
	w.b[o] = byte(uint32(a) >> 8)
	w.b[o+1] = byte(a)
}
