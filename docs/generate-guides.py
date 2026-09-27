#!/usr/bin/env python3
"""Render maintained Markdown guides to portable HTML (python-markdown required)."""
from pathlib import Path
from html import escape, unescape
import math
import re
import struct
import markdown
from markdown.extensions.toc import slugify_unicode

ROOT = Path(__file__).resolve().parent
CSS = """
:root{font-family:'Noto Sans KR','Apple SD Gothic Neo','Malgun Gothic',system-ui,sans-serif;color:#203e36;background:#f5f7f2;font-size:17px;line-height:1.85;font-synthesis:none}*{box-sizing:border-box}body{margin:0;word-break:keep-all;overflow-wrap:anywhere}header{max-width:1170px;margin:auto;padding:30px 40px;display:flex;align-items:center;justify-content:space-between;gap:20px}header>a{display:flex;align-items:center;gap:11px;font-size:25px;font-weight:800;text-decoration:none;color:#136e63}header img{width:38px;height:38px}nav{display:flex;gap:22px;font-size:14px}nav a{color:#476c58}main{max-width:1090px;margin:0 auto 60px;padding:55px 65px;background:white;border:1px solid #e0e8dd;border-radius:20px;box-shadow:0 12px 40px #23473106}h1,h2,h3{line-height:1.4;letter-spacing:-.045em;color:#173e36;scroll-margin-top:25px}h1{font-size:39px;margin-top:0}h2{font-size:29px;border-top:1px solid #dce6da;padding-top:38px;margin-top:55px}h3{font-size:22px;margin-top:35px}p{margin:16px 0}a{color:#136e63;text-underline-offset:4px}a:hover{color:#093e36}li{margin:8px 0}li>p{margin:0}table{display:block;width:100%;border-collapse:collapse;overflow-x:auto;margin:25px 0;font-size:15px;line-height:1.7}thead{background:#edf3e9}th,td{text-align:left;vertical-align:top;padding:12px 15px;border:1px solid #dae4d8}th{font-weight:700;white-space:nowrap}tr:nth-child(even){background:#fafcf8}code{font-family:'SFMono-Regular',Consolas,monospace;font-size:.86em;color:#14584d;background:#eef4eb;padding:2px 5px;border-radius:4px;word-break:break-word}pre{background:#163b33;color:#eff7e8;padding:24px;overflow:auto;border-radius:12px;font-size:14px;line-height:1.75;white-space:pre-wrap;word-break:break-word}pre code{padding:0;background:transparent;color:inherit;font-size:inherit}img{max-width:100%;height:auto;border:1px solid #dce6da;border-radius:12px;box-shadow:0 9px 28px #173e3608}header img{border:0;box-shadow:none}.toc{padding:24px 30px;background:#f3f7ef;border:1px solid #e0e8d9;border-radius:13px;margin:35px 0}.toctitle{font-size:20px;font-weight:700}.toc ul{list-style:none;padding-left:0;margin:10px 0}.toc li{margin:7px 0;font-size:15px}.toc li ul{padding-left:22px}.toc a{text-decoration:none}blockquote{margin:24px 0;border-left:4px solid #91ba80;padding:2px 22px;background:#f5f8f1;color:#526b56}footer{text-align:center;font-size:13px;color:#68816a;padding:0 25px 35px}::selection{background:#d4ecb5}:focus-visible{outline:3px solid #619a82;outline-offset:4px}@media(max-width:800px){header{padding:24px;align-items:flex-start}nav{flex-wrap:wrap;justify-content:flex-end;gap:8px 15px;font-size:12px}main{margin:0 12px 35px;padding:30px 23px;border-radius:15px}h1{font-size:31px}h2{font-size:25px}h3{font-size:21px}table{font-size:14px}th,td{padding:10px}pre{padding:18px;font-size:12px}.toc{padding:20px}header>a{font-size:22px}}@page{size:A4;margin:17mm 16mm 19mm}@media print{:root{background:white;font-size:10.5pt;line-height:1.65}body{background:white}header{padding:0 0 9mm;max-width:none;border-bottom:1px solid #dce6da}header>a{font-size:20pt}header img{width:30px;height:30px}nav{display:none}main{max-width:none;margin:0;padding:10mm 0 0;border:0;box-shadow:none;border-radius:0}h1{font-size:25pt}h2{font-size:19pt;margin-top:10mm;padding-top:7mm;break-before:page}h3{font-size:14pt;margin-top:7mm}h1,h2,h3{break-after:avoid}p,li{orphans:3;widows:3}table{display:table;font-size:9pt;overflow:visible;table-layout:auto;word-break:break-word}thead{display:table-header-group}tr{break-inside:avoid}th,td{padding:7px 9px}th{white-space:normal}pre{font-size:8pt;border:1px solid #cddccb;color:#1b3c32;background:#f1f6ed;box-shadow:none;break-inside:avoid}code{font-size:.84em}img{max-height:210mm;object-fit:contain;display:block;margin:auto;break-inside:avoid;box-shadow:none}.toc{break-after:page;font-size:10pt}.toc li{font-size:10pt}.toc ul ul{display:none}footer{padding-top:8mm;font-size:8pt}a{color:#136e63}*{-webkit-print-color-adjust:exact;print-color-adjust:exact}}
"""

CAPTURE_CSS = """
.doc-capture{margin:25px 0}.capture-caption{font-size:12px;text-align:center;margin:8px 0 0}.print-capture{display:none}@media print{.doc-capture{margin:5mm 0;break-inside:auto}.screen-capture,.capture-caption{display:none}.print-capture{display:block}.capture-part{break-inside:avoid;margin:0 0 7mm}.capture-part+.capture-part{break-before:page}.capture-part-caption{font-size:8pt;color:#526d5c;line-height:1.5;margin:0 0 3mm}.capture-slice{position:relative;width:100%;overflow:hidden;border:1px solid #d7e3d4;border-radius:0;break-inside:avoid}.capture-slice img{position:absolute;left:0;width:100%;height:auto;max-width:none;max-height:none;margin:0;border:0;border-radius:0;box-shadow:none;display:block}}
"""

def print_captures(body):
    """Preserve original screenshots; lay tall images across readable PDF pages.

    Only HTML presentation is changed. No source image is cropped or rewritten.
    PNG dimensions are read from the standard IHDR header with no image editor.
    """
    def wrap(match):
        image, src = match.group(1), match.group(2)
        path = ROOT / src
        if not path.is_file():
            return match.group(0)
        raw = path.read_bytes()[:24]
        if raw[:8] != b'\x89PNG\r\n\x1a\n':
            return match.group(0)
        width, height = struct.unpack('>II', raw[16:24])
        alt_match = re.search(r'alt="([^"]*)"', image)
        alt = unescape(alt_match.group(1)) if alt_match else 'NextRole 화면'
        max_height = int(width * 180 / 178)
        count = max(1, math.ceil(height / max_height))
        part_height = math.ceil(height / count)
        pieces = []
        for i in range(count):
            offset = i * part_height
            size = min(part_height, height - offset)
            label = f'{alt} · {i+1}/{count}'
            pieces.append(f'<div class="capture-part"><p class="capture-part-caption">{escape(label)}</p><div class="capture-slice" style="aspect-ratio:{width}/{size}"><img src="{escape(src)}" alt="{escape(label)}" style="top:{-100*offset/size:.8f}%"></div></div>')
        return f'<figure class="doc-capture"><div class="screen-capture">{image}</div><figcaption class="capture-caption"><a href="{escape(src)}">원본 화면을 크게 보기 ↗</a></figcaption><div class="print-capture">{"".join(pieces)}</div></figure>'
    return re.sub(r'<p>(<img [^>]*src="(screenshots/[^"]+)"[^>]*>)</p>', wrap, body)


def render(stem, title, description):
    text = (ROOT / f'{stem}.md').read_text()
    body = markdown.markdown(text, extensions=['tables', 'fenced_code', 'toc', 'sane_lists'], extension_configs={'toc': {'slugify': slugify_unicode, 'title': '목차', 'toc_depth': '2-2'}})
    body = print_captures(body)
    page = f'''<!doctype html>
<html lang="ko"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{escape(title)} | NextRole</title><meta name="description" content="{escape(description)}"><link rel="canonical" href="https://hkjang.github.io/nextRole/{stem}.html"><link rel="icon" type="image/svg+xml" href="assets/logo.svg"><meta name="theme-color" content="#136e63"><link rel="stylesheet" href="assets/fonts.css"><style>{CSS}{CAPTURE_CSS}</style></head><body><header><a href="index.html"><img src="assets/logo.svg" alt="">NextRole</a><nav aria-label="문서 탐색"><a href="index.html">서비스 소개</a><a href="user-guide.html">사용자 가이드</a><a href="admin-guide.html">관리자 가이드</a><a href="{stem}.pdf">PDF</a><a href="{stem}.md">Markdown</a></nav></header><main>{body}</main><footer>NextRole v1.1.0 · 경력의 다음 가능성을 실험하다.</footer></body></html>'''
    (ROOT / f'{stem}.html').write_text(page)

if __name__ == '__main__':
    render('user-guide', '사용자 가이드', 'NextRole 경력 입력·이력서 분석·직무 비교·What-if 시뮬레이션·로드맵·개인 키의 상세 이미지 가이드입니다.')
    render('admin-guide', '관리자 가이드', 'NextRole 오프라인 Docker 배포, 네 환경변수, PostgreSQL, SSO·OIDC, AI 스트리밍, 데이터 연동·키 정책·백업의 상세 가이드입니다.')
