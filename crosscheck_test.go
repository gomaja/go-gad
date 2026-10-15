package gad_test

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gomaja/go-gad"
)

type crosscheckVector struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Type      int    `json:"type"`
	Valid     bool   `json:"valid"`
	Hex       string `json:"hex"`
	Wireshark struct {
		Malformed bool              `json:"malformed"`
		Expert    []json.RawMessage `json:"expert"`
		Fields    [][]string        `json:"fields"`
	} `json:"wireshark"`
	Libosmocore *struct {
		RawReadRC   *int            `json:"raw_read_rc"`
		RawFields   json.RawMessage `json:"raw_fields"`
		DecodeRC    *int            `json:"decode_rc"`
		Decoded     json.RawMessage `json:"decoded"`
		RawError    string          `json:"raw_error"`
		DecodeError string          `json:"decode_error"`
	} `json:"libosmocore"`
}

type crosscheckField struct {
	name       string
	show       string
	conversion string
	skip       string
}

type crosscheckFields []crosscheckField

func (f *crosscheckFields) integer(name string, value int64) {
	*f = append(*f, crosscheckField{name: name, show: strconv.FormatInt(value, 10)})
}

func (f *crosscheckFields) flag(name string, value bool) {
	var n int64
	if value {
		n = 1
	}
	f.integer(name, n)
}

func (f *crosscheckFields) annotation(name string, code int64, conversion, skip string) {
	*f = append(*f, crosscheckField{name: name, show: strconv.FormatInt(code, 10), conversion: conversion, skip: skip})
}

func (f *crosscheckFields) point(p gad.Point) {
	f.flag("sign_of_latitude", p.Latitude.South)
	// W1 drops a single-point southern sign; W2 rescales latitude by 2^23-1.
	f.annotation("deg_of_latitude", int64(p.Latitude.N), "", "W1/W2")
	// W3 rescales longitude by 2^24-1.
	f.annotation("deg_of_longitude", int64(p.Longitude), "", "W3")
}

func (f *crosscheckFields) highPoint(p gad.HighAccuracyPoint) {
	// W5 interprets HA latitude as sign/magnitude (and rescales by 2^31-1).
	f.annotation("hig_acc_deg_of_lat", int64(p.Latitude), "", "W5")
	// W6 prefixes a second minus sign (and rescales by 2^31-1).
	f.annotation("high_acc_deg_of_long", int64(p.Longitude), "", "W6")
}

func (f *crosscheckFields) altitude(a gad.Altitude) {
	f.flag("D", a.Depth)
	f.integer("altitude", int64(a.N))
}

func (f *crosscheckFields) orientation(o gad.Orientation) {
	// W4 doubles the major-axis orientation, unlike TS 23.032 §7.3.3.
	f.annotation("orientation_of_major_axis", int64(o), "", "W4")
}

func (f *crosscheckFields) ellipse(major, minor gad.Uncertainty, orientation gad.Orientation) {
	f.annotation("uncertainty_semi_major", int64(major), fmt.Sprintf("%.1f m", major.Metres()), "")
	f.annotation("uncertainty_semi_minor", int64(minor), fmt.Sprintf("%.1f m", minor.Metres()), "")
	f.orientation(orientation)
}

func (f *crosscheckFields) highEllipse(major, minor gad.ScalableUncertainty, orientation gad.Orientation) {
	for _, axis := range []struct {
		name        string
		uncertainty gad.ScalableUncertainty
	}{
		{"high_acc_uncertainty_semi_major", major}, {"high_acc_uncertainty_semi_minor", minor},
	} {
		if axis.uncertainty.Extended {
			// W7 ignores U/HU when converting both horizontal uncertainty axes.
			f.annotation(axis.name, int64(axis.uncertainty.K), "", "W7")
		} else {
			f.annotation(axis.name, int64(axis.uncertainty.K), fmt.Sprintf("%.5f m", axis.uncertainty.Metres()), "")
		}
	}
	f.orientation(orientation)
}

func crosscheckExpected(value any) crosscheckFields {
	var f crosscheckFields
	if d, ok := value.(gad.Description); ok {
		f.integer("location_estimate", int64(d.Shape()))
	}
	if v, ok := value.(gad.Velocity); ok {
		f.integer("velocity_type", int64(v.Type()))
		f.integer("spare_bits", 0)
	}
	switch v := value.(type) {
	case gad.EllipsoidPoint:
		f.point(v.Point)
	case gad.EllipsoidPointUncertaintyCircle:
		f.point(v.Point)
		f.annotation("uncertainty_code", int64(v.Uncertainty), fmt.Sprintf("%.1f m", v.Uncertainty.Metres()), "")
	case gad.EllipsoidPointUncertaintyEllipse:
		f.point(v.Point)
		f.ellipse(v.SemiMajor, v.SemiMinor, v.Orientation)
		f.integer("confidence", int64(v.Confidence))
	case gad.Polygon:
		f.integer("no_of_points", int64(len(v.Points)))
		for _, p := range v.Points {
			f.point(p)
		}
	case gad.EllipsoidPointAltitude:
		f.point(v.Point)
		f.altitude(v.Altitude)
	case gad.EllipsoidPointAltitudeUncertaintyEllipsoid:
		f.point(v.Point)
		f.altitude(v.Altitude)
		f.ellipse(v.SemiMajor, v.SemiMinor, v.Orientation)
		f.annotation("uncertainty_altitude", int64(v.UncertaintyAltitude), fmt.Sprintf("%.1f m", v.UncertaintyAltitude.Metres()), "")
		f.integer("confidence", int64(v.Confidence))
	case gad.EllipsoidArc:
		f.point(v.Point)
		f.integer("inner_radius", int64(v.InnerRadius))
		f.integer("uncertainty_radius", int64(v.UncertaintyRadius))
		f.integer("offset_angle", int64(v.OffsetAngle))
		f.integer("included_angle", int64(v.IncludedAngle))
		f.integer("confidence", int64(v.Confidence))
	case gad.HighAccuracyEllipsoidPointUncertaintyEllipse:
		f.highPoint(v.HighAccuracyPoint)
		f.highEllipse(gad.ScalableUncertainty{K: uint8(v.SemiMajor)}, gad.ScalableUncertainty{K: uint8(v.SemiMinor)}, v.Orientation)
		f.integer("confidence", int64(v.Confidence))
	case gad.HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid:
		f.highPoint(v.HighAccuracyPoint)
		f.annotation("high_acc_alt", int64(v.Altitude), fmt.Sprintf("%.1f m", v.Altitude.Metres()), "")
		f.highEllipse(gad.ScalableUncertainty{K: uint8(v.SemiMajor)}, gad.ScalableUncertainty{K: uint8(v.SemiMinor)}, v.Orientation)
		f.integer("horizontal_confidence", int64(v.HorizontalConfidence))
		// W8 uses classic altitude uncertainty instead of §7.3.6a NOTE's HA formula.
		f.annotation("high_acc_uncertainty_alt", int64(v.UncertaintyAltitude), "", "W8")
		f.integer("vertical_confidence", int64(v.VerticalConfidence))
	case gad.HighAccuracyEllipsoidPointScalableUncertaintyEllipse:
		f.highPoint(v.HighAccuracyPoint)
		f.highEllipse(v.SemiMajor, v.SemiMinor, v.Orientation)
		f.integer("confidence", int64(v.Confidence))
		f.flag("uncertainty_range", v.SemiMajor.Extended)
	case gad.HighAccuracyEllipsoidPointAltitudeScalableUncertaintyEllipsoid:
		f.highPoint(v.HighAccuracyPoint)
		f.annotation("high_acc_alt", int64(v.Altitude), fmt.Sprintf("%.1f m", v.Altitude.Metres()), "")
		f.highEllipse(v.SemiMajor, v.SemiMinor, v.Orientation)
		f.integer("horizontal_confidence", int64(v.HorizontalConfidence))
		f.flag("horizontal_uncertainty_range", v.SemiMajor.Extended)
		// W8 also ignores VU and applies the classic altitude formula to shape 14.
		f.annotation("high_acc_uncertainty_alt", int64(v.UncertaintyAltitude.K), "", "W8")
		f.integer("vertical_confidence", int64(v.VerticalConfidence))
		f.flag("vertical_uncertainty_range", v.UncertaintyAltitude.Extended)
	case gad.HorizontalVelocity:
		f.integer("bearing", int64(v.Bearing))
		f.integer("horizontal_velocity", int64(v.HorizontalSpeed))
	case gad.HorizontalWithVerticalVelocity:
		f = append(f, crosscheckField{name: "d", show: map[bool]string{false: "False", true: "True"}[v.Downward]})
		f.integer("bearing", int64(v.Bearing))
		f.integer("horizontal_velocity", int64(v.HorizontalSpeed))
		f.integer("vertical_speed", int64(v.VerticalSpeed))
	case gad.HorizontalVelocityWithUncertainty:
		f.integer("bearing", int64(v.Bearing))
		f.integer("horizontal_velocity", int64(v.HorizontalSpeed))
		f.integer("uncertainty_speed", int64(v.UncertaintySpeed))
	case gad.HorizontalWithVerticalVelocityAndUncertainty:
		f = append(f, crosscheckField{name: "d", show: map[bool]string{false: "False", true: "True"}[v.Downward]})
		f.integer("bearing", int64(v.Bearing))
		f.integer("horizontal_velocity", int64(v.HorizontalSpeed))
		f.integer("vertical_speed", int64(v.VerticalSpeed))
		f.integer("h_uncertainty_speed", int64(v.HorizontalUncertaintySpeed))
		f.integer("v_uncertainty_speed", int64(v.VerticalUncertaintySpeed))
	default:
		panic(fmt.Sprintf("unhandled cross-check value %T", value))
	}
	return f
}

func crosscheckInvalidSentinel(name string) error {
	switch name {
	case "invalid_header_spare", "invalid_unc_spare", "invalid_velocity_spare":
		return gad.ErrSpareBit
	case "invalid_reserved_shape":
		return gad.ErrShapeTypeInvalid
	case "invalid_velocity_reserved":
		return gad.ErrVelocityTypeInvalid
	case "invalid_polygon_2", "invalid_orientation_180", "invalid_arc_offset_180", "invalid_arc_included_180", "invalid_bearing_360", "invalid_ha_alt_1280001":
		return gad.ErrOutOfRange
	case "invalid_shape_length_minus", "invalid_shape_length_plus", "invalid_velocity_length_minus", "invalid_velocity_length_plus":
		return gad.ErrLength
	default:
		return nil
	}
}

func TestCrosscheck(t *testing.T) {
	data, err := os.ReadFile("testdata/crosscheck/vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct {
		Generated string                     `json:"generated"`
		Tools     map[string]json.RawMessage `json:"tools"`
		Vectors   []crosscheckVector         `json:"vectors"`
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if err = decoder.Decode(new(any)); err != io.EOF {
		t.Fatalf("trailing fixture data: %v", err)
	}
	if fixture.Generated == "" || len(fixture.Tools) == 0 {
		t.Fatal("missing fixture provenance")
	}
	_, err = time.Parse(time.RFC3339Nano, fixture.Generated)
	if err != nil {
		t.Fatalf("invalid fixture generation timestamp %q: %v", fixture.Generated, err)
	}
	for name, want := range map[string]struct{ version, commit string }{
		"wireshark":   {"4.6.8", "e677bf052328efc1ed897a547fa161836a0e4ff7"},
		"libosmocore": {"1.14.2", "73fed749b1e0ccb8f55226adab0f023561e51ed3"},
	} {
		var tool struct{ Version, Commit string }
		if err := json.Unmarshal(fixture.Tools[name], &tool); err != nil {
			t.Fatalf("%s provenance: %v", name, err)
		}
		// TShark --version reports an abbreviated commit; accept its prefix
		// while keeping the captured decoder version pinned for W1–W8.
		if tool.Version != want.version || len(tool.Commit) < 12 || !strings.HasPrefix(want.commit, tool.Commit) {
			t.Fatalf("%s provenance = %+v, want %+v", name, tool, want)
		}
	}
	var generator struct{ Seed int64 }
	if err := json.Unmarshal(fixture.Tools["generator"], &generator); err != nil {
		t.Fatalf("generator provenance: %v", err)
	}
	if generator.Seed != 230321900 {
		t.Fatalf("generator seed %d, want 230321900", generator.Seed)
	}
	var valid, invalid, rawFields, conversions, skipped, libVectors, libFields, libDecoded, l1 int
	coverage := make(map[string]int)
	names := make(map[string]bool)
	for _, vector := range fixture.Vectors {
		if vector.Name == "" {
			t.Fatal("unnamed cross-check vector")
		}
		if names[vector.Name] {
			t.Fatalf("duplicate vector %q", vector.Name)
		}
		names[vector.Name] = true
		t.Run(vector.Name, func(t *testing.T) {
			b, err := hex.DecodeString(vector.Hex)
			if err != nil {
				t.Fatal(err)
			}
			if len(b) == 0 || vector.Type != int(b[0]>>4) {
				t.Fatalf("type metadata %d does not match header %x", vector.Type, b)
			}
			var value any
			var encoded []byte
			switch vector.Kind {
			case "shape":
				var d gad.Description
				d, err = gad.Decode(b)
				value = d
				if err == nil && vector.Valid {
					encoded, err = d.Encode()
				}
			case "velocity":
				var v gad.Velocity
				v, err = gad.DecodeVelocity(b)
				value = v
				if err == nil && vector.Valid {
					encoded, err = v.Encode()
				}
			default:
				t.Fatalf("unknown kind %q", vector.Kind)
			}
			if !vector.Valid {
				invalid++
				want := crosscheckInvalidSentinel(vector.Name)
				if want == nil {
					t.Fatalf("invalid vector lacks explicit sentinel: %s", vector.Name)
				}
				if !errors.Is(err, want) {
					t.Fatalf("error %v, want %v", err, want)
				}
				raw, converted, ignored, vectors, fields, decoded := crosscheckInvalid(t, vector, b)
				rawFields += raw
				conversions += converted
				skipped += ignored
				libVectors += vectors
				libFields += fields
				libDecoded += decoded
				return
			}
			valid++
			coverage[fmt.Sprintf("%s/%d", vector.Kind, vector.Type)]++
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(encoded, b) {
				t.Fatalf("re-encoded %x, want %x", encoded, b)
			}
			if vector.Wireshark.Malformed || len(vector.Wireshark.Expert) != 0 {
				t.Fatalf("valid vector has Wireshark diagnostics: %+v", vector.Wireshark)
			}
			expected := crosscheckExpected(value)
			raw, converted, ignored := crosscheckCompareWireshark(t, vector.Wireshark.Fields, expected)
			rawFields += raw
			conversions += converted
			skipped += ignored
			if vector.Kind == "velocity" {
				if vector.Libosmocore != nil {
					t.Fatal("velocity unexpectedly has libosmocore data")
				}
				return
			}
			if vector.Type == 13 || vector.Type == 14 {
				if vector.Libosmocore != nil && (vector.Libosmocore.RawReadRC == nil || *vector.Libosmocore.RawReadRC >= 0 || !crosscheckNull(vector.Libosmocore.RawFields) || !crosscheckNull(vector.Libosmocore.Decoded)) {
					t.Fatal("unsupported scalable shape unexpectedly has libosmocore comparison data")
				}
				return
			}
			osmo := vector.Libosmocore
			if osmo == nil || osmo.RawReadRC == nil {
				t.Fatal("missing libosmocore raw read result")
			}
			if vector.Type == 11 {
				// L1: libosmocore 73fed749 gsm_23_032.h:185–207 inserts alt[3];
				// gad.c:312–313,363–365 consequently requires 16 instead of 13 octets.
				if *osmo.RawReadRC != -22 || !strings.Contains(osmo.RawError, "expected 16 bytes, got 13") || !crosscheckNull(osmo.RawFields) || !crosscheckNull(osmo.Decoded) {
					t.Fatalf("expected measured L1 failure, got %+v", osmo)
				}
				l1++
				return
			}
			if *osmo.RawReadRC != 0 {
				t.Fatalf("libosmocore raw read: %d %s", *osmo.RawReadRC, osmo.RawError)
			}
			libVectors++
			libFields += crosscheckCompareRaw(t, osmo.RawFields, crosscheckRawExpected(value, expected))
			if vector.Type == 1 {
				if osmo.DecodeRC == nil || *osmo.DecodeRC != 0 {
					t.Fatalf("circle decode failed: %v %s", osmo.DecodeRC, osmo.DecodeError)
				}
				libDecoded += crosscheckCircle(t, value.(gad.EllipsoidPointUncertaintyCircle), osmo.Decoded)
			} else if osmo.DecodeRC == nil || *osmo.DecodeRC != -95 || !crosscheckNull(osmo.Decoded) {
				// gad.c:399–404 implements physical conversion only for shape 1.
				t.Fatalf("unsupported semantic conversion: %+v", osmo)
			}
		})
	}
	if valid != 676 || invalid != 15 {
		t.Fatalf("vector counts valid=%d invalid=%d, want 676 and 15", valid, invalid)
	}
	for _, typ := range []int{0, 1, 3, 5, 8, 9, 10, 11, 12, 13, 14} {
		if coverage[fmt.Sprintf("shape/%d", typ)] == 0 {
			t.Errorf("missing shape %d", typ)
		}
	}
	for typ := 0; typ < 4; typ++ {
		if coverage[fmt.Sprintf("velocity/%d", typ)] == 0 {
			t.Errorf("missing velocity %d", typ)
		}
	}
	if rawFields != 6208 || conversions == 0 || skipped == 0 || conversions+skipped != 3180 || libVectors != 344 || libFields != 3412 || libDecoded != 102 || l1 != 53 {
		t.Fatalf("incomplete checks: raw=%d conversions=%d skipped=%d lib_vectors=%d lib_fields=%d lib_converted=%d L1=%d", rawFields, conversions, skipped, libVectors, libFields, libDecoded, l1)
	}
	t.Logf("valid=%d invalid=%d types=%d Wireshark_raw=%d conversions_compared=%d annotations_skipped=%d libosmocore_raw_vectors=%d raw_fields=%d converted_fields=%d L1_failures=%d", valid, invalid, len(coverage), rawFields, conversions, skipped, libVectors, libFields, libDecoded, l1)
}

func crosscheckNull(b json.RawMessage) bool { return len(b) == 0 || bytes.Equal(b, []byte("null")) }

func crosscheckRawExpected(value any, fields crosscheckFields) map[string]any {
	coded := make(map[string]int64)
	for _, field := range fields {
		n, _ := strconv.ParseInt(field.show, 10, 64)
		coded[field.name] = n
	}
	raw := map[string]any{"type": coded["location_estimate"]}
	if polygon, ok := value.(gad.Polygon); ok {
		raw["num_points"] = len(polygon.Points)
		points := make([]map[string]int64, len(polygon.Points))
		for i, p := range polygon.Points {
			lat := int64(p.Latitude.N)
			if p.Latitude.South {
				lat |= 1 << 23
			}
			points[i] = map[string]int64{"lat": lat, "lon": int64(uint32(p.Longitude) & 0xffffff)}
		}
		raw["points"] = points
		return raw
	}
	// libosmocore 73fed749 include/osmocom/gsm/protocol/gsm_23_032.h:
	// classic point arrays :62–106, altitude :124–158, arc :160–183,
	// HA altitude :209–236. Arrays are unsigned, network-order packed codes.
	if lat, ok := coded["deg_of_latitude"]; ok {
		raw["lat"] = lat | coded["sign_of_latitude"]<<23
		raw["lon"] = coded["deg_of_longitude"] & 0xffffff
	} else {
		raw["lat"] = int64(uint32(int32(coded["hig_acc_deg_of_lat"])))
		raw["lon"] = int64(uint32(int32(coded["high_acc_deg_of_long"])))
	}
	for ws, osmo := range map[string]string{
		"uncertainty_code": "unc", "uncertainty_semi_major": "unc_semi_major", "uncertainty_semi_minor": "unc_semi_minor",
		"orientation_of_major_axis": "major_ori", "confidence": "confidence", "uncertainty_altitude": "unc_alt",
		"inner_radius": "inner_r", "uncertainty_radius": "unc_r", "offset_angle": "ofs_angle", "included_angle": "incl_angle",
		"high_acc_uncertainty_semi_major": "unc_semi_major", "high_acc_uncertainty_semi_minor": "unc_semi_minor",
		"horizontal_confidence": "h_confidence", "high_acc_uncertainty_alt": "unc_alt", "vertical_confidence": "v_confidence",
	} {
		if code, ok := coded[ws]; ok {
			raw[osmo] = code
		}
	}
	if code, ok := coded["altitude"]; ok {
		raw["alt"] = code | coded["D"]<<15
	}
	if code, ok := coded["high_acc_alt"]; ok {
		raw["alt"] = code & 0x3fffff
	}
	spares := map[int64][]string{1: {"spare2"}, 3: {"spare1", "spare2", "spare3"}, 9: {"spare1", "spare2", "spare3", "spare4"}, 10: {"spare1", "spare2"}, 12: {"spare1", "spare2"}}
	for _, name := range spares[coded["location_estimate"]] {
		raw[name] = 0
	}
	return raw
}

func crosscheckCompareRaw(t *testing.T, actual json.RawMessage, expected map[string]any) int {
	t.Helper()
	encoded, err := json.Marshal(expected)
	if err != nil {
		t.Fatal(err)
	}
	decode := func(b []byte) any {
		var v any
		d := json.NewDecoder(bytes.NewReader(b))
		d.UseNumber()
		if err := d.Decode(&v); err != nil {
			t.Fatal(err)
		}
		return v
	}
	got, want := decode(actual), decode(encoded)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("libosmocore raw_fields = %s, want %s", actual, encoded)
	}
	count := len(expected)
	if points, ok := expected["points"].([]map[string]int64); ok {
		count += 2*len(points) - 1
	}
	return count
}

func crosscheckCircle(t *testing.T, v gad.EllipsoidPointUncertaintyCircle, actual json.RawMessage) int {
	t.Helper()
	// libosmocore 73fed749 src/gsm/gad.c:81–93 shifts the positive
	// latitude magnitude before applying its sign; :128–146 truncates signed
	// longitude division toward zero. Both formulas are exact integer checks.
	lat := int64(v.Latitude.N) * 90000000 >> 23
	if v.Latitude.South {
		lat = -lat
	}
	lon := int64(v.Longitude) * 360000000 / (1 << 24)
	if int64(math.Trunc(v.Latitude.Degrees()*1e6)) != lat || int64(math.Trunc(v.Longitude.Degrees()*1e6)) != lon {
		t.Fatal("coordinate conversion differs from integer microdegree formulas")
	}
	// gad.c:149–169 generates its millimetre table by truncating
	// 10000*((11/10)^K-1). All 128 source entries equal this exact rational
	// floor; compute it without copying the upstream table or inheriting
	// floating pow rounding. :178–180 selects that table by the raw code.
	k := big.NewInt(int64(v.Uncertainty))
	a := new(big.Int).Exp(big.NewInt(11), k, nil)
	b := new(big.Int).Exp(big.NewInt(10), k, nil)
	a.Sub(a, b)
	a.Mul(a, big.NewInt(10000))
	a.Quo(a, b)
	unc := a.Int64()
	// A truncated millimetre value loses less than one millimetre.
	// Compare our physical conversion to that same bin, not a wider tolerance.
	mm := v.Uncertainty.Metres() * 1000
	if mm < float64(unc) || mm >= float64(unc+1) {
		t.Fatalf("uncertainty %.17g mm outside [%d,%d)", mm, unc, unc+1)
	}
	crosscheckCompareRaw(t, actual, map[string]any{"type": 1, "lat_microdegrees": lat, "lon_microdegrees": lon, "unc_mm": unc})
	return 3
}

func crosscheckCompareWireshark(t *testing.T, actual [][]string, expected crosscheckFields) (raw, conversions, skipped int) {
	t.Helper()
	if len(expected) != len(actual) {
		t.Fatalf("Wireshark emitted %d fields, expected all %d: %v", len(actual), len(expected), actual)
	}
	for i, want := range expected {
		got := actual[i]
		if len(got) < 2 || len(got) > 3 {
			t.Fatalf("field %d has malformed tuple %v", i, got)
		}
		if got[0] != want.name || got[1] != want.show {
			t.Fatalf("field %d = %v, want [%s %s]", i, got, want.name, want.show)
		}
		raw++
		if want.skip != "" {
			if len(got) != 3 {
				t.Fatalf("missing known %s annotation for %s", want.skip, want.name)
			}
			skipped++
		} else if want.conversion != "" {
			if len(got) != 3 || got[2] != want.conversion {
				t.Fatalf("%s conversion %v, want exactly %q", want.name, got, want.conversion)
			}
			conversions++
		} else if len(got) != 2 {
			t.Fatalf("unhandled annotation %v", got)
		}
	}
	return
}

// Rejected octets have no decoded Go value. Construct unchecked coded-field
// values from the contract layouts for these named cases, preserving the bad
// ranges and keeping spare bits separate. Never feed normalized data to Decode.
// Explicit partial lists below model what the captured length-error dissection
// actually reached; they cannot hide a missing field in any valid vector.
func crosscheckInvalid(t *testing.T, vector crosscheckVector, b []byte) (raw, conversions, skipped, libVectors, libFields, libDecoded int) {
	t.Helper()
	var value any
	var expected crosscheckFields
	point := func(offset int) gad.Point {
		n := uint32(b[offset])<<16 | uint32(b[offset+1])<<8 | uint32(b[offset+2])
		lon := int32(b[offset+3])<<24 | int32(b[offset+4])<<16 | int32(b[offset+5])<<8
		return gad.Point{Latitude: gad.Latitude{South: n&0x800000 != 0, N: n & 0x7fffff}, Longitude: gad.Longitude(lon >> 8)}
	}
	switch vector.Name {
	case "invalid_header_spare", "invalid_unc_spare", "invalid_shape_length_plus":
		value = gad.EllipsoidPointUncertaintyCircle{Point: point(1), Uncertainty: gad.Uncertainty(b[7] & 127)}
	case "invalid_shape_length_minus":
		expected.integer("location_estimate", int64(b[0]>>4))
		expected.point(point(1))
	case "invalid_reserved_shape":
		expected.integer("location_estimate", int64(b[0]>>4))
	case "invalid_polygon_2":
		points := make([]gad.Point, int(b[0]&15))
		for i := range points {
			points[i] = point(1 + 6*i)
		}
		value = gad.Polygon{Points: points}
	case "invalid_orientation_180":
		value = gad.EllipsoidPointUncertaintyEllipse{Point: point(1), SemiMajor: gad.Uncertainty(b[7] & 127), SemiMinor: gad.Uncertainty(b[8] & 127), Orientation: gad.Orientation(b[9]), Confidence: gad.Confidence(b[10] & 127)}
	case "invalid_arc_offset_180", "invalid_arc_included_180":
		value = gad.EllipsoidArc{Point: point(1), InnerRadius: gad.InnerRadius(b[7])<<8 | gad.InnerRadius(b[8]), UncertaintyRadius: gad.Uncertainty(b[9] & 127), OffsetAngle: gad.OffsetAngle(b[10]), IncludedAngle: gad.IncludedAngle(b[11]), Confidence: gad.Confidence(b[12] & 127)}
	case "invalid_ha_alt_1280001":
		high := func(offset int) int32 {
			return int32(b[offset])<<24 | int32(b[offset+1])<<16 | int32(b[offset+2])<<8 | int32(b[offset+3])
		}
		alt := int32(b[9]&63)<<26 | int32(b[10])<<18 | int32(b[11])<<10
		value = gad.HighAccuracyEllipsoidPointAltitudeUncertaintyEllipsoid{HighAccuracyPoint: gad.HighAccuracyPoint{Latitude: gad.HighAccuracyLatitude(high(1)), Longitude: gad.HighAccuracyLongitude(high(5))}, Altitude: gad.HighAccuracyAltitude(alt >> 10), SemiMajor: gad.HighAccuracyUncertainty(b[12]), SemiMinor: gad.HighAccuracyUncertainty(b[13]), Orientation: gad.Orientation(b[14]), HorizontalConfidence: gad.Confidence(b[15] & 127), UncertaintyAltitude: gad.HighAccuracyUncertainty(b[16]), VerticalConfidence: gad.Confidence(b[17] & 127)}
	case "invalid_velocity_reserved":
		expected.integer("velocity_type", int64(b[0]>>4))
	case "invalid_bearing_360", "invalid_velocity_spare":
		value = gad.HorizontalVelocity{Bearing: gad.Bearing(b[0]&1)<<8 | gad.Bearing(b[1]), HorizontalSpeed: gad.HorizontalSpeed(b[2])<<8 | gad.HorizontalSpeed(b[3])}
	case "invalid_velocity_length_minus", "invalid_velocity_length_plus":
		expected.integer("velocity_type", int64(b[0]>>4))
		expected.integer("spare_bits", int64(b[0]&12)>>2)
		expected = append(expected, crosscheckField{name: "d", show: map[bool]string{false: "False", true: "True"}[b[0]&2 != 0]})
		expected.integer("bearing", int64(b[0]&1)<<8|int64(b[1]))
		expected.integer("horizontal_velocity", int64(b[2])<<8|int64(b[3]))
		expected.integer("vertical_speed", int64(b[4]))
		expected.integer("h_uncertainty_speed", int64(b[5]))
		if vector.Name == "invalid_velocity_length_plus" {
			expected.integer("v_uncertainty_speed", int64(b[6]))
		}
	default:
		t.Fatalf("no unchecked layout for %s", vector.Name)
	}
	if value != nil {
		expected = crosscheckExpected(value)
	}
	if vector.Name == "invalid_velocity_spare" {
		expected[1].show = strconv.Itoa(int(b[0]&14) >> 1)
	}
	raw, conversions, skipped = crosscheckCompareWireshark(t, vector.Wireshark.Fields, expected)
	wantMalformed := vector.Name == "invalid_velocity_length_minus"
	if vector.Wireshark.Malformed != wantMalformed || (len(vector.Wireshark.Expert) > 0) != wantMalformed {
		t.Fatalf("unexpected captured malformed/expert state: %+v", vector.Wireshark)
	}
	if vector.Kind == "velocity" {
		if vector.Libosmocore != nil {
			t.Fatal("invalid velocity unexpectedly has libosmocore data")
		}
		return
	}
	osmo := vector.Libosmocore
	if osmo == nil || osmo.RawReadRC == nil {
		t.Fatal("invalid shape missing libosmocore result")
	}
	switch vector.Name {
	case "invalid_reserved_shape", "invalid_polygon_2", "invalid_shape_length_minus", "invalid_shape_length_plus":
		if *osmo.RawReadRC != -22 || osmo.RawError == "" || !crosscheckNull(osmo.RawFields) || !crosscheckNull(osmo.Decoded) {
			t.Fatalf("expected libosmocore structural failure, got %+v", osmo)
		}
		return
	}
	if *osmo.RawReadRC != 0 {
		t.Fatalf("expected captured permissive raw read, got %+v", osmo)
	}
	libVectors = 1
	packed := crosscheckRawExpected(value, expected)
	if vector.Name == "invalid_unc_spare" {
		packed["spare2"] = int(b[7] >> 7)
	}
	libFields = crosscheckCompareRaw(t, osmo.RawFields, packed)
	switch vector.Name {
	case "invalid_header_spare":
		if osmo.DecodeRC == nil || *osmo.DecodeRC != 0 {
			t.Fatalf("expected captured permissive header decode, got %+v", osmo)
		}
		libDecoded = crosscheckCircle(t, value.(gad.EllipsoidPointUncertaintyCircle), osmo.Decoded)
	case "invalid_unc_spare":
		if osmo.DecodeRC == nil || *osmo.DecodeRC != -22 || !strings.Contains(osmo.DecodeError, "Bit 8 of Uncertainty code should be zero") || !crosscheckNull(osmo.Decoded) {
			t.Fatalf("expected captured uncertainty spare rejection, got %+v", osmo)
		}
	default:
		if osmo.DecodeRC == nil || *osmo.DecodeRC != -95 || !crosscheckNull(osmo.Decoded) {
			t.Fatalf("expected unsupported semantic conversion, got %+v", osmo)
		}
	}
	return
}
