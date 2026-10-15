#!/usr/bin/env python3
"""Join repeated PDML fields and libosmocore results by generated packet order."""
import datetime
import json
import pathlib
import re
import sys
import xml.etree.ElementTree as xml

vectors = json.loads(pathlib.Path(sys.argv[1]).read_text())
packets = xml.parse(sys.argv[2]).getroot().findall('packet')
osmo = iter(map(json.loads, pathlib.Path(sys.argv[3]).read_text().splitlines()))
assert len(vectors) == len(packets), 'packet/vector count differs'
for vector, packet in zip(vectors, packets):
    fields = [dict(field.attrib) for field in packet.iter('field')
              if field.get('name', '').startswith(('gsm_a.gad.', 'gsm_a.spare'))]
    vector['wireshark'] = {
        'fields': fields,
        'expert': [dict(field.attrib) for field in packet.iter('field')
                   if field.get('name', '').startswith('_ws.expert.')],
        'malformed': any(proto.get('name') == '_ws.malformed' for proto in packet.iter('proto')),
    }
    vector['libosmocore'] = next(osmo) if vector['kind'] == 'shape' else None
    assert any(field['name'] in ('gsm_a.gad.location_estimate', 'gsm_a.gad.velocity_type')
               for field in fields), vector['name']
assert next(osmo, None) is None, 'unused libosmocore results'
version = pathlib.Path(sys.argv[4]).read_text().splitlines()[0]
match = re.search(r'TShark \(Wireshark\) (\S+)', version)
assert match, version
commit = re.search(r'Git commit ([^ )]+)', version)
tools = {
    'wireshark': {'version': match.group(1), 'commit': commit.group(1) if commit else None},
    'libosmocore': {'version': '1.14.2', 'commit': '73fed749b1e0ccb8f55226adab0f023561e51ed3'},
    'generator': {'seed': 230321900},
    'source_sha256': {
        'sources/libosmocore-1.14.2.tar.gz':
            '901ab0d956548da130424913fa7de37debe2ea03a20a45cc72973735c1fe61e3',
    },
}
result = {'generated': datetime.datetime.now(datetime.timezone.utc).isoformat(),
          'tools': tools, 'vectors': vectors}
pathlib.Path(sys.argv[5]).write_text(json.dumps(result, separators=(',', ':')) + '\n')
