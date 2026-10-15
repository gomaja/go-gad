# Standards and scope

Versions checked on **2026-10-15**. The latest published version of each
document is the one listed.

| Document | Version | Clauses used | Sources |
| --- | --- | --- | --- |
| 3GPP TS 23.032, *Universal Geographical Area Description (GAD)* | V19.0.0 (2025-09) | §6 coding, §7.2 Table 2a, §7.3 figures, §8 velocity, Annex A | [3GPP archive](https://www.3gpp.org/ftp/Specs/archive/23_series/23.032/) (`23032-j00.zip`, 2025-10-06), [ETSI TS 123 032 V19.0.0](https://www.etsi.org/deliver/etsi_ts/123000_123099/123032/19.00.00_60/ts_123032v190000p.pdf) |

How the version was checked: the 3GPP archive has no newer file than
`23032-j00.zip` and no Release 20 version; the newest ETSI publication is
19.00.00; the 3GPP change request database lists CR 0030 (SP-241465, approved
at SA#106, 2024-12, first in V18.3.0) as the last change request to TS 23.032,
with nothing agreed or pending after it. The figures and equations were read
from the ETSI PDF, which renders them; in the archive's Word file they are
embedded images.

## Scope

TS 23.032 codes eleven shapes in octets (§7.3) and four velocity types (§8,
Table 3); this module decodes, encodes and validates all of them. The shapes of
§5.8 to §5.12 (local and relative locations, range and direction) and the
relative velocity of §8.4a have no octet coding there and are out of scope, as
are the polygon geometry rules of §5.4 (no crossing lines, no diametrically
opposed successive points).

The field layouts of the high-accuracy shapes (§7.3.3a, §7.3.3b, §7.3.6a and
§7.3.6b) exist only as figures; Annex A gives the compact notation of the other
shapes and of the velocity types.

## Where the specification is inconsistent or silent

- Figure 17 (§8.15) draws the velocity type of horizontal with vertical velocity
  and uncertainty as `0010`; Table 3 and Annex A give `0011`. The code follows
  Table 3.
- §6.1a maps a latitude of +90° to N = 2^31, which a 32-bit two's complement
  field cannot hold. The code uses 2^31−1, extending the top bin to include 90°
  as §6.1 does for the 24-bit latitude.
- §6.5 calls confidence codes 100 < K ≤ 128 unusable; the field has 7 bits, so
  the range is 101 to 127. Decoding accepts and preserves them; `Percent`
  reports them, like 0, as no information.
- §8.8 says bearing codes 360 ≤ N < 511 are not used; 511 is as unusable, since
  N ≤ b < N+1 puts it outside 0° to 360°. Bearings 360 to 511 are rejected.
- The functions of §6.2, §6.2a, §6.2b, §6.4 and §8.11 give the value of each
  code but no rule for choosing a code. The constructors choose the smallest code
  whose value is not below the given uncertainty, so an encoded uncertainty is
  never smaller than the real one.
- §6.3a defines a = N·2^−7 without a bin rule. The constructor uses ⌊128·a⌋,
  the floor rule of §6.1a.
- Annex A writes every spare bit as `<spare>` while the figures mark some as
  "0 spare" and the type-of-shape octet as "spare". Decoding requires all of
  them to be zero (issue #1).
- The example tables are approximate: Table 1 prints 3034.8 m as "3 km",
  20474.0 m as "20 km" and 1806.6 km as "1800 km"; Table 6.2a-1 truncates to five
  decimals and Table 6.2b-1 rounds. The tests compare each row at the precision
  it is printed.

## Independent decoders

The classic and high-accuracy shapes and the velocity types are cross-checked
against Wireshark's GAD dissector and libosmocore's GAD code; versions, the
regeneration procedure and the defects found in those decoders are in
[testdata/crosscheck/README.md](testdata/crosscheck/README.md).

Recheck the archive for newer versions before a significant revisit of this
package, and update this file with the date of the check.
