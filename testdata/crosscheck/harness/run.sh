#!/usr/bin/env bash
set -euo pipefail
harness=$(cd "$(dirname "$0")" && pwd)
output=${1:-"$harness/../vectors.json"}
TSHARK=${TSHARK:-tshark}
command -v "$TSHARK" >/dev/null
work=$(mktemp -d "${TMPDIR:-/tmp}/gad-crosscheck.XXXXXXXX")
container="$(basename "$work")"
cleanup() {
  docker rm -f "$container" >/dev/null 2>&1 || true
  rm -rf "$work"
}
trap cleanup EXIT
python3 "$harness/generate.py" "$work/input.json"
python3 "$harness/pcap.py" "$work/input.json" "$work/vectors.pcap"
python3 - "$work" <<'PY'
import json,pathlib,sys
work=pathlib.Path(sys.argv[1])
vectors=json.loads((work/'input.json').read_text())
(work/'osmo-input.txt').write_text(''.join(v['hex']+'\n' for v in vectors if v['kind']=='shape'))
PY
timeout 2700 docker build -f "$harness/Dockerfile" -t gad-crosscheck-osmo:1.14.2 "$harness"
docker run --rm --name "$container" -i gad-crosscheck-osmo:1.14.2 < "$work/osmo-input.txt" > "$work/libosmocore.jsonl"
"$TSHARK" --version > "$work/tshark-version.txt"
"$TSHARK" -n -r "$work/vectors.pcap" -T pdml > "$work/wireshark.pdml"
python3 "$harness/extract.py" "$work/input.json" "$work/wireshark.pdml" "$work/libosmocore.jsonl" "$work/tshark-version.txt" "$work/full.json"
python3 "$harness/compact.py" "$work/full.json" "$output"
printf 'Wrote %s (%s bytes)\n' "$output" "$(wc -c < "$output")"
