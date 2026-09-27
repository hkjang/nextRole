#!/usr/bin/env python3
"""Collect installed Go / production npm license notices without substituting summaries.

Run after `go mod download all` and `npm ci --prefix web`.
The service image retains Debian package notices under /usr/share/doc.
"""
from __future__ import annotations
import json
import re
import shutil
import subprocess
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
LICENSE_NAME = re.compile(r'^(?:licen[sc]e|copying|ofl|notice|copyright)(?:[._-].*|$)', re.I)


def command(args: list[str], cwd: Path = ROOT) -> str:
    result = subprocess.run(args, cwd=cwd, capture_output=True, text=True, check=True)
    return result.stdout


def json_stream(raw: str):
    decoder = json.JSONDecoder()
    while raw.strip():
        value, end = decoder.raw_decode(raw.lstrip())
        yield value
        raw = raw.lstrip()[end:]


def notices(directory: Path) -> list[Path]:
    return sorted(p for p in directory.iterdir() if p.is_file() and LICENSE_NAME.match(p.name))


def section(name: str, version: str, origin: str, files: list[Path]) -> str:
    if not files:
        raise RuntimeError(f'No license notice found for {name} {version}: {origin}')
    parts = ['=' * 79, f'{name} {version}'.strip(), f'Origin: {origin}', '=' * 79]
    for file in files:
        notice = '\n'.join(line.rstrip() for line in file.read_text(errors='replace').splitlines()).rstrip()
        parts.extend([f'--- {file.name} ---', notice, ''])
    return '\n'.join(parts)


def main():
    parts = ['''NEXTROLE THIRD-PARTY NOTICES

This file preserves license/copyright notices distributed with the dependencies
used to build NextRole. Inclusion does not imply endorsement. Package-level
license terms remain authoritative; no new license is applied to third-party
code. The list can include build/test or type-only modules from the pinned Go
module graph and the production npm dependency graph.

Generated from go.mod/go.sum and web/package-lock.json installed dependency
sources by scripts/generate-notices.py. Regenerate when dependencies change.

The service Docker image also contains Debian system software including curl,
poppler-utils (pdftotext), CA certificates and their shared libraries. Their
package-specific copyright/license notices remain inside the image under
/usr/share/doc/<package>/copyright and applicable shared license texts under
/usr/share/common-licenses. OS package versions can be inspected with
`dpkg-query -W` inside the image. These operating-system package notices are
not replaced by this application-dependency notice file.
''']
    goroot = Path(command(['go', 'env', 'GOROOT']).strip())
    version = command(['go', 'version']).strip()
    go_license = goroot / 'LICENSE'
    if not go_license.is_file():
        candidates = sorted(Path('/usr/share/doc').glob('golang-*-src/copyright'))
        if not candidates:
            raise RuntimeError('Go standard library LICENSE not found in GOROOT or distro source package')
        go_license = next((p for p in candidates if goroot.name.removeprefix('go-') in str(p)), candidates[-1])
    parts.append(section('Go toolchain and standard library', version, 'https://go.dev/', [go_license]))
    modules = list(json_stream(command(['go', 'list', '-m', '-json', 'all'])))
    go_count = 0
    for mod in sorted(modules, key=lambda m: m['Path']):
        if mod.get('Main'):
            continue
        actual = mod.get('Replace', mod)
        if not actual.get('Dir'):
            downloaded = json.loads(command(['go', 'mod', 'download', '-json', actual['Path'] + '@' + actual['Version']]))
            actual['Dir'] = downloaded['Dir']
        directory = Path(actual['Dir'])
        parts.append(section(mod['Path'], mod.get('Version', ''), 'https://' + mod['Path'], notices(directory)))
        go_count += 1
    web = ROOT / 'web'
    npm_count = 0
    dirs = {Path(p) for p in command(['npm', 'ls', '--omit=dev', '--all', '--parseable'], web).splitlines()}
    packages = []
    for directory in dirs:
        if directory == web:
            continue
        pkg = json.loads((directory / 'package.json').read_text())
        packages.append((pkg['name'], pkg.get('version', ''), directory, pkg.get('license', '')))
    for name, version, directory, license_id in sorted(packages):
        files = notices(directory)
        if not files:
            supplement = {
                'react-remove-scroll-bar': 'react-remove-scroll-bar.LICENSE',
                'use-composed-ref': 'use-composed-ref.NOTICE.txt',
            }.get(name)
            if supplement:
                files = [ROOT / 'docs/licenses' / supplement]
        parts.append(section(name, version, f'https://www.npmjs.com/package/{name} (declared license: {license_id})', files))
        npm_count += 1
    output = ROOT / 'THIRD_PARTY_NOTICES.txt'
    output.write_text('\n\n'.join(parts).rstrip() + '\n')
    font_license = web / 'node_modules/@fontsource/noto-sans-kr/LICENSE'
    destination = web / 'public/licenses/noto-sans-kr.txt'
    destination.parent.mkdir(parents=True, exist_ok=True)
    shutil.copyfile(font_license, destination)
    print(f'Wrote {output.name}: Go standard library + {go_count} Go modules + {npm_count} npm packages; copied Noto Sans KR OFL.')


if __name__ == '__main__':
    main()
