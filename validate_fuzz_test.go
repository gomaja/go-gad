package gad

import (
	"bytes"
	"testing"
)

// TS 23.032 V19.0.0 §7.3: validation preserves octets and shape acceptance.
func FuzzValidateDecodeEncode(f *testing.F) {
	for _, shape := range []byte{0, 1, 3, 5, 8, 9, 10, 11, 12, 13, 14, 15} {
		lengths := map[byte]int{0: 7, 1: 8, 3: 11, 5: 19, 8: 9, 9: 14, 10: 13, 11: 13, 12: 18, 13: 13, 14: 18, 15: 7}
		v := make([]byte, lengths[shape])
		v[0] = shape << 4
		if shape == 5 {
			v[0] |= 3
		}
		f.Add(v)
		f.Add(v[:len(v)-1])
		f.Add(append(bytes.Clone(v), 0))
	}
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		checkValidate(t, b, !referenceValidGeographicalInformation(b))
		decoded, err := Decode(b)
		if err != nil {
			return
		}
		encoded, err := decoded.Encode()
		if err != nil {
			if Validate(b) == nil {
				t.Fatalf("valid description cannot encode: %x: %v", b, err)
			}
			return
		}
		if err := Validate(encoded); err != nil {
			t.Fatalf("encoded description invalid: %x: %v", encoded, err)
		}
		again, err := Decode(encoded)
		if err != nil {
			t.Fatal(err)
		}
		stable, err := again.Encode()
		if err != nil || !bytes.Equal(stable, encoded) {
			t.Fatalf("unstable encoding: %x => %x: %v", encoded, stable, err)
		}
	})
}

// TS 23.032 V19.0.0 §7.3: boolean acceptance reference.
func referenceValidGeographicalInformation(d []byte) bool {
	if len(d) == 0 {
		return false
	}
	switch d[0] >> 4 {
	case 0: // §7.3.1, figure 5.
		return len(d) == 7
	case 1: // §7.3.2, figure 6.
		return len(d) == 8 && d[7]&0x80 == 0
	case 3: // §7.3.3, figure 6a.
		return len(d) == 11 && (d[7]|d[8]|d[10])&0x80 == 0 && d[9] <= 179
	case 5: // §7.3.4, figure 7.
		n := int(d[0] & 0xf)
		return n >= 3 && n <= 15 && len(d) == 1+6*n
	case 8: // §7.3.5, figure 8; altitude includes a direction bit.
		return len(d) == 9
	case 9: // §7.3.6, figure 9.
		return len(d) == 14 && (d[9]|d[10]|d[12]|d[13])&0x80 == 0 && d[11] <= 179
	case 10: // §7.3.7, figure 10; arc angles follow §6.7.
		return len(d) == 13 && (d[9]|d[12])&0x80 == 0 && d[10] <= 179 && d[11] <= 179
	case 11: // §7.3.3a, figure 7.3.3a-1.
		return len(d) == 13 && d[11] <= 179 && d[12]&0x80 == 0
	case 13: // §7.3.3b, figure 7.3.3b-1; bit 8 selects uncertainty range.
		return len(d) == 13 && d[11] <= 179
	case 12, 14: // §§7.3.6a,7.3.6b, figures 7.3.6a-1 and 7.3.6b-1.
		if len(d) != 18 || d[9]&0xc0 != 0 || d[14] > 179 {
			return false
		}
		if d[0]>>4 == 12 && (d[15]|d[17])&0x80 != 0 {
			return false
		}
		// §6.3a: signed 22-bit altitude codes range from -64000 to 1280000.
		altitude := int32(d[9]&0x3f)<<16 | int32(d[10])<<8 | int32(d[11])
		if altitude&0x200000 != 0 {
			altitude -= 1 << 22
		}
		return altitude >= -64000 && altitude <= 1280000
	default:
		return false
	}
}
