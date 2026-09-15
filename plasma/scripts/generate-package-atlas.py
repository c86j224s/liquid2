#!/usr/bin/env python3
"""Generate the offline package atlas from go list; run from any directory."""
import argparse
import html
import json
import subprocess
from pathlib import Path

DOC_ROOT = Path(__file__).resolve().parents[1]
ROOT = DOC_ROOT
PREFIX = 'github.com/c86j224s/liquid2/plasma/'
FALLBACK = {
 'cmd/plasma': 'Plasma CLI 진입점. HTTP 서버, MCP 서버, 미션·소스·보고서 명령을 구성하고 실행합니다.',
 'cmd/plasma-article-pilot-preflight': 'Article 실험의 protocol·archive·실행 전 조건을 검증하는 CLI입니다.',
 'cmd/plasma-article-real-pilot-validate': '실제 Article pilot의 고정 protocol과 결과 기록을 검증합니다.',
 'cmd/plasma-report-experiment': '보고서 실험 설정·fixture·실행 식별자를 받아 기존 실험 runner를 호출합니다.',
 'cmd/plasma-report-il-phase0': 'Report IL 입력 bundle을 읽고 HTML/PDF 생성 실험을 실행합니다.',
 'cmd/plasma-report-il-recover': '기존 장부·artifact에서 검증된 IL checkpoint를 복구합니다. 쓰기는 별도 옵션입니다.',
 'cmd/plasma-report-il-recover-parts': '장문 Part artifact를 바탕으로 IL 재개 checkpoint를 복구하는 CLI입니다.',
 'cmd/plasma-report-il-reproject': '저장된 semantic IL에서 Markdown·HTML·PDF를 재투영합니다.',
 'internal/articlepilot': '수락된 Article Markdown을 읽기용 HTML로 변환하고 기존 PDF renderer에 인쇄를 위임합니다.',
 'internal/reportilcontract': 'IL source catalog, 저작 문서, 편집 메모, 장문 계획, checkpoint와 영수증의 자료형·검증 계약입니다.',
 'internal/reportilpdf': 'Chrome 인쇄 기반 PDF adapter. 입력·readiness·timeout·PDF 응답을 검사합니다.',
 'internal/reportmathassets': '고정된 KaTeX JavaScript를 embed하여 반환하는 정적 자산 패키지입니다.',
 'internal/reportmermaidassets': '고정 Mermaid·DOMPurify·renderer·CSS·라이선스를 embed하는 정적 자산 패키지입니다.',
 'internal/storage/sqlite/reportrunrepo': '논리 report run의 소속 이벤트·artifact와 삭제 판단 자료를 조회하고 transaction 단위로 정리합니다.',
}
GROUPS = ['실행 진입점', 'HTTP · MCP 표면', '애플리케이션 · 미션', '에이전트 · 대화', '연구 객체', '소스 · 수집', '보고서 실행 · IL', '보고서 workflow 단계', '저장 adapter', '실험 · 검증 · 기반']

def group(p):
 if p.startswith('cmd/'): return GROUPS[0]
 if p == 'internal/web' or p.startswith('internal/mcp'): return GROUPS[1]
 if p.startswith('internal/storage/'): return GROUPS[8]
 if p.startswith('internal/reportworkflow'): return GROUPS[7]
 if p.startswith('internal/research'): return GROUPS[4]
 if p.startswith(('internal/source', 'internal/connectors/')) or p == 'internal/confluenceaccess': return GROUPS[5]
 if p.startswith(('internal/agent', 'internal/conversation')): return GROUPS[3]
 if p.startswith(('internal/reportexperiment','internal/article')): return GROUPS[9]
 if p.startswith(('internal/report','internal/pdfdocument','internal/mermaid')): return GROUPS[6]
 if p.startswith(('internal/app','internal/mission','internal/ledger','internal/workflow')): return GROUPS[2]
 return GROUPS[9]

def run(*args): return subprocess.check_output(args, cwd=ROOT, text=True)
def esc(s): return html.escape(str(s), quote=True)
def parse_stream(text):
 decoder = json.JSONDecoder()
 while text.strip():
  item, end = decoder.raw_decode(text.lstrip()); yield item; text = text.lstrip()[end:]

def main():
 global ROOT
 parser=argparse.ArgumentParser(description=__doc__)
 parser.add_argument('--source-root',type=Path,default=DOC_ROOT,help='Plasma product root to inspect')
 parser.add_argument('--output',type=Path,default=DOC_ROOT/'docs/architecture/package-atlas.html')
 parser.add_argument('--legacy-label',default='',help='Historical milestone label, not a release claim')
 args=parser.parse_args()
 ROOT=args.source_root.resolve()
 raw = list(parse_stream(run('go','list','-json','./...')))
 rows=[]
 for o in raw:
  p=o['ImportPath'].removeprefix(PREFIX)
  if not o['ImportPath'].startswith(PREFIX): raise ValueError(o['ImportPath'])
  directory=Path(o['Dir']); files=o.get('GoFiles',[])+o.get('CgoFiles',[])
  docfiles=[]
  for name in files:
   text=(directory/name).read_text()
   if 'Package '+o['Name'] in text: docfiles.append(name)
  doc='\n\n'.join((directory/name).read_text().split('package '+o['Name'])[0].strip() for name in docfiles)
  role=o.get('Doc') or FALLBACK.get(p)
  if not role: raise ValueError('Missing documented role: '+p)
  rows.append(dict(id=p,group=group(p),role=role,doc=doc,basis='근거: 패키지 주석' if o.get('Doc') else '근거: 소스 진입점·공개 함수 확인 (수동 설명)',files=sorted(files),tests=sorted(o.get('TestGoFiles',[])+o.get('XTestGoFiles',[])),ignored=o.get('IgnoredGoFiles',[]),imports=sorted(x.removeprefix(PREFIX) for x in o.get('Imports',[]) if x.startswith(PREFIX)),external=sorted(x for x in o.get('Imports',[]) if not x.startswith(PREFIX)),embeds=sorted(o.get('EmbedFiles',[]))))
 ids={p['id'] for p in rows}
 for i,p in enumerate(rows):
  assert set(p['imports'])<=ids
  p['index']=i;p['incoming']=[x['id'] for x in rows if p['id'] in x['imports']]
 # Detect Go directories omitted by the active platform/build flags, rather than silently omitting them.
 dirs={str(p.parent.relative_to(ROOT)) for p in ROOT.rglob('*.go') if not any(x in p.relative_to(ROOT).parts for x in ['vendor','.git'])}
 excluded=sorted(dirs-ids)
 revision=run('git','rev-parse','HEAD').strip();env=json.loads(run('go','env','-json','GOOS','GOARCH'))
 data=dict(revision=revision,platform=env,groups=GROUPS,packages=rows,excludedDirectories=excluded,legacyLabel=args.legacy_label)
 def filelinks(p,names):
  if not names:
   return '<p class="muted">없음</p>'
  if args.legacy_label:
   return '<ul class="list">'+''.join('<li>'+esc(name)+'</li>' for name in names)+'</ul>'
  return '<ul class="list">'+''.join('<li><a href="'+esc('../../'+p['id']+'/'+name)+'">'+esc(name)+'</a></li>' for name in names)+'</ul>'
 def refs(names): return '<ul class="list">'+''.join('<li><a href="#pkg-'+str(next(p['index'] for p in rows if p['id']==n))+'">'+esc(n)+'</a></li>' for n in names)+'</ul>' if names else '<p class="muted">없음</p>'
 groups=''.join('<section class="group"><h3>'+esc(g)+'</h3>'+''.join('<button class="node" data-pkg="'+esc(p['id'])+'" title="'+esc(p['role'])+'">'+esc(p['id'])+'</button>' for p in rows if p['group']==g)+'</section>' for g in GROUPS)
 catalog=''
 for p in rows:
  catalog+='<article class="entry" id="pkg-'+str(p['index'])+'"><span class="badge">'+esc(p['group'])+'</span><h3 class="path">'+esc(p['id'])+'</h3><p>'+esc(p['role'])+'</p><small>'+esc(p['basis'])+'</small>'
  if p['doc']:catalog+='<details><summary>패키지 주석 전문</summary><pre>'+esc(p['doc'])+'</pre></details>'
  catalog+='<div class="cols"><div><h4>직접 import</h4>'+refs(p['imports'])+'</div><div><h4>역참조</h4>'+refs(p['incoming'])+'</div></div>'
  catalog+='<details><summary>소스 '+str(len(p['files']))+' · 테스트 '+str(len(p['tests']))+' · 제외 '+str(len(p['ignored']))+' 파일</summary><h4>운영 소스</h4>'+filelinks(p,p['files'])+'<h4>테스트</h4>'+filelinks(p,p['tests'])+'<h4>현재 build 제외</h4>'+filelinks(p,p['ignored'])+'</details>'
  catalog+='<details><summary>표준 라이브러리·외부 모듈 / embed</summary><pre>'+esc('\n'.join(p['external']))+'</pre>'+filelinks(p,p['embeds'])+'</details></article>'
 template=(DOC_ROOT/'docs/architecture/package-atlas.template.html').read_text()
 scope='go list ./... 기준 '+env['GOOS']+'/'+env['GOARCH']+' 기본 build에서 확인된 '+str(len(rows))+'개 Go 패키지.'
 scope+=' 다른 디렉터리: '+(', '.join(excluded) if excluded else '없음 (Go 파일이 있는 모든 디렉터리가 포함됨).')
 replacements={'__META__':esc(revision[:12]+' · '+env['GOOS']+'/'+env['GOARCH'])+' · '+str(len(rows))+' packages · '+str(sum(len(p['imports']) for p in rows))+' internal imports','__GROUPS__':groups,'__CATALOG__':catalog,'__COUNT__':str(len(rows)),'__SCOPE__':esc(scope),'__DATA__':json.dumps(data,ensure_ascii=False).replace('<','\\u003c')}
 for a,b in replacements.items():template=template.replace(a,b)
 if args.legacy_label:
  label=esc(args.legacy_label)
  template=template.replace('<title>Plasma Package Atlas</title>','<title>Plasma Legacy Atlas</title>').replace('<h1>Plasma Package Atlas</h1>','<h1>Plasma Legacy Atlas</h1>')
  banner='<p class="notice"><strong>LEGACY · '+label+'</strong><br>#483 시작 전 '+revision[:12]+'의 역사적 구조입니다. 릴리즈 태그 snapshot이 아니며 현재 설계 기준으로 사용하지 마세요. <a href="package-atlas.html">현재 지도 보기</a>. 파일명은 당시 구조를 설명하기 위한 정보이며 소스 링크를 제공하지 않습니다.</p>'
  template=template.replace('<div class="tools">',banner+'<div class="tools">',1)
 else:
  template=template.replace('<div class="tools">','<p><a href="package-atlas-v0.17-pre-refactor-legacy.html">v0.17 리팩토링 착수 전 · LEGACY 지도</a></p><div class="tools">',1)
 target=args.output.resolve();target.write_text(template)
 print(f'{len(rows)} packages, {sum(len(p["imports"]) for p in rows)} internal edges, {len(excluded)} excluded directories; wrote {target.name}')
if __name__=='__main__':main()
