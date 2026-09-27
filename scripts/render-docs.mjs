import { createServer } from 'node:http';
import { readFile, stat } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from '../web/node_modules/playwright/index.mjs';
const root=path.resolve(fileURLToPath(new URL('../docs/',import.meta.url)));
const mime={'.html':'text/html; charset=utf-8','.css':'text/css','.js':'text/javascript','.svg':'image/svg+xml','.png':'image/png','.json':'application/json','.woff2':'font/woff2','.pdf':'application/pdf','.vtt':'text/vtt; charset=utf-8','.mp4':'video/mp4'};
const server=createServer(async(req,res)=>{try{const url=new URL(req.url,'http://localhost');const target=path.resolve(root,'.'+decodeURIComponent(url.pathname));if(target!==root&&!target.startsWith(root+path.sep)){res.writeHead(403);return res.end();}const info=await stat(target);const file=info.isDirectory()?path.join(target,'index.html'):target;res.setHeader('Content-Type',mime[path.extname(file)]||'text/plain; charset=utf-8');res.end(await readFile(file));}catch{res.writeHead(404);res.end('Not found');}});
await new Promise(resolve=>server.listen(0,'127.0.0.1',resolve));
const base=`http://127.0.0.1:${server.address().port}`;
const browser=await chromium.launch();
try{
 const selectedGuide=process.argv.find(value=>value.startsWith('--guide='))?.slice(8);
 if(selectedGuide&&!['user-guide','admin-guide'].includes(selectedGuide))throw new Error('Unknown guide: '+selectedGuide);
 const page=await browser.newPage({viewport:{width:1440,height:1050},locale:'ko-KR'});
 const errors=[];page.on('pageerror',e=>errors.push(e.message));page.on('response',r=>{if(r.status()>=400)errors.push(r.status()+' '+r.url());});
 await page.setViewportSize({width:1200,height:630});await page.goto(base+'/assets/og-cover.svg');await page.screenshot({path:path.join(root,'assets/og-cover.png')});
 if(!process.argv.includes('--website-only')) for(const name of (selectedGuide?[selectedGuide]:['user-guide','admin-guide'])){
  await page.goto(base+'/'+name+'.html');await page.evaluate(()=>document.fonts.ready);await page.waitForLoadState('networkidle');
  const broken=await page.locator('img').evaluateAll(images=>images.filter(i=>!i.complete||i.naturalWidth===0).map(i=>i.src));if(broken.length)throw new Error('Missing guide images: '+broken.join(','));
  // Published PDFs must not retain links to the temporary rendering server.
  // Keep same-document fragments intact so the PDF table of contents works.
  await page.evaluate(({localOrigin,publicBase})=>{for(const anchor of document.querySelectorAll('a[href]')){const raw=anchor.getAttribute('href');if(!raw||raw.startsWith('#'))continue;const url=new URL(anchor.href);if(url.origin===localOrigin)anchor.href=publicBase+url.pathname+url.search+url.hash;}},{localOrigin:base,publicBase:'https://hkjang.github.io/nextRole'});
  await page.pdf({path:path.join(root,name+'.pdf'),format:'A4',tagged:true,outline:true,printBackground:true,preferCSSPageSize:true,displayHeaderFooter:true,headerTemplate:'<span></span>',footerTemplate:'<div style="font-size:9px;color:#65736e;width:100%;text-align:center">NextRole v1.0.0 · <span class="pageNumber"></span> / <span class="totalPages"></span></div>',margin:{top:'16mm',bottom:'18mm',left:'15mm',right:'15mm'}});
  console.log('generated '+name+'.pdf');
 }
 if(!process.argv.includes('--guides-only')) for(const mobile of [false,true]){
  await page.setViewportSize(mobile?{width:390,height:844}:{width:1440,height:1050});await page.goto(base+'/');await page.waitForLoadState('networkidle');await page.evaluate(()=>document.fonts.ready);
  await page.evaluate(async()=>{const images=[...document.images];for(const image of images)image.loading='eager';await Promise.all(images.map(image=>image.decode()));});
  const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2);if(overflow)throw new Error('Marketing horizontal overflow '+mobile);
  await page.screenshot({path:path.join(root,'screenshots',mobile?'website-mobile.png':'website.png'),fullPage:true});
 }
 if(errors.length)throw new Error(errors.join('\n'));
}finally{await browser.close();await new Promise(resolve=>server.close(resolve));}
