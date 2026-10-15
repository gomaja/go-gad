package gad

import (
	"bytes"
	"testing"
)

// referenceShapeAccepts applies TS 23.032 V19.0.0 Table 2a and the
// figures in §7.3 directly, independently of the codec's validation helpers.
func referenceShapeAccepts(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	shape := b[0] >> 4
	var size int
	switch shape {
	case 0:
		size = 7
	case 1:
		size = 8
	case 3:
		size = 11
	case 5:
		n := int(b[0] & 15)
		if n < 3 {
			return false
		}
		size = 1 + 6*n
	case 8:
		size = 9
	case 9:
		size = 14
	case 10, 11, 13:
		size = 13
	case 12, 14:
		size = 18
	default:
		return false
	}
	if len(b) != size || (shape != 5 && b[0]&15 != 0) {
		return false
	}
	switch shape {
	case 1:
		return b[7] < 128
	case 3:
		return b[7] < 128 && b[8] < 128 && b[9] < 180 && b[10] < 128
	case 9:
		return b[9] < 128 && b[10] < 128 && b[11] < 180 && b[12] < 128 && b[13] < 128
	case 10:
		return b[9] < 128 && b[10] < 180 && b[11] < 180 && b[12] < 128
	case 11:
		return b[11] < 180 && b[12] < 128
	case 13:
		return b[11] < 180 // Bit 8 of octet 13 is U, not spare.
	case 12, 14:
		if b[9]&0xc0 != 0 || b[14] >= 180 {
			return false
		}
		// §6.3a: sign-extend the 22-bit two's complement altitude.
		altitude := int32(b[9])<<16 | int32(b[10])<<8 | int32(b[11])
		if altitude&0x200000 != 0 {
			altitude -= 0x400000
		}
		if altitude < -64000 || altitude > 1280000 {
			return false
		}
		return shape == 14 || (b[15] < 128 && b[17] < 128)
	}
	return true
}

// referenceVelocityAccepts applies TS 23.032 V19.0.0 Table 3 and
// §§8.8, 8.12–8.15 without calling production validation or conversion code.
func referenceVelocityAccepts(b []byte) bool {
	if len(b) == 0 {
		return false
	}
	typ := b[0] >> 4
	var size int
	switch typ {
	case 0:
		size = 4
	case 1, 2:
		size = 5
	case 3:
		size = 7
	default:
		return false
	}
	if len(b) != size {
		return false
	}
	if b[0]&0x0c != 0 || ((typ == 0 || typ == 2) && b[0]&2 != 0) {
		return false
	}
	return int(b[0]&1)*256+int(b[1]) <= 359
}

func FuzzDecode(f *testing.F) {
	for _, b := range shapeFuzzSeeds() {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		before := bytes.Clone(b)
		d, err := Decode(b)
		validationErr := Validate(b)
		if (err == nil) != referenceShapeAccepts(b) {
			t.Fatalf("Decode(%x) acceptance differs from reference: %v", b, err)
		}
		if (err == nil) != (validationErr == nil) {
			t.Fatalf("Decode error %v differs from Validate error %v", err, validationErr)
		}
		if err != nil {
			if err.Error() != validationErr.Error() {
				t.Fatalf("Decode error %q differs from Validate error %q", err, validationErr)
			}
		} else {
			encoded, encodeErr := d.Encode()
			if encodeErr != nil || !bytes.Equal(encoded, b) {
				t.Fatalf("Decode(%x).Encode() = %x, %v", b, encoded, encodeErr)
			}
		}
		if !bytes.Equal(b, before) {
			t.Fatalf("decoder modified input: %x became %x", before, b)
		}
	})
}

func FuzzDecodeVelocity(f *testing.F) {
	for _, b := range velocityFuzzSeeds() {
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		before := bytes.Clone(b)
		v, err := DecodeVelocity(b)
		validationErr := ValidateVelocity(b)
		if (err == nil) != referenceVelocityAccepts(b) {
			t.Fatalf("DecodeVelocity(%x) acceptance differs from reference: %v", b, err)
		}
		if (err == nil) != (validationErr == nil) {
			t.Fatalf("DecodeVelocity error %v differs from ValidateVelocity error %v", err, validationErr)
		}
		if err != nil {
			if err.Error() != validationErr.Error() {
				t.Fatalf("DecodeVelocity error %q differs from ValidateVelocity error %q", err, validationErr)
			}
		} else {
			encoded, encodeErr := v.Encode()
			if encodeErr != nil || !bytes.Equal(encoded, b) {
				t.Fatalf("DecodeVelocity(%x).Encode() = %x, %v", b, encoded, encodeErr)
			}
		}
		if !bytes.Equal(b, before) {
			t.Fatalf("decoder modified input: %x became %x", before, b)
		}
	})
}

func shapeFuzzSeeds() [][]byte {
	seeds := [][]byte{nil, {}, {0}, {0xff}}
	// Numeric type and length values are copied from Table 2a, not the codec.
	for _, row := range []struct{ typ, size int }{
		{0, 7}, {1, 8}, {3, 11}, {5, 19}, {5, 91}, {8, 9},
		{9, 14}, {10, 13}, {11, 13}, {12, 18}, {13, 13}, {14, 18},
	} {
		b := make([]byte, row.size)
		b[0] = byte(row.typ << 4)
		if row.typ == 5 {
			b[0] |= byte((row.size - 1) / 6)
		}
		seeds = append(seeds, bytes.Clone(b), bytes.Clone(b[:len(b)-1]), append(bytes.Clone(b), 0))
		// Individually toggle every bit to cover spare bits, signs and flags.
		for i := range b {
			for bit := uint(0); bit < 8; bit++ {
				mutated := bytes.Clone(b)
				mutated[i] ^= 1 << bit
				seeds = append(seeds, mutated)
			}
		}
		// Maximum unstructured values exercise several simultaneous defects.
		maximal := bytes.Repeat([]byte{255}, len(b))
		maximal[0] = b[0]
		seeds = append(seeds, maximal)
		var angles, confidences []int // Zero-based field positions from §7.3 figures.
		switch row.typ {
		case 3:
			angles, confidences = []int{9}, []int{10}
		case 9:
			angles, confidences = []int{11}, []int{13}
		case 10:
			angles, confidences = []int{10, 11}, []int{12}
		case 11, 13:
			angles, confidences = []int{11}, []int{12}
		case 12, 14:
			angles, confidences = []int{14}, []int{15, 17}
		}
		for _, position := range angles {
			for _, value := range []byte{179, 180, 255} {
				mutated := bytes.Clone(b)
				mutated[position] = value
				seeds = append(seeds, mutated)
			}
		}
		for _, position := range confidences {
			for _, value := range []byte{100, 101, 127, 128, 255} {
				mutated := bytes.Clone(b)
				mutated[position] = value
				seeds = append(seeds, mutated)
			}
		}
	}
	for _, typ := range []byte{2, 4, 6, 7, 15} {
		seeds = append(seeds, []byte{typ << 4})
	}
	for _, typ := range []byte{12, 14} {
		for _, altitude := range []int32{-64001, -64000, -1, 0, 1280000, 1280001} {
			b := make([]byte, 18)
			b[0] = typ << 4
			n := uint32(altitude) & 0x3fffff
			b[9], b[10], b[11] = byte(n>>16), byte(n>>8), byte(n)
			seeds = append(seeds, b)
		}
	}
	return seeds
}

func velocityFuzzSeeds() [][]byte {
	seeds := [][]byte{nil, {}, {0}, {0xff}}
	for typ, size := range []int{4, 5, 5, 7} {
		for _, bearing := range []int{0, 255, 256, 359, 360, 511} {
			b := bytes.Repeat([]byte{255}, size)
			b[0], b[1] = byte(typ<<4|bearing>>8), byte(bearing)
			seeds = append(seeds, bytes.Clone(b), bytes.Clone(b[:size-1]), append(bytes.Clone(b), 0))
			for bit := uint(0); bit < 4; bit++ {
				mutated := bytes.Clone(b)
				mutated[0] ^= 1 << bit
				seeds = append(seeds, mutated)
			}
		}
	}
	for typ := byte(4); typ < 16; typ++ {
		seeds = append(seeds, []byte{typ << 4, 0, 0, 0, 0, 0, 0})
	}
	return seeds
}
