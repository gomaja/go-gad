#!/usr/bin/env python3
"""Keep decoder observations without packet offsets or presentation labels."""
import json
import pathlib
import re
import sys


def compact(source):
    tools = source['tools']
    selected = {}
    for name in ('wireshark', 'libosmocore'):
        selected[name] = {
            key: value for key, value in tools[name].items()
            if key in ('version', 'commit', 'latest_stable_verified',
                       'latest_stable_commit', 'master_commit', 'tag_object', 'image')
        }
    selected['generator'] = {'seed': tools['generator']['seed']}
    selected['source_sha256'] = tools.get('source_sha256', {})
    vectors = []
    for original in source['vectors']:
        vector = {key: original[key] for key in ('name', 'kind', 'type', 'valid', 'hex')}
        observed = original['wireshark']
        fields = []
        for field in observed['fields']:
            abbrev = field['name'].removeprefix('gsm_a.gad.').removeprefix('gsm_a.')
            if abbrev == 'location_uri':
                continue
            row = [abbrev, field.get('show', '')]
            annotation = re.search(r'\(([^()]* (?:m|degrees))\)$', field.get('showname', ''))
            if annotation:
                row.append(annotation.group(1))
            fields.append(row)
        vector['wireshark'] = {'malformed': observed['malformed'],
                               'expert': observed['expert'], 'fields': fields}
        osmo = original['libosmocore']
        vector['libosmocore'] = None if osmo is None else {
            key: osmo.get(key) for key in ('raw_read_rc', 'raw_fields', 'decode_rc',
                                         'decoded', 'raw_error', 'decode_error')
        }
        vectors.append(vector)
    return {'generated': source['generated'], 'tools': selected, 'vectors': vectors}


def write(document, destination):
    encode = lambda value: json.dumps(value, separators=(',', ':'), ensure_ascii=False)
    with pathlib.Path(destination).open('w', encoding='utf-8', newline='\n') as output:
        output.write('{"generated":' + encode(document['generated']))
        output.write(',"tools":' + encode(document['tools']) + ',"vectors":[\n')
        output.write(',\n'.join(encode(vector) for vector in document['vectors']))
        output.write('\n]}\n')


if __name__ == '__main__':
    write(compact(json.loads(pathlib.Path(sys.argv[1]).read_text())), sys.argv[2])
