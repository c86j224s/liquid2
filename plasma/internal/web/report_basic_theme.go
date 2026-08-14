package web

// selfContainedReportCSS는 기본 HTML 보고서의 읽기 표면을 소유한다.
// 장식용 카드 대신 여백과 기능적 구분선으로 구조를 드러내며, 외부 폰트에 의존하지 않는다.
func selfContainedReportCSS() string {
	return `<style>
:root{color-scheme:light;--bg:#fff;--panel:#fff;--text:#111827;--muted:#475569;--line:#dbe4f0;--accent:#1d4ed8;--accent2:#2563eb;--code-bg:#f8fafc;--focus:#1d4ed8;--danger:#b91c1c;--heading-1:#172554;--heading-2:#1e3a8a;--heading-3:#1e40af;--heading-4:#1d4ed8;--heading-5:#2563eb;--heading-6:#0077b6}
body.dark{color-scheme:dark;--bg:#050505;--panel:#050505;--text:#f8fafc;--muted:#cbd5e1;--line:#3f3520;--accent:#d4af37;--accent2:#f0c75e;--code-bg:#111;--focus:#f0c75e;--danger:#f87171;--heading-1:#987922;--heading-2:#a17f22;--heading-3:#b58e27;--heading-4:#c89d2c;--heading-5:#d4af37;--heading-6:#f0c75e}
body{margin:0;background:var(--bg);color:var(--text);font-family:system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI","Noto Sans KR","Apple SD Gothic Neo",sans-serif;font-size:15px;line-height:1.68;letter-spacing:0}
.hero{display:flex;justify-content:space-between;gap:24px;align-items:flex-end;padding:42px clamp(20px,5vw,72px) 28px;border-bottom:1px solid var(--line);background:transparent}
.eyebrow{margin:0 0 8px;color:var(--accent);font-size:12px;font-weight:700;line-height:1.2;text-transform:uppercase}
h1{margin:0;font-size:clamp(28px,5vw,58px);font-weight:700;line-height:1.08;max-width:980px;color:var(--heading-1)}h2{font-size:24px;font-weight:600;color:var(--heading-2)}h3{font-size:20px;font-weight:600;color:var(--heading-3)}h4{font-size:18px;font-weight:500;color:var(--heading-4)}h5{font-size:16px;font-weight:500;color:var(--heading-5)}h6{font-size:15px;font-weight:500;color:var(--heading-6)}.sub{max-width:780px;color:var(--muted);margin:14px 0 0}
button{border:1px solid var(--line);background:transparent;color:var(--text);border-radius:0;padding:9px 12px;cursor:pointer;font:inherit}button:hover{border-color:var(--accent);color:var(--accent)}button:focus-visible,a:focus-visible,summary:focus-visible{outline:3px solid var(--focus);outline-offset:3px}
.layout{display:grid;grid-template-columns:minmax(180px,260px) minmax(0,1fr);gap:36px;max-width:1320px;margin:0 auto;padding:30px clamp(16px,4vw,56px) 80px}.rail{position:sticky;top:0;align-self:start;display:grid;gap:18px}
.metric,.media-panel,.notes,.report-body{background:transparent;border:0;border-radius:0}.metric{padding:0}.metric span{display:block;color:var(--muted);font-size:12px}.metric strong{font-size:30px;color:var(--accent)}.metric code{word-break:break-all}
nav{display:grid;gap:8px;margin-top:8px}nav a,.report-body a{color:var(--accent2);text-decoration-thickness:1px;text-underline-offset:3px}
.report-body{padding:clamp(10px,2vw,28px) 0;min-width:0;overflow-wrap:anywhere}.report-body>:is(h1,h2,h3,h4,h5,h6,p,ul,ol,blockquote){max-width:72ch}.report-body h1{margin-top:3.25rem}.report-body h1:first-child{margin-top:0;font-size:clamp(30px,5vw,54px)}.report-body h2{margin-top:38px;border-top:1px solid var(--line);padding-top:22px}.report-body h2.marked,.report-body h3.marked{text-decoration:underline;text-decoration-color:var(--accent);text-decoration-thickness:3px;text-underline-offset:5px}
.report-body p,.report-body li{font-size:17px}.report-body img{max-width:100%;height:auto;border-radius:0}.report-body table{display:block;width:100%;max-width:100%;overflow-x:auto;border-collapse:collapse;margin:18px 0;background:transparent}.report-body th,.report-body td{border:1px solid var(--line);padding:10px 12px;text-align:left;vertical-align:top}.report-body th{background:var(--code-bg);color:var(--heading-2);font-size:13px;font-weight:600}
pre,code{font-family:ui-monospace,SFMono-Regular,Menlo,Monaco,Consolas,"Liberation Mono",monospace}pre{max-width:100%;overflow:auto;padding:14px;border-left:3px solid var(--accent);background:var(--code-bg);border-radius:0}.report-markdown-raw{white-space:pre-wrap;overflow-wrap:anywhere}
.media-panel,.notes{grid-column:2;padding:24px 0 0;border-top:1px solid var(--line)}.section-head{display:flex;justify-content:space-between;gap:12px;align-items:center}.muted,.notes{color:var(--muted)}
.gallery{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:22px}.gallery figure{margin:0;border:0;border-radius:0;overflow:hidden;background:transparent}.gallery img{display:block;width:100%;height:auto}.gallery figcaption{display:grid;gap:4px;padding:10px 0}.gallery figcaption span{font-size:13px;color:var(--muted);word-break:break-word}
@media(max-width:820px){.hero{display:block}.hero button{margin-top:20px}.layout{display:block}.rail{position:static;margin-bottom:24px}.media-panel,.notes{margin-top:24px}.report-body{padding:12px 0}.report-body p,.report-body li{font-size:15px}}
</style>
`
}

// selfContainedBasicMermaidCSS는 공유 Mermaid 자산을 기본 HTML 테마에만 맞춘다.
func selfContainedBasicMermaidCSS() string {
	return `<style>
.plasma-mermaid-card{background:var(--code-bg);border:0;border-top:1px solid var(--line);border-bottom:1px solid var(--line);border-radius:0;padding:16px 0}
.plasma-mermaid-diagram,.plasma-mermaid-line-legend{color:var(--text)}
.plasma-mermaid-source{color:var(--muted)}
.plasma-mermaid-line-legend-item:focus-visible{outline:3px solid var(--focus);outline-offset:2px}
</style>
`
}

// selfContainedReportThemeScript는 밝은 기본값과 현재 페이지에 한정된 다크 모드 전환을 제공한다.
func selfContainedReportThemeScript() string {
	return `<script>(()=>{const b=document.body,t=document.getElementById("themeToggle");if(!t)return;t.hidden=false;const apply=dark=>{b.classList.toggle("dark",dark);t.setAttribute("aria-pressed",dark?"true":"false");t.setAttribute("aria-label",dark?"라이트 모드 켜기":"다크 모드 켜기");t.textContent=dark?"라이트 모드":"다크 모드"};const rerender=dark=>{const m=window.mermaid,r=window.Plasma?.reports;if(!m||!r?.renderPlasmaMermaid)return;if(!m.__plasmaThemeInitialize)m.__plasmaThemeInitialize=m.initialize.bind(m);m.initialize=config=>m.__plasmaThemeInitialize({...config,theme:dark?"dark":"default"});m.__plasmaConfigured=false;document.querySelectorAll(".plasma-mermaid-card").forEach(figure=>{const source=figure.querySelector(".plasma-mermaid-raw")?.textContent?.trim();if(!source)return;const pre=document.createElement("pre"),code=document.createElement("code");code.className="language-mermaid";code.textContent=source;pre.appendChild(code);figure.replaceWith(pre)});r.renderPlasmaMermaid(document)};apply(false);t.addEventListener("click",()=>{const dark=!b.classList.contains("dark");apply(dark);rerender(dark)})})();</script>
`
}
