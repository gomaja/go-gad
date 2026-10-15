# Independent decoder cross-check

`vectors.json` preserves observations captured on 2026-10-15 from Wireshark
4.6.8 (`e677bf052328efc1ed897a547fa161836a0e4ff7`) and libosmocore 1.14.2
(`73fed749b1e0ccb8f55226adab0f023561e51ed3`). Its 691 synthetic vectors contain
676 structurally valid descriptions/velocities and 15 invalid controls. The
stdlib Python generator uses seed `230321900`; each supported shape and velocity
type has 20 random cases plus explicit field boundaries. Duplicate byte strings
with different boundary names are intentional. Polygon geometry is outside the
TS 23.032 V19.0.0 codec contract.

The original source comparison recorded identical complete GAD dissector files
in Wireshark 4.6.8 and 4.6.9 (`2d548b197c7582332f143c15259e1650aff7cc26`), and
identical geographical/velocity decoder functions at master
`b97ab61b7fdcdb15c3c8b184fac6d0526cf4cb4e`. These are historical observations,
not a claim about subsequent releases. The original libosmocore tag object was
`d4b18ea65759e1e9e5f2eded53cabafdf4c8decc`.

Source provenance from that capture:

- [Wireshark 4.6.8 GAD dissector](https://gitlab.com/wireshark/wireshark/-/raw/v4.6.8/epan/dissectors/packet-gsm_a_common.c).
  The auxiliary [4.6.8 source archive](https://api.github.com/repos/wireshark/wireshark/tarball/v4.6.8)
  had SHA-256 `e4101f9ea7d24cfb59ef7fdb71f22f43cdd0a073f92b4b6d28a1e22b31439eab`.
- [libosmocore commit archive](https://api.github.com/repos/osmocom/libosmocore/tarball/73fed749b1e0ccb8f55226adab0f023561e51ed3),
  SHA-256 `901ab0d956548da130424913fa7de37debe2ea03a20a45cc72973735c1fe61e3`.
  The Dockerfile downloads and verifies this archive; no upstream sources or
  archives are committed here. `.tarball-version` supplies the release metadata
  missing from a commit archive; decoder sources are unchanged.
- The build base is `golang:1.27.2-bookworm`, digest
  `sha256:5cf287a799e6b94384bad13d16b14904c531f51ba65792237e122ce42b392f61`.
  Debian build packages are not snapshot pinned, so their versions can change.

Regenerate with Docker, Python 3.9 or newer, GNU `timeout`, and Wireshark 4.6.8
`tshark` to reproduce this snapshot. The Go test pins decoder versions because
upgrading them requires rechecking the known conversion defects below:

```sh
TSHARK=/path/to/tshark testdata/crosscheck/harness/run.sh
# An optional first argument selects a different output fixture.
```

`TSHARK` defaults to `tshark`. The actual executable's version and reported
commit are recorded on each run. A new run records its runtime versions and the
libosmocore archive hash; historical comparisons against other Wireshark commits,
the auxiliary source archive, and the original image ID are not carried forward.
The capture is generated locally: shapes use
Diameter TCP/3868 vendor 10415 AVP 1242; velocities use CoAP PUT `/6/0`, content
format 11542, LwM2M Location resource 4. No packets are transmitted. Working
files live in a temporary directory and are removed on exit; only the selected
fixture destination is written. Docker retains its build image/cache.

`harness/compact.py INPUT OUTPUT` also derives the compact fixture directly from
a full capture JSON. It preserves field order and repeated polygon fields.
Each Wireshark field is `[abbreviation, show]` or
`[abbreviation, show, conversion]`, with `gsm_a.gad.`/`gsm_a.` removed. Only a
trailing parenthesized annotation ending in ` m` or ` degrees` becomes a
conversion; its parentheses are removed. Location URIs and other labels are
dropped. Expert records are preserved unchanged. libosmocore retains separate
raw-read and semantic-decode results/errors; unavailable members are null, and
velocity entries are null because that release has no velocity API.

The Go cross-check verifies valid byte round trips, strict rejection of invalid
controls, every Wireshark coded field, and the physical-unit annotations whose
conversions follow the specification. It compares libosmocore raw fields for
supported shapes and shape-1 conversions using that library's integer arithmetic.
It explicitly expects the shape-11 failure below. libosmocore converts only
shape 1; unsupported semantic conversions supply no independent confirmation.
Shapes 13/14 have no raw support. The independent tools are observations, not
validation authorities: most invalid controls produce Wireshark fields without
an expert warning.

| Finding | Observed upstream defect | Conversion excluded from comparison |
|---|---|---|
| W1 | Southern single-point latitude loses its sign (§6.1). | Classic latitude annotation; raw sign remains checked. |
| W2 | Classic latitude divides by 2^23−1 rather than 2^23 (§6.1). | Classic latitude annotation, including polygon vertices. |
| W3 | Classic longitude divides by 2^24−1 rather than 2^24 (§6.1). | Classic longitude annotation. |
| W4 | Major-axis orientation is doubled (§7.3.3). | Orientation annotation. |
| W5 | HA latitude uses sign/magnitude instead of two's complement (§6.1a). | HA latitude annotation. |
| W6 | HA longitude has two minus signs for negative codes (§6.1a). | HA longitude annotation. |
| W7 | Scalable horizontal uncertainty ignores U/HU (§6.2b). | HA horizontal uncertainty when extended range is selected. |
| W8 | HA vertical uncertainty uses the classic altitude formula and ignores VU (§7.3.6a NOTE, §7.3.6b NOTE). | HA uncertainty-altitude annotation. |
| L1 | Shape 11's raw structure inserts three altitude bytes and expects 16 bytes instead of 13 (§7.3.3a). | libosmocore raw-read failure is expected for shape 11. |

Wireshark arc radius/angles are raw codes without physical-unit annotations;
confidence is also retained as a raw code. Their missing interpretation is a
display limitation. None of these limitations relaxes the Go codec's validation.
