package gad

import "fmt"

// Shape identifies TS 23.032 V19.0.0 §7.2 Table 2a coding.
type Shape int

const (
	// ShapeEllipsoidPoint identifies ellipsoid point, TS 23.032 V19.0.0 §7.3.1.
	ShapeEllipsoidPoint Shape = 0
	// ShapeEllipsoidPointUncertaintyCircle identifies ellipsoid point with uncertainty circle, TS 23.032 V19.0.0 §7.3.2.
	ShapeEllipsoidPointUncertaintyCircle Shape = 1
	// ShapeEllipsoidPointUncertaintyEllipse identifies ellipsoid point with uncertainty ellipse, TS 23.032 V19.0.0 §7.3.3.
	ShapeEllipsoidPointUncertaintyEllipse Shape = 3
	// ShapePolygon identifies polygon, TS 23.032 V19.0.0 §7.3.4.
	ShapePolygon Shape = 5
	// ShapeEllipsoidPointAltitude identifies ellipsoid point with altitude, TS 23.032 V19.0.0 §7.3.5.
	ShapeEllipsoidPointAltitude Shape = 8
	// ShapeEllipsoidPointAltitudeUncertaintyEllipsoid identifies ellipsoid point with altitude and uncertainty ellipsoid, TS 23.032 V19.0.0 §7.3.6.
	ShapeEllipsoidPointAltitudeUncertaintyEllipsoid Shape = 9
	// ShapeEllipsoidArc identifies ellipsoid arc, TS 23.032 V19.0.0 §7.3.7.
	ShapeEllipsoidArc Shape = 10
	// ShapeHighAccuracyEllipsoidPointUncertaintyEllipse identifies high accuracy ellipsoid point with uncertainty ellipse, TS 23.032 V19.0.0 §7.3.3a.
	ShapeHighAccuracyEllipsoidPointUncertaintyEllipse Shape = 11
	// ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid identifies high accuracy ellipsoid point with altitude and uncertainty ellipsoid, TS 23.032 V19.0.0 §7.3.6a.
	ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid Shape = 12
	// ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse identifies high accuracy ellipsoid point with scalable uncertainty ellipse, TS 23.032 V19.0.0 §7.3.3b.
	ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse Shape = 13
	// ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid identifies high accuracy ellipsoid point with altitude and scalable uncertainty ellipsoid, TS 23.032 V19.0.0 §7.3.6b.
	ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid Shape = 14
)

// String returns the TS 23.032 V19.0.0 §7.2 Table 2a name.
func (s Shape) String() string {
	switch s {
	case ShapeEllipsoidPoint:
		return "ellipsoid point"
	case ShapeEllipsoidPointUncertaintyCircle:
		return "ellipsoid point with uncertainty circle"
	case ShapeEllipsoidPointUncertaintyEllipse:
		return "ellipsoid point with uncertainty ellipse"
	case ShapePolygon:
		return "polygon"
	case ShapeEllipsoidPointAltitude:
		return "ellipsoid point with altitude"
	case ShapeEllipsoidPointAltitudeUncertaintyEllipsoid:
		return "ellipsoid point with altitude and uncertainty ellipsoid"
	case ShapeEllipsoidArc:
		return "ellipsoid arc"
	case ShapeHighAccuracyEllipsoidPointUncertaintyEllipse:
		return "high accuracy ellipsoid point with uncertainty ellipse"
	case ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid:
		return "high accuracy ellipsoid point with altitude and uncertainty ellipsoid"
	case ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse:
		return "high accuracy ellipsoid point with scalable uncertainty ellipse"
	case ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid:
		return "high accuracy ellipsoid point with altitude and scalable uncertainty ellipsoid"
	default:
		return fmt.Sprintf("Shape(%d)", s)
	}
}
