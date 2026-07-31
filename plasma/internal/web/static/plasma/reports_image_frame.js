(function reportsImageFrame() {
  "use strict";
  const reports = window.Plasma.reports;
  const MESSAGE_TYPE = "plasma:image-viewer:open";
  function frameScript() {
    return `
<style>
.plasma-image-viewer-target{position:relative;display:inline-block;max-width:100%;vertical-align:top}
.plasma-image-viewer-target img{display:block;max-width:100%;height:auto}
.plasma-image-viewer-target>svg{display:block;max-width:100%;height:auto}
.plasma-image-viewer-open{position:absolute;top:8px;right:8px;z-index:2147483647;display:inline-flex;align-items:center;justify-content:center;width:30px;height:30px;border:1px solid rgba(255,255,255,.42);border-radius:999px;background:rgba(17,19,24,.78);color:#fff;box-shadow:0 8px 22px rgba(0,0,0,.28);font:700 17px/1 system-ui,sans-serif;opacity:0;transform:translateY(-2px);transition:opacity .12s ease,transform .12s ease}
.plasma-image-viewer-target:hover .plasma-image-viewer-open,.plasma-image-viewer-target:focus-within .plasma-image-viewer-open,.plasma-image-viewer-open:focus-visible{opacity:1;transform:translateY(0)}
@media (hover:none){.plasma-image-viewer-open{opacity:1;transform:translateY(0)}}
</style>
<script>
(function(){
  var type=${JSON.stringify(MESSAGE_TYPE)};
  function targetFor(img){return img.closest("a")||img.closest("picture")||img;}
  function details(img){return{type:type,src:img.currentSrc||img.src||img.getAttribute("src")||"",alt:img.getAttribute("alt")||"",title:img.getAttribute("title")||""};}
  function svgSelector(){return [".plasma-mermaid-diagram > svg",".mermaid svg","svg[id^='plasma-mermaid-']","svg[id^='mermaid-']","svg[aria-label='Mermaid 그래프']"].join(",");}
  function svgTitle(svg){return svg.getAttribute("aria-label")||(svg.querySelector("title")&&svg.querySelector("title").textContent)||"Mermaid 그래프";}
  function legend(svg){
    var figure=svg.closest(".plasma-mermaid-card"),legendNode=figure&&figure.querySelector(".plasma-mermaid-line-legend");
    if(!legendNode)return[];
    return Array.prototype.map.call(legendNode.querySelectorAll(".plasma-mermaid-line-legend-item"),function(item){
      var marker=item.querySelector(".plasma-mermaid-line-legend-marker");
      var label=((item.querySelector(".plasma-mermaid-line-legend-text")||item).textContent||"").replace(/\\s+/g," ").trim();
      var color=marker?(marker.style.backgroundColor||getComputedStyle(marker).backgroundColor):"";
      return{label:label,color:color};
    }).filter(function(item){return item.label;});
  }
  function svgDetails(svg){var title=svgTitle(svg);return{type:type,kind:"svg",svg:svg.outerHTML||"",alt:title,title:title,legend:legend(svg)};}
  function addButton(wrapper,label,handler){
      var button=document.createElement("button");
      button.type="button";
      button.className="plasma-image-viewer-open";
      button.title=label;
      button.setAttribute("aria-label",label);
      button.textContent="⌕";
      button.addEventListener("click",handler);
      wrapper.appendChild(button);
  }
  function enhance(){
    Array.prototype.forEach.call(document.querySelectorAll("img"),function(img){
      if(img.dataset.plasmaImageViewerBound==="true")return;
      var src=img.currentSrc||img.src||img.getAttribute("src")||"";
      if(!src)return;
      var target=targetFor(img);
      if(!target||target.dataset.plasmaImageViewerTarget==="true")return;
      var wrapper=document.createElement("span");
      wrapper.className="plasma-image-viewer-target";
      target.parentNode.insertBefore(wrapper,target);
      wrapper.appendChild(target);
      target.dataset.plasmaImageViewerTarget="true";
      img.dataset.plasmaImageViewerBound="true";
      addButton(wrapper,"이미지 크게 보기",function(event){event.preventDefault();event.stopPropagation();parent.postMessage(details(img),"*");});
    });
    Array.prototype.forEach.call(document.querySelectorAll(svgSelector()),function(svg){
      if(svg.dataset.plasmaImageViewerBound==="true"||!svg.parentNode||svg.dataset.plasmaImageViewerTarget==="true")return;
      var wrapper=document.createElement("span");
      wrapper.className="plasma-image-viewer-target plasma-image-viewer-target--svg";
      svg.parentNode.insertBefore(wrapper,svg);
      wrapper.appendChild(svg);
      svg.dataset.plasmaImageViewerTarget="true";
      svg.dataset.plasmaImageViewerBound="true";
      addButton(wrapper,"Mermaid 그래프 크게 보기",function(event){event.preventDefault();event.stopPropagation();parent.postMessage(svgDetails(svg),"*");});
    });
  }
  if(document.readyState==="loading")document.addEventListener("DOMContentLoaded",enhance);else enhance();
}());
</script>`;
  }

  function preparePlasmaHTMLPreview(content) {
    const html = String(content || "");
    const injection = frameScript();
    if (/<\/body\s*>/i.test(html)) {
      return html.replace(/<\/body\s*>/i, `${injection}</body>`);
    }
    return `${html}${injection}`;
  }

  function onFrameMessage(event) {
    const data = event.data || {};
    if (!data || data.type !== MESSAGE_TYPE) return;
    const frames = Array.from(document.querySelectorAll("iframe.plasma-html-preview-frame"));
    const sourceFrame = frames.find((frame) => frame.contentWindow === event.source);
    if (!sourceFrame) return;
    reports.openImageViewer(data);
  }

  window.addEventListener("message", onFrameMessage);

  Object.assign(reports, { prepareHTMLPreview: preparePlasmaHTMLPreview });
}());
