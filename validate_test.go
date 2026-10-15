package gad

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

// TS 23.032 V19.0.0 §7.2 Table 2a and §7.3.
func TestTable2aShapes(t *testing.T) {
	for _, tc := range []struct {
		shape  Shape
		header byte
		length int
	}{
		{ShapeEllipsoidPoint, 0x00, 7},
		{ShapeEllipsoidPointUncertainty, 0x10, 8},
		{ShapeEllipsoidPointUncertaintyEllipse, 0x30, 11},
		{ShapePolygon, 0x53, 19},
		{ShapeEllipsoidPointAltitude, 0x80, 9},
		{ShapeEllipsoidPointAltitudeUncertaintyEllipsoid, 0x90, 14},
		{ShapeEllipsoidArc, 0xa0, 13},
		{ShapeHighAccuracyEllipsoidPointUncertaintyEllipse, 0xb0, 13},
		{ShapeHighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid, 0xc0, 18},
		{ShapeHighAccuracyEllipsoidPointScalableUncertaintyEllipse, 0xd0, 13},
		{ShapeHighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid, 0xe0, 18},
	} {
		t.Run(fmt.Sprint(tc.shape), func(t *testing.T) {
			for _, delta := range []int{-1, 0, 1} {
				b := make([]byte, tc.length+delta)
				b[0] = tc.header
				checkValidate(t, b, delta != 0)
			}
			if byte(tc.shape) != tc.header>>4 {
				t.Fatalf("shape %d has header %x", tc.shape, tc.header)
			}
		})
	}
	for _, shape := range []byte{2, 4, 6, 7, 15} {
		b := make([]byte, 7)
		b[0] = shape << 4
		if err := Validate(b); !errors.Is(err, ErrShapeTypeInvalid) {
			t.Errorf("reserved shape %d: %v", shape, err)
		}
	}
	for _, b := range [][]byte{nil, {}} {
		if err := Validate(b); !errors.Is(err, ErrInformationTooShort) {
			t.Errorf("empty: %v", err)
		}
	}
	for n := 0; n <= 15; n++ {
		b := make([]byte, 1+6*n)
		b[0] = 0x50 | byte(n)
		checkValidate(t, b, n < 3)
	}
}

func checkValidate(t *testing.T, b []byte, invalid bool) {
	t.Helper()
	before := bytes.Clone(b)
	err := Validate(b)
	if (err != nil) != invalid {
		t.Fatalf("Validate(%x) = %v, want invalid=%v", b, err, invalid)
	}
	if err != nil && !strings.HasPrefix(err.Error(), "gad: ") {
		t.Errorf("missing error prefix: %v", err)
	}
	if !bytes.Equal(before, b) {
		t.Fatalf("Validate changed %x to %x", before, b)
	}
}

// TS 23.032 V19.0.0 §§8.5, 8.6, 8.12–8.15.
func TestSenderLCSVelocityTypes(t *testing.T) {
	lengths := map[byte]int{0: 4, 1: 5, 2: 5, 3: 7}
	for typ := byte(0); typ < 16; typ++ {
		for n := 0; n <= 9; n++ {
			v := make([]byte, n)
			if n > 0 {
				v[0] = typ<<4 | 3
			}
			var want error
			switch {
			case n == 0:
				want = ErrVelocityLengthInvalid
			case typ > 3:
				want = ErrVelocityTypeInvalid
			case n != lengths[typ]:
				want = ErrVelocityLengthInvalid
			}
			if err := ValidateVelocity(v); !errors.Is(err, want) {
				t.Errorf("type %d length %d: %v, want %v", typ, n, err, want)
			}
		}
	}
}

// TS 23.032 V19.0.0 §8.6 Table 3 and §§8.12–8.15.
func TestReceiverVelocityType(t *testing.T) {
	lengths := [4]int{4, 5, 5, 7}
	for typ := byte(0); typ < 16; typ++ {
		n := 4
		if typ < 4 {
			n = lengths[typ]
		}
		v := make([]byte, n)
		v[0] = typ<<4 | 1
		if typ == 1 || typ == 3 {
			v[0] |= 2
		}
		var want error
		if typ > 3 {
			want = ErrVelocityTypeInvalid
		}
		if err := ValidateVelocity(v); !errors.Is(err, want) {
			t.Errorf("type %d: %v, want %v", typ, err, want)
		}
	}
	for _, tc := range []struct {
		typ    VelocityType
		header byte
		n      int
	}{
		{VelocityHorizontal, 0x00, 4},
		{VelocityHorizontalVertical, 0x10, 5},
		{VelocityHorizontalUncertainty, 0x20, 5},
		{VelocityHorizontalVerticalUncertainty, 0x30, 7},
	} {
		v := make([]byte, tc.n)
		v[0] = byte(tc.typ) << 4
		if v[0] != tc.header {
			t.Errorf("velocity type %d: header %x", tc.typ, v[0])
		}
		if err := ValidateVelocity(v); err != nil {
			t.Fatal(err)
		}
	}
}

// TS 23.032 V19.0.0 §§6.3a, 6.5, 6.7, 7.3.
func TestSenderLCSGeographicalCoding(t *testing.T) {
	coding := []struct {
		shape         byte
		n             int
		zeros, angles []int
	}{
		{1, 8, []int{7}, nil}, {3, 11, []int{7, 8, 10}, []int{9}}, {9, 14, []int{9, 10, 12, 13}, []int{11}},
		{10, 13, []int{9, 12}, []int{10, 11}}, {11, 13, []int{12}, []int{11}},
		{12, 18, []int{9, 15, 17}, []int{14}}, {13, 13, nil, []int{11}}, {14, 18, []int{9}, []int{14}},
	}
	for _, c := range coding {
		for _, i := range c.zeros {
			for _, mask := range []byte{0x80, 0x40} {
				if i != 9 || (c.shape != 12 && c.shape != 14) {
					if mask == 0x40 {
						continue
					}
				}
				d := make([]byte, c.n)
				d[0] = c.shape << 4
				d[i] = mask
				checkValidate(t, d, true)

			}
		}
		for _, i := range c.angles {
			for _, v := range []byte{0, 179, 180, 255} {
				d := make([]byte, c.n)
				d[0] = c.shape << 4
				d[i] = v
				wantInvalid := false
				if v > 179 {
					wantInvalid = true
				}
				checkValidate(t, d, wantInvalid)

			}
		}
	}
	for flags := byte(0); flags < 4; flags++ {
		d := make([]byte, 18)
		d[0] = 0xe0
		d[15] = (flags & 1) << 7
		d[17] = (flags & 2) << 6
		checkValidate(t, d, false)
	}
	for _, shape := range []byte{11, 13} {
		for uncertainty := 0; uncertainty < 256; uncertainty++ {
			d := make([]byte, 13)
			d[0] = shape << 4
			d[9] = byte(uncertainty)
			d[10] = byte(uncertainty)
			d[12] = 127
			if shape == 13 {
				d[12] |= 0x80
			}
			checkValidate(t, d, false)
		}
	}
	for _, shape := range []byte{12, 14} {
		for _, alt := range []int32{-64001, -64000, 1280000, 1280001} {
			d := make([]byte, 18)
			d[0] = shape << 4
			n := uint32(alt) & 0x3fffff
			d[9] = byte(n >> 16)
			d[10] = byte(n >> 8)
			d[11] = byte(n)
			wantInvalid := false
			if alt < -64000 || alt > 1280000 {
				wantInvalid = true
			}
			checkValidate(t, d, wantInvalid)
		}
	}
	d := make([]byte, 11)
	d[0] = 0x3f
	d[10] = 127
	checkValidate(t, d, false)
}
