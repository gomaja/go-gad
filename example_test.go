package gad_test

import (
	"fmt"

	"github.com/gomaja/go-gad"
)

func ExampleDecode() {
	// TS 23.032 V19.0.0 §7.3.1: 45° north, 90° west.
	location, err := gad.Decode([]byte{0x00, 0x40, 0x00, 0x00, 0xc0, 0x00, 0x00})
	if err != nil {
		fmt.Println(err)
		return
	}
	point := location.(gad.EllipsoidPoint)
	fmt.Println(point.Shape())
	fmt.Printf("latitude %.1f°, longitude %.1f°\n", point.Latitude.Degrees(), point.Longitude.Degrees())
	// Output:
	// ellipsoid point
	// latitude 45.0°, longitude -90.0°
}

func ExampleEllipsoidPointUncertaintyCircle_Encode() {
	point, err := gad.PointFromDegrees(45, -90)
	if err != nil {
		fmt.Println(err)
		return
	}
	uncertainty, err := gad.UncertaintyFromMetres(10)
	if err != nil {
		fmt.Println(err)
		return
	}
	location := gad.EllipsoidPointUncertaintyCircle{Point: point, Uncertainty: uncertainty}
	encoded, err := location.Encode()
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%x\n", encoded)
	fmt.Printf("uncertainty code %d covers at least 10 metres\n", uncertainty)
	// Output:
	// 10400000c0000008
	// uncertainty code 8 covers at least 10 metres
}

func ExampleDecodeVelocity() {
	// TS 23.032 V19.0.0 §8.12: bearing 270°, horizontal speed 50 km/h.
	velocity, err := gad.DecodeVelocity([]byte{0x01, 0x0e, 0x00, 0x32})
	if err != nil {
		fmt.Println(err)
		return
	}
	horizontal := velocity.(gad.HorizontalVelocity)
	fmt.Println(horizontal.Type())
	fmt.Printf("bearing %.0f°, speed %.0f km/h\n", horizontal.Bearing.Degrees(), horizontal.HorizontalSpeed.Kmh())
	// Output:
	// horizontal velocity
	// bearing 270°, speed 50 km/h
}
