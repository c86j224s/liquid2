package web

import (
	"os/exec"
	"strings"
	"testing"
)

func TestReportMathStaticAssetOrder(t *testing.T) {
	index := string(mustReadStatic(t, "static/index.html"))
	ordered := []string{"vendor/katex/katex.min.css", "report_math.css", "vendor/markdown-it.min.js", "vendor/markdown-it-texmath.js", "vendor/purify.min.js", "vendor/katex/katex.min.js", "report_math.js", "app.js"}
	last := -1
	for _, asset := range ordered {
		at := strings.Index(index, asset)
		if at < 0 || at <= last {
			t.Fatalf("math asset %q missing or out of order", asset)
		}
		last = at
	}
	for _, asset := range []string{"static/report_math.js", "static/report_math.css", "static/vendor/markdown-it-texmath.js", "static/vendor/katex/katex.min.js", "static/vendor/katex/katex.min.css", "static/vendor/katex/fonts/KaTeX_Main-Regular.woff2"} {
		if len(mustReadStatic(t, asset)) == 0 {
			t.Fatalf("empty static asset %q", asset)
		}
	}
}

func TestReportMathBrowserFixtureAndKaTeXContract(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node is required")
	}
	script := `
const fs=require("fs"), vm=require("vm");
const markdownit=require("./static/vendor/markdown-it.min.js");
global.window=global;
vm.runInThisContext(fs.readFileSync("./static/vendor/markdown-it-texmath.js","utf8"));
vm.runInThisContext(fs.readFileSync("./static/report_math.js","utf8"));
const fixtures=JSON.parse(fs.readFileSync("./testdata/report_math_cases.json","utf8"));
for(const fixture of fixtures){
  const md=markdownit({html:false,linkify:true,breaks:true});
  installMarkdownItMath(md);
  const html=md.render(fixture.markdown);
  const inline=(html.match(/plasma-math-inline/g)||[]).length;
  const display=(html.match(/plasma-math-display/g)||[]).length;
  if(inline!==fixture.inline||display!==fixture.display) throw new Error(fixture.name+": "+inline+"/"+display+" in "+html);
  for(const tex of (fixture.tex||[])) if(!html.includes('data-tex="'+tex+'"')) throw new Error(fixture.name+": TeX missing "+tex+" in "+html);
  for(const raw of (fixture.raw||[])) if(raw.startsWith("$")&&!html.includes(raw)) throw new Error(fixture.name+": raw text missing "+raw+" in "+html);
  for(const expected of (fixture.contains||[])) if(!html.includes(expected)) throw new Error(fixture.name+": trailing content missing "+expected+" in "+html);
  if(fixture.name==="table_link"&&(!html.includes("<table>")||!html.includes("href=\"https://example.com\""))) throw new Error("table/link regression");
  if(fixture.name==="invalid_tex"&&!html.includes(">\\(\\notacommand{\\)</span>")) throw new Error("placeholder doubled raw delimiters");
}
const designedHolders=[];
const designedRoot={querySelectorAll(){return designedHolders;}};
function textNode(value, tagName="P"){
  return {nodeValue:value,parentElement:{tagName,parentElement:designedRoot},replaceWith(fragment){this.fragment=fragment;}};
}
const visible=textNode("Before \\(x\\) and \\[y\\] after.");
const code=textNode("Code \\(z\\)","CODE");
const link=textNode("Link \\(q\\)","A");
const svg=textNode("SVG \\(s\\)","svg");
global.NodeFilter={SHOW_TEXT:4};
global.document={
  createTreeWalker(){let index=0;const nodes=[visible,code,link,svg];return {currentNode:null,nextNode(){this.currentNode=nodes[index++]||null;return this.currentNode;}};},
  createDocumentFragment(){return {children:[],append(value){this.children.push(value);}};},
  createElement(){const holder={dataset:{},classList:{add(){}},removeAttribute(){},textContent:""};designedHolders.push(holder);return holder;}
};
renderDesignedTextMath(designedRoot);
if(!visible.fragment||visible.fragment.children.length!==5) throw new Error("sentence-middle designed math was not split");
if(designedHolders.length!==2||designedHolders[0].textContent!=="\\(x\\)"||designedHolders[1].textContent!=="\\[y\\]") throw new Error("designed delimiters were not preserved");
if(code.fragment||link.fragment||svg.fragment) throw new Error("designed exclusion regression");
const malformed=textNode("\\(".repeat(32*1024));
global.document.createTreeWalker=()=>{let done=false;return {currentNode:null,nextNode(){if(done)return false;done=true;this.currentNode=malformed;return true;}};};
const malformedStarted=Date.now();
renderDesignedTextMath(designedRoot);
if(Date.now()-malformedStarted>1000) throw new Error("unmatched designed math exceeded 1s");
if(malformed.fragment) throw new Error("unmatched designed math changed");
const heading={dataset:{},classList:{marked:false,toggle(name){this.marked=!this.marked;}},addEventListener(type,listener){this.listener=listener;}};
bindReportHeadingInteractions({querySelectorAll(selector){if(selector!=="h2,h3") throw new Error("heading selector");return [heading];}});
if(heading.tabIndex!==0||!heading.listener) throw new Error("heading binding missing");
heading.listener();if(!heading.classList.marked) throw new Error("heading interaction missing");
bindReportHeadingInteractions({querySelectorAll(){return [heading];}});
if(heading.dataset.plasmaHeadingBound!=="true") throw new Error("heading binding marker missing");
const katex=require("./static/vendor/katex/katex.min.js");
if(katex.version!=="0.17.0") throw new Error("KaTeX version");
let received;
global.katex={renderToString(tex,options){received=options;return katex.renderToString(tex,options)}};
global.DOMPurify={sanitize(value,options){if(!options.USE_PROFILES.html||!options.USE_PROFILES.mathMl||!options.USE_PROFILES.svg) throw new Error("HTML/MathML/SVG profiles missing");return value;}};
const classes=new Set();
const node={dataset:{tex:"x^2",display:"false"},textContent:"\\(x^2\\)",innerHTML:"",removeAttribute(){},classList:{add(v){classes.add(v)}}};
renderPlasmaMath({querySelectorAll(){return [node]}});
if(!node.innerHTML.includes("<math")||!node.innerHTML.includes("aria-hidden=\"true\"")) throw new Error("accessible KaTeX output missing");
for(const [key,want] of Object.entries({throwOnError:true,trust:false,output:"htmlAndMathml",maxSize:20,maxExpand:1000,displayMode:false})) if(received[key]!==want) throw new Error("option "+key);
const sanitizeOptions=plasmaMathSanitizeOptions();
if(!sanitizeOptions.USE_PROFILES.html||!sanitizeOptions.USE_PROFILES.mathMl||!sanitizeOptions.USE_PROFILES.svg) throw new Error("HTML/MathML/SVG profiles missing");
for(const tex of ["\\sqrt{x}","\\widehat{abc}","\\overbrace{x+y}","\\xrightarrow{test}"]){
  const markup=katex.renderToString(tex,plasmaMathOptions);
  if(!markup.includes("<svg")||!markup.includes("<math")||!markup.includes("aria-hidden=\"true\"")) throw new Error("KaTeX SVG fixture missing: "+tex);
}
const bad={dataset:{tex:"\\notacommand{",display:"false"},textContent:"\\(\\notacommand{\\)",innerHTML:"",removeAttribute(){},classList:{add(v){classes.add(v)}}};
renderPlasmaMath({querySelectorAll(){return [bad]}});
if(bad.textContent!=="\\(\\notacommand{\\)"||!classes.has("plasma-math-error")) throw new Error("raw fallback missing");
delete global.DOMPurify;
const trusted={dataset:{tex:"x+1",display:"false"},textContent:"\\(x+1\\)",innerHTML:"",removeAttribute(){},classList:{add(){}}};
renderPlasmaMath({querySelectorAll(){return [trusted]}},{trustedPlaceholders:true});
if(!trusted.innerHTML.includes("<math")) throw new Error("trusted export rendering missing");
`
	if out, err := exec.Command("node", "-e", script).CombinedOutput(); err != nil {
		t.Fatalf("browser math fixture: %v: %s", err, out)
	}
}
