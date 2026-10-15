package gad

import (
	"bytes"
	"testing"
)

// TS 23.032 V19.0.0 §§8.6, 8.8, 8.12–8.15.
func TestVelocityStrict(t *testing.T) {
	for typ, n := range []int{4, 5, 5, 7} {
		b := make([]byte, n)
		b[0] = byte(typ) << 4
		if _, err := DecodeVelocity(b); err != nil {
			t.Fatal(err)
		}
		for _, mask := range []byte{4, 8} {
			c := append([]byte(nil), b...)
			c[0] |= mask
			_, err := DecodeVelocity(c)
			assertFieldError(t, err, ErrSpareBit, "Spare", 1)
		}
		if typ%2 == 0 {
			c := append([]byte(nil), b...)
			c[0] |= 2
			_, err := DecodeVelocity(c)
			assertFieldError(t, err, ErrSpareBit, "Spare", 1)
		}
		for _, length := range []int{n - 1, n + 1} {
			c := make([]byte, length)
			copy(c, b)
			_, err := DecodeVelocity(c)
			names := [...]string{
				"horizontal velocity",
				"horizontal with vertical velocity",
				"horizontal velocity with uncertainty",
				"horizontal with vertical velocity and uncertainty",
			}
			assertFieldError(t, err, ErrLength, names[typ], min(length, n)+1)
		}
		b[0] |= 1
		b[1] = 103 // 359 degrees.
		v, err := DecodeVelocity(b)
		if err != nil {
			t.Fatal(err)
		}
		got, err := v.Encode()
		if err != nil || string(got) != string(b) {
			t.Fatalf("%x %v", got, err)
		}
		b[1] = 104
		_, err = DecodeVelocity(b)
		assertFieldError(t, err, ErrOutOfRange, "Bearing", 1)
	}
	for typ := 4; typ < 16; typ++ {
		_, err := DecodeVelocity([]byte{byte(typ) << 4})
		assertFieldError(t, err, ErrVelocityTypeInvalid, "Velocity type", 1)
	}
	_, err := DecodeVelocity(nil)
	assertFieldError(t, err, ErrLength, "Velocity", 1)
}

// TS 23.032 V19.0.0 §8.8: unused bearing codes never encode.
func TestVelocityEncodeBearing(t *testing.T) {
	for _, bearing := range []Bearing{359, 360, 511, 65535} {
		for _, v := range []Velocity{
			HorizontalVelocity{Bearing: bearing, HorizontalSpeed: 65535},
			HorizontalWithVerticalVelocity{Bearing: bearing, HorizontalSpeed: 65535, VerticalSpeed: 255, Downward: true},
			HorizontalVelocityWithUncertainty{Bearing: bearing, HorizontalSpeed: 65535, UncertaintySpeed: 255},
			HorizontalWithVerticalVelocityAndUncertainty{Bearing: bearing, HorizontalSpeed: 65535, VerticalSpeed: 255, Downward: true, HorizontalUncertaintySpeed: 255, VerticalUncertaintySpeed: 255},
		} {
			b, err := v.Encode()
			if bearing > 359 {
				assertFieldError(t, err, ErrOutOfRange, "Bearing", 1)
				if b != nil {
					t.Fatal("invalid bytes")
				}
			} else if err != nil {
				t.Fatal(err)
			} else {
				decoded, err := DecodeVelocity(b)
				if err != nil {
					t.Fatal(err)
				}
				again, err := decoded.Encode()
				if err != nil || !bytes.Equal(again, b) {
					t.Fatalf("velocity round trip = %x, %v; want %x", again, err, b)
				}
			}
		}
	}
}

// TS 23.032 V19.0.0 §§8.6, 8.8, 8.12–8.15: structural checks precede fields.
func TestVelocityErrorOrder(t *testing.T) {
	for _, tc := range []struct {
		name     string
		b        []byte
		sentinel error
		field    string
		octet    int
	}{
		{"empty", nil, ErrLength, "Velocity", 1},
		{"reserved before length and spare", []byte{0xff}, ErrVelocityTypeInvalid, "Velocity type", 1},
		{"length before spare and bearing", []byte{0x0f, 0xff, 0}, ErrLength, "horizontal velocity", 4},
		{"unexpected octet before spare and bearing", []byte{0x0f, 0xff, 0, 0, 0}, ErrLength, "horizontal velocity", 5},
		{"spare before bearing", []byte{0x0f, 0xff, 0, 0}, ErrSpareBit, "Spare", 1},
		{"vertical spare before bearing", []byte{0x1f, 0xff, 0, 0, 0}, ErrSpareBit, "Spare", 1},
		{"uncertainty spare before bearing", []byte{0x2f, 0xff, 0, 0, 0}, ErrSpareBit, "Spare", 1},
		{"vertical uncertainty spare before bearing", []byte{0x3f, 0xff, 0, 0, 0, 0, 0}, ErrSpareBit, "Spare", 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeVelocity(tc.b)
			assertFieldError(t, err, tc.sentinel, tc.field, tc.octet)
			assertFieldError(t, ValidateVelocity(tc.b), tc.sentinel, tc.field, tc.octet)
		})
	}
}
