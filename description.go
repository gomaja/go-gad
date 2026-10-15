package gad

// Description is one of the eleven TS 23.032 V19.0.0 §7.3 shapes.
type Description interface {
	Shape() Shape
	Encode() ([]byte, error)
	isDescription()
}

// EllipsoidPoint codes TS 23.032 V19.0.0 §7.3.1.
type EllipsoidPoint struct {
	Point
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidPoint) Shape() Shape { return ShapeEllipsoidPoint }

func (EllipsoidPoint) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.1 octets.
func (v EllipsoidPoint) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidPoint, 7)
	w.point(v.Point, 2)
	return w.result()
}

// EllipsoidPointUncertaintyCircle codes TS 23.032 V19.0.0 §7.3.2.
type EllipsoidPointUncertaintyCircle struct {
	Point
	Uncertainty Uncertainty
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidPointUncertaintyCircle) Shape() Shape { return ShapeEllipsoidPointUncertaintyCircle }

func (EllipsoidPointUncertaintyCircle) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.2 octets.
func (v EllipsoidPointUncertaintyCircle) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidPointUncertaintyCircle, 8)
	w.point(v.Point, 2)
	w.code("Uncertainty", 8, int64(v.Uncertainty), 127)
	return w.result()
}

// EllipsoidPointUncertaintyEllipse codes TS 23.032 V19.0.0 §7.3.3.
type EllipsoidPointUncertaintyEllipse struct {
	Point
	SemiMajor   Uncertainty
	SemiMinor   Uncertainty
	Orientation Orientation
	Confidence  Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidPointUncertaintyEllipse) Shape() Shape { return ShapeEllipsoidPointUncertaintyEllipse }

func (EllipsoidPointUncertaintyEllipse) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.3 octets.
func (v EllipsoidPointUncertaintyEllipse) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidPointUncertaintyEllipse, 11)
	w.point(v.Point, 2)
	w.code("Uncertainty semi-major", 8, int64(v.SemiMajor), 127)
	w.code("Uncertainty semi-minor", 9, int64(v.SemiMinor), 127)
	w.code("Orientation of major axis", 10, int64(v.Orientation), 179)
	w.code("Confidence", 11, int64(v.Confidence), 127)
	return w.result()
}

// Polygon codes TS 23.032 V19.0.0 §7.3.4.
// Polygon geometry constraints in §5.4 are not checked.
type Polygon struct {
	Points []Point
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (Polygon) Shape() Shape { return ShapePolygon }

func (Polygon) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.4 octets.
func (v Polygon) Encode() ([]byte, error) {
	if len(v.Points) < 3 || len(v.Points) > 15 {
		return nil, rangeError("Number of points", 1, int64(len(v.Points)), 3, 15)
	}
	w := newShapeWriter(ShapePolygon, 1+6*len(v.Points))
	w.b[0] |= byte(len(v.Points))
	for i, p := range v.Points {
		w.point(p, 2+6*i)
	}
	return w.result()
}

// EllipsoidPointAltitude codes TS 23.032 V19.0.0 §7.3.5.
type EllipsoidPointAltitude struct {
	Point
	Altitude Altitude
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidPointAltitude) Shape() Shape { return ShapeEllipsoidPointAltitude }

func (EllipsoidPointAltitude) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.5 octets.
func (v EllipsoidPointAltitude) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidPointAltitude, 9)
	w.point(v.Point, 2)
	w.altitude(v.Altitude, 8)
	return w.result()
}

// EllipsoidPointAltitudeUncertaintyEllipsoid codes TS 23.032 V19.0.0 §7.3.6.
type EllipsoidPointAltitudeUncertaintyEllipsoid struct {
	Point
	Altitude            Altitude
	SemiMajor           Uncertainty
	SemiMinor           Uncertainty
	Orientation         Orientation
	UncertaintyAltitude UncertaintyAltitude
	Confidence          Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidPointAltitudeUncertaintyEllipsoid) Shape() Shape {
	return ShapeEllipsoidPointAltitudeUncertaintyEllipsoid
}

func (EllipsoidPointAltitudeUncertaintyEllipsoid) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.6 octets.
func (v EllipsoidPointAltitudeUncertaintyEllipsoid) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidPointAltitudeUncertaintyEllipsoid, 14)
	w.point(v.Point, 2)
	w.altitude(v.Altitude, 8)
	w.code("Uncertainty semi-major", 10, int64(v.SemiMajor), 127)
	w.code("Uncertainty semi-minor", 11, int64(v.SemiMinor), 127)
	w.code("Orientation of major axis", 12, int64(v.Orientation), 179)
	w.code("Uncertainty altitude", 13, int64(v.UncertaintyAltitude), 127)
	w.code("Confidence", 14, int64(v.Confidence), 127)
	return w.result()
}

// EllipsoidArc codes TS 23.032 V19.0.0 §7.3.7.
type EllipsoidArc struct {
	Point
	InnerRadius       InnerRadius
	UncertaintyRadius Uncertainty
	OffsetAngle       OffsetAngle
	IncludedAngle     IncludedAngle
	Confidence        Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (EllipsoidArc) Shape() Shape { return ShapeEllipsoidArc }

func (EllipsoidArc) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.7 octets.
func (v EllipsoidArc) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeEllipsoidArc, 13)
	w.point(v.Point, 2)
	w.uint16(uint16(v.InnerRadius), 8)
	w.code("Uncertainty radius", 10, int64(v.UncertaintyRadius), 127)
	w.code("Offset angle", 11, int64(v.OffsetAngle), 179)
	w.code("Included angle", 12, int64(v.IncludedAngle), 179)
	w.code("Confidence", 13, int64(v.Confidence), 127)
	return w.result()
}

// HighAccuracyEllipsoidPointUncertaintyEllipse codes TS 23.032 V19.0.0 §7.3.3a.
type HighAccuracyEllipsoidPointUncertaintyEllipse struct {
	HighAccuracyPoint
	SemiMajor   HighAccuracyUncertainty
	SemiMinor   HighAccuracyUncertainty
	Orientation Orientation
	Confidence  Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (HighAccuracyEllipsoidPointUncertaintyEllipse) Shape() Shape {
	return ShapeHighAccuracyEllipsoidPointUncertaintyEllipse
}

func (HighAccuracyEllipsoidPointUncertaintyEllipse) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.3a octets.
func (v HighAccuracyEllipsoidPointUncertaintyEllipse) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeHighAccuracyEllipsoidPointUncertaintyEllipse, 13)
	w.highPoint(v.HighAccuracyPoint)
	w.code("Uncertainty semi-major", 10, int64(v.SemiMajor), 255)
	w.code("Uncertainty semi-minor", 11, int64(v.SemiMinor), 255)
	w.code("Orientation of major axis", 12, int64(v.Orientation), 179)
	w.code("Confidence", 13, int64(v.Confidence), 127)
	return w.result()
}

// HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid codes TS 23.032 V19.0.0 §7.3.6a.
type HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid struct {
	HighAccuracyPoint
	Altitude             HighAccuracyAltitude
	SemiMajor            HighAccuracyUncertainty
	SemiMinor            HighAccuracyUncertainty
	Orientation          Orientation
	HorizontalConfidence Confidence
	UncertaintyAltitude  HighAccuracyUncertainty
	VerticalConfidence   Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid) Shape() Shape {
	return ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid
}

func (HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.6a octets.
func (v HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid, 18)
	w.highPoint(v.HighAccuracyPoint)
	w.highAltitude(v.Altitude, 10)
	w.code("Uncertainty semi-major", 13, int64(v.SemiMajor), 255)
	w.code("Uncertainty semi-minor", 14, int64(v.SemiMinor), 255)
	w.code("Orientation of major axis", 15, int64(v.Orientation), 179)
	w.code("Horizontal confidence", 16, int64(v.HorizontalConfidence), 127)
	w.code("Uncertainty altitude", 17, int64(v.UncertaintyAltitude), 255)
	w.code("Vertical confidence", 18, int64(v.VerticalConfidence), 127)
	return w.result()
}

// HighAccuracyEllipsoidPointScalableUncertaintyEllipse codes TS 23.032 V19.0.0 §7.3.3b.
type HighAccuracyEllipsoidPointScalableUncertaintyEllipse struct {
	HighAccuracyPoint
	SemiMajor   ScalableUncertainty
	SemiMinor   ScalableUncertainty
	Orientation Orientation
	Confidence  Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (HighAccuracyEllipsoidPointScalableUncertaintyEllipse) Shape() Shape {
	return ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse
}

func (HighAccuracyEllipsoidPointScalableUncertaintyEllipse) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.3b octets.
func (v HighAccuracyEllipsoidPointScalableUncertaintyEllipse) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse, 13)
	w.highPoint(v.HighAccuracyPoint)
	w.b[9] = v.SemiMajor.K
	if v.SemiMinor.Extended != v.SemiMajor.Extended {
		w.fail(fieldError("Uncertainty semi-minor", 11, ErrUncertaintyRange, ""))
	}
	w.b[10] = v.SemiMinor.K
	w.code("Orientation of major axis", 12, int64(v.Orientation), 179)
	w.code("Confidence", 13, int64(v.Confidence), 127)
	if v.SemiMajor.Extended {
		w.b[12] |= 0x80
	}
	return w.result()
}

// HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid codes TS 23.032 V19.0.0 §7.3.6b.
type HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid struct {
	HighAccuracyPoint
	Altitude             HighAccuracyAltitude
	SemiMajor            ScalableUncertainty
	SemiMinor            ScalableUncertainty
	Orientation          Orientation
	HorizontalConfidence Confidence
	UncertaintyAltitude  ScalableUncertainty
	VerticalConfidence   Confidence
}

// Shape returns the type in TS 23.032 V19.0.0 §7.2.
func (HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid) Shape() Shape {
	return ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid
}

func (HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid) isDescription() {}

// Encode writes TS 23.032 V19.0.0 §7.3.6b octets.
func (v HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid) Encode() ([]byte, error) {
	w := newShapeWriter(ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid, 18)
	w.highPoint(v.HighAccuracyPoint)
	w.highAltitude(v.Altitude, 10)
	w.b[12] = v.SemiMajor.K
	if v.SemiMinor.Extended != v.SemiMajor.Extended {
		w.fail(fieldError("Uncertainty semi-minor", 14, ErrUncertaintyRange, ""))
	}
	w.b[13] = v.SemiMinor.K
	w.code("Orientation of major axis", 15, int64(v.Orientation), 179)
	w.code("Horizontal confidence", 16, int64(v.HorizontalConfidence), 127)
	w.b[16] = v.UncertaintyAltitude.K
	w.code("Vertical confidence", 18, int64(v.VerticalConfidence), 127)
	if v.SemiMajor.Extended {
		w.b[15] |= 0x80
	}
	if v.UncertaintyAltitude.Extended {
		w.b[17] |= 0x80
	}
	return w.result()
}

// Decode reads exact, strict TS 23.032 V19.0.0 §7.3 octet coding.
// It copies values and retains no input storage.
func Decode(b []byte) (Description, error) {
	if err := checkShape(b); err != nil {
		return nil, err
	}
	switch Shape(b[0] >> 4) {
	case ShapeEllipsoidPoint:
		return EllipsoidPoint{Point: readPoint(b, 2)}, nil
	case ShapeEllipsoidPointUncertaintyCircle:
		return EllipsoidPointUncertaintyCircle{Point: readPoint(b, 2), Uncertainty: Uncertainty(b[7])}, nil
	case ShapeEllipsoidPointUncertaintyEllipse:
		return EllipsoidPointUncertaintyEllipse{Point: readPoint(b, 2), SemiMajor: Uncertainty(b[7]), SemiMinor: Uncertainty(b[8]), Orientation: Orientation(b[9]), Confidence: Confidence(b[10] & 127)}, nil
	case ShapePolygon:
		v := Polygon{Points: make([]Point, int(b[0]&15))}
		for i := range v.Points {
			v.Points[i] = readPoint(b, 2+6*i)
		}
		return v, nil
	case ShapeEllipsoidPointAltitude:
		return EllipsoidPointAltitude{Point: readPoint(b, 2), Altitude: Altitude{Depth: b[7]&128 != 0, N: read16(b, 8) & 32767}}, nil
	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid:
		return EllipsoidPointAltitudeUncertaintyEllipsoid{Point: readPoint(b, 2), Altitude: Altitude{Depth: b[7]&128 != 0, N: read16(b, 8) & 32767}, SemiMajor: Uncertainty(b[9]), SemiMinor: Uncertainty(b[10]), Orientation: Orientation(b[11]), UncertaintyAltitude: UncertaintyAltitude(b[12]), Confidence: Confidence(b[13] & 127)}, nil
	case ShapeEllipsoidArc:
		return EllipsoidArc{Point: readPoint(b, 2), InnerRadius: InnerRadius(read16(b, 8)), UncertaintyRadius: Uncertainty(b[9]), OffsetAngle: OffsetAngle(b[10]), IncludedAngle: IncludedAngle(b[11]), Confidence: Confidence(b[12] & 127)}, nil
	case ShapeHighAccuracyEllipsoidPointUncertaintyEllipse:
		return HighAccuracyEllipsoidPointUncertaintyEllipse{HighAccuracyPoint: readHighPoint(b), SemiMajor: HighAccuracyUncertainty(b[9]), SemiMinor: HighAccuracyUncertainty(b[10]), Orientation: Orientation(b[11]), Confidence: Confidence(b[12] & 127)}, nil
	case ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid:
		return HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{HighAccuracyPoint: readHighPoint(b), Altitude: HighAccuracyAltitude(readHighAltitude(b)), SemiMajor: HighAccuracyUncertainty(b[12]), SemiMinor: HighAccuracyUncertainty(b[13]), Orientation: Orientation(b[14]), HorizontalConfidence: Confidence(b[15] & 127), UncertaintyAltitude: HighAccuracyUncertainty(b[16]), VerticalConfidence: Confidence(b[17] & 127)}, nil
	case ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse:
		return HighAccuracyEllipsoidPointScalableUncertaintyEllipse{HighAccuracyPoint: readHighPoint(b), SemiMajor: ScalableUncertainty{Extended: b[12]&128 != 0, K: b[9]}, SemiMinor: ScalableUncertainty{Extended: b[12]&128 != 0, K: b[10]}, Orientation: Orientation(b[11]), Confidence: Confidence(b[12] & 127)}, nil
	case ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid:
		return HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid{HighAccuracyPoint: readHighPoint(b), Altitude: HighAccuracyAltitude(readHighAltitude(b)), SemiMajor: ScalableUncertainty{Extended: b[15]&128 != 0, K: b[12]}, SemiMinor: ScalableUncertainty{Extended: b[15]&128 != 0, K: b[13]}, Orientation: Orientation(b[14]), HorizontalConfidence: Confidence(b[15] & 127), UncertaintyAltitude: ScalableUncertainty{Extended: b[17]&128 != 0, K: b[16]}, VerticalConfidence: Confidence(b[17] & 127)}, nil
	}
	panic("unreachable shape")
}

// Validate returns exactly Decode's error, TS 23.032 V19.0.0 §7.3.
func Validate(b []byte) error {
	_, err := Decode(b)
	return err
}
