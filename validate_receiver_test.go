package gad

import (
	"bytes"
	"fmt"
	"testing"
)

// TS 23.032 V19.0.0 §§6.3a, 6.5, 6.7, 7.3.
func TestReceiverGeographicalShapes(t *testing.T) {
	lengths := map[byte]int{0: 7, 1: 8, 3: 11, 5: 19, 8: 9, 9: 14, 10: 13, 11: 13, 12: 18, 13: 13, 14: 18}
	for shape := byte(0); shape < 16; shape++ {
		length, defined := lengths[shape]
		if !defined {
			length = 7
		}
		v := make([]byte, length)
		v[0] = shape << 4
		if shape == 5 {
			v[0] |= 3
		}
		t.Run(fmt.Sprintf("shape=%d", shape), func(t *testing.T) {
			err := Validate(v)
			if defined {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				t.Fatalf("want %v, got %v", "rejection", err)
			}
			if !defined {
				return
			}
			for _, d := range [][]byte{v[:len(v)-1], append(bytes.Clone(v), 0)} {
				if err := Validate(d); err == nil {
					t.Errorf("wrong length %d: %v", len(d), err)
				}
			}
			if shape != 5 {
				d := bytes.Clone(v)
				d[0] |= 0xf
				if err := Validate(d); err != nil {
					t.Errorf("unspecified header spare bits: %v", err)
				}
			}
			var zeroBits []int
			var angles []int
			switch shape {
			case 1:
				zeroBits = []int{7}
			case 3:
				zeroBits = []int{7, 8, 10}
				angles = []int{9}
			case 9:
				zeroBits = []int{9, 10, 12, 13}
				angles = []int{11}
			case 10:
				zeroBits = []int{9, 12}
				angles = []int{10, 11}
			case 11:
				zeroBits = []int{12}
				angles = []int{11}
			case 12:
				zeroBits = []int{9, 15, 17}
				angles = []int{14}
			case 13:
				angles = []int{11}
			case 14:
				zeroBits = []int{9}
				angles = []int{14}
			}
			for _, i := range zeroBits {
				d := bytes.Clone(v)
				d[i] |= 0x80
				if err := Validate(d); err == nil {
					t.Errorf("explicit zero bit octet %d: %v", i+1, err)
				}
			}
			if shape == 12 || shape == 14 {
				d := bytes.Clone(v)
				d[9] |= 0x40
				if err := Validate(d); err == nil {
					t.Errorf("second explicit zero altitude bit: %v", err)
				}
			}
			for _, i := range angles {
				for _, angle := range []byte{179, 180, 255} {
					d := bytes.Clone(v)
					d[i] = angle
					err := Validate(d)
					if angle == 179 {
						if err != nil {
							t.Errorf("angle179: %v", err)
						}
					} else if err == nil {
						t.Errorf("angle%d: %v", angle, err)
					}
				}
			}
			confidences := map[byte][]int{3: {10}, 9: {13}, 10: {12}, 11: {12}, 12: {15, 17}, 13: {12}, 14: {15, 17}}
			for _, i := range confidences[shape] {
				for _, c := range []byte{100, 101, 127} {
					d := bytes.Clone(v)
					d[i] = c
					if err := Validate(d); err != nil {
						t.Errorf("confidence%d: %v", c, err)
					}
				}
			}
			if shape == 5 {
				for _, n := range []byte{2, 3, 15} {
					d := make([]byte, 1+6*int(n))
					d[0] = 0x50 | n
					err := Validate(d)
					if n >= 3 {
						if err != nil {
							t.Errorf("polygon%d: %v", n, err)
						}
					} else if err == nil {
						t.Errorf("polygon%d: %v", n, err)
					}
				}
			}
			if shape == 8 || shape == 9 {
				d := bytes.Clone(v)
				d[7] = 0x80
				if err := Validate(d); err != nil {
					t.Errorf("altitude direction: %v", err)
				}
			}
			if shape == 12 || shape == 14 {
				for _, altitude := range []int32{-64001, -64000, 1280000, 1280001} {
					d := bytes.Clone(v)
					raw := uint32(altitude) & 0x3fffff
					d[9], d[10], d[11] = byte(raw>>16), byte(raw>>8), byte(raw)
					err := Validate(d)
					if altitude >= -64000 && altitude <= 1280000 {
						if err != nil {
							t.Errorf("altitude%d: %v", altitude, err)
						}
					} else if err == nil {
						t.Errorf("altitude%d: %v", altitude, err)
					}
				}
			}
			if shape >= 11 {
				uncertainties := []int{9, 10}
				if shape == 12 || shape == 14 {
					uncertainties = []int{12, 13, 16}
				}
				for _, i := range uncertainties {
					for code := range 256 {
						d := bytes.Clone(v)
						d[i] = byte(code)
						if err := Validate(d); err != nil {
							t.Fatalf("uncertainty octet%d code%d: %v", i+1, code, err)
						}
					}
				}
				for _, coordinate := range []byte{0, 0x7f, 0x80, 0xff} {
					d := bytes.Clone(v)
					for i := 1; i <= 8; i++ {
						d[i] = coordinate
					}
					if err := Validate(d); err != nil {
						t.Errorf("coordinate pattern %x: %v", coordinate, err)
					}
				}
			}
			if shape == 13 {
				d := bytes.Clone(v)
				d[12] = 0xff
				if err := Validate(d); err != nil {
					t.Errorf("scalable U: %v", err)
				}
			}
			if shape == 14 {
				for flags := range 4 {
					d := bytes.Clone(v)
					d[15] = byte(flags&1) << 7
					d[17] = byte(flags>>1) << 7
					if err := Validate(d); err != nil {
						t.Errorf("scalable HU/VU%d: %v", flags, err)
					}
				}
			}
		})
	}
}
