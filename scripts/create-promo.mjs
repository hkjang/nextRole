#!/usr/bin/env node
// Rebuild with: node scripts/create-promo.mjs all [test-config.json] [base-url]
// Modes record/compose allow final screenshot capture between the two stages.
// Credentials are read only for local fixture login and are never persisted.
import fs from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { createServer } from 'node:http';
import { spawn } from 'node:child_process';
import { chromium } from '../web/node_modules/playwright/index.mjs';
const root = path.resolve(fileURLToPath(new URL('../', import.meta.url)));
const work = '/tmp/nextrole-promo';
const mode = process.argv[2] || 'all';
const configFile = process.argv[3] || '/tmp/nextrole-test/config.json';
const appURL = process.argv[4] || 'http://127.0.0.1:8080';
const duration = 60;
await fs.mkdir(work, { recursive: true });
function run(command, args, quiet = true) {
  return new Promise((resolve, reject) => {
    const process = spawn(command, args, { stdio: ['ignore', 'pipe', 'pipe'] });
    let stdout = '', stderr = '';
    process.stdout.on('data', data => { stdout += data; if (!quiet) globalThis.process.stdout.write(data); });
    process.stderr.on('data', data => { stderr += data; });
    process.on('error', reject);
    process.on('close', code => code === 0 ? resolve(stdout) : reject(new Error(`${command} exited ${code}: ${stderr.slice(-3000)}`)));
  });
}
async function probe(file) {
  return JSON.parse(await run('ffprobe', ['-v', 'error', '-show_format', '-show_streams', '-of', 'json', file]));
}
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
async function recordLiveSimulation() {
  const config = JSON.parse(await fs.readFile(configFile, 'utf8'));
  const browser = await chromium.launch({ headless: true });
  const context = await browser.newContext({ viewport: { width: 1600, height: 900 }, locale: 'ko-KR', recordVideo: { dir: path.join(work, 'live'), size: { width: 1600, height: 900 } } });
  try {
    const login = await context.request.post(`${appURL}/api/v1/auth/login`, { data: { email: config.BOOTSTRAP_ADMIN, password: config.BOOTSTRAP_ADMIN_PASSWORD } });
    if (!login.ok()) throw new Error(`Fixture login failed (${login.status()})`);
    const page = await context.newPage();
    const pageErrors = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    let latest;
    page.on('response', async response => {
      if (response.url().endsWith('/api/v1/simulate') && response.request().method() === 'POST' && response.ok()) latest = await response.json();
    });
    await page.goto(`${appURL}/simulator?job=ai-platform`, { waitUntil: 'networkidle' });
    await page.locator('.score-change').waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.getByRole('button', { name: '조건 초기화', exact: true }).click();
    await page.waitForTimeout(500);
    const baselineResponse = await context.request.post(`${appURL}/api/v1/simulate`, { data: { jobId: 'ai-platform', months: 6, addedSkills: [] } });
    const baseline = await baselineResponse.json();
    if (!baselineResponse.ok() || !Number.isFinite(baseline.score)) throw new Error('Baseline simulation unavailable');
    const viewportBoxes = {};
    for (const skill of ['Python', 'Kubernetes']) {
      const checkbox = page.getByRole('checkbox', { name: skill, exact: true });
      await checkbox.waitFor({ state: 'visible' });
      viewportBoxes[skill] = await checkbox.boundingBox();
      if (!viewportBoxes[skill] || viewportBoxes[skill].y > 875) throw new Error(`${skill} control is outside the recorded viewport`);
    }
    await page.evaluate(() => {
      const cursor = document.createElement('div');
      cursor.id = 'promo-pointer';
      cursor.style.cssText = 'position:fixed;left:0;top:0;width:24px;height:24px;border:3px solid #087d68;border-radius:50%;background:#d2f6dfbb;box-shadow:0 0 0 9px #7bc6a322;z-index:2147483647;pointer-events:none;transform:translate(-200px,-200px);transition:width .15s,height .15s;';
      document.body.append(cursor);
      document.addEventListener('mousemove', event => { cursor.style.transform = `translate(${event.clientX - 12}px,${event.clientY - 12}px)`; });
    });
    await page.mouse.move(1480, 210);
    const start = Date.now();
    const waitUntil = async second => { const remaining = start + second * 1000 - Date.now(); if (remaining > 0) await sleep(remaining); };
    await waitUntil(3);
    const pythonControl = page.getByRole('checkbox', { name: 'Python', exact: true });
    const pbox = await pythonControl.boundingBox();
    await page.mouse.move(pbox.x + pbox.width / 2, pbox.y + pbox.height / 2, { steps: 18 });
    const pythonResponse = page.waitForResponse(r => r.url().endsWith('/api/v1/simulate') && r.request().method() === 'POST' && r.ok());
    await pythonControl.check();
    const python = await (await pythonResponse).json();
    await page.locator('.score-change strong').getByText(`${Math.round(python.score)}점`, { exact: true }).waitFor();
    await waitUntil(7);
    const kubeControl = page.getByRole('checkbox', { name: 'Kubernetes', exact: true });
    const kbox = await kubeControl.boundingBox();
    await page.mouse.move(kbox.x + kbox.width / 2, kbox.y + kbox.height / 2, { steps: 18 });
    const kubeResponse = page.waitForResponse(r => r.url().endsWith('/api/v1/simulate') && r.request().method() === 'POST' && r.ok());
    await kubeControl.check();
    const kubernetes = await (await kubeResponse).json();
    await page.locator('.score-change strong').getByText(`${Math.round(kubernetes.score)}점`, { exact: true }).waitFor();
    if (!(python.score > baseline.score && kubernetes.score > python.score)) throw new Error('The actual scenario did not improve at each skill step; do not fabricate scores');
    await waitUntil(11);
    const scoreBox = await page.locator('.score-change').boundingBox();
    await page.mouse.move(scoreBox.x + scoreBox.width / 2, scoreBox.y + scoreBox.height / 2, { steps: 24 });
    await waitUntil(15);
    const liveSpan = (Date.now() - start) / 1000;
    const video = page.video();
    await context.close();
    const videoFile = await video.path();
    const metadata = await probe(videoFile);
    const offset = Math.max(0, Number(metadata.format.duration) - liveSpan);
    await run('ffmpeg', ['-y', '-i', videoFile, '-ss', offset.toFixed(3), '-t', '15', '-an', '-vf', 'fps=30,scale=1600:900', '-c:v', 'libx264', '-preset', 'fast', '-crf', '19', '-pix_fmt', 'yuv420p', '-movflags', '+faststart', path.join(work, 'whatif.mp4')]);
    const facts = { capturedAt: new Date().toISOString(), jobId: baseline.job.id, jobTitle: baseline.job.title, baseline: baseline.score, python: python.score, kubernetes: kubernetes.score, synthetic: baseline.source.synthetic, duration: 15, viewport: { width: 1600, height: 900 }, actions: ['Python 목표 수준 선택', 'Kubernetes 목표 수준 선택'] };
    if (pageErrors.length) throw new Error(`Live UI errors: ${pageErrors.join(', ')}`);
    await fs.writeFile(path.join(work, 'recording.json'), JSON.stringify(facts, null, 2));
    console.log(`Actual simulator recorded: ${Math.round(facts.baseline)} → ${Math.round(facts.python)} → ${Math.round(facts.kubernetes)} (source synthetic=${facts.synthetic})`);
  } finally { await browser.close(); }
}
function stamp(seconds) {
  const ms = Math.round(seconds * 1000);
  return `00:${String(Math.floor(ms / 60000)).padStart(2, '0')}:${String(Math.floor(ms / 1000) % 60).padStart(2, '0')}.${String(ms % 1000).padStart(3, '0')}`;
}
function sceneMarkup(scene, index) {
  const media = scene.video ? '<video id="whatif" preload="auto" muted playsinline src="/work/whatif.mp4"></video>' : `<img src="${scene.source || `/docs/screenshots/${scene.image}.png`}" alt="${scene.label} 실제 화면">`;
  return `<section class="scene feature" data-index="${index}" data-start="${scene.start}" data-end="${scene.end}"><div class="feature-copy"><div class="eyebrow">${scene.label}</div><h2>${scene.title}</h2><p>${scene.description}</p><div class="feature-pill">${scene.pill}</div>${scene.video ? '<div id="actual-score" class="actual-score"></div><p class="evidence">합성 경력·직무로 시연한 실제 계산<br>점수는 취업 확률이 아닙니다.</p>' : ''}</div><div class="browser-frame"><div class="chrome"><i></i><i></i><i></i><span>NextRole · ${scene.label}</span><b>●</b></div><div class="screen">${media}</div></div></section>`;
}
async function captureComparison() {
  const config = JSON.parse(await fs.readFile(configFile, 'utf8'));
  const browser = await chromium.launch({ headless: true });
  try {
    const context = await browser.newContext({ viewport: { width: 1440, height: 1050 }, locale: 'ko-KR' });
    const login = await context.request.post(`${appURL}/api/v1/auth/login`, { data: { email: config.BOOTSTRAP_ADMIN, password: config.BOOTSTRAP_ADMIN_PASSWORD } });
    if (!login.ok()) throw new Error(`Comparison fixture login failed (${login.status()})`);
    const page = await context.newPage();
    await page.goto(`${appURL}/compare`, { waitUntil: 'networkidle' });
    const selector = page.getByPlaceholder('관심 있는 직무를 최대 3개 선택하세요');
    for (const title of ['AI 플랫폼 엔지니어', 'DevOps 엔지니어', '데이터 엔지니어']) {
      await selector.click();
      await page.getByRole('option', { name: title, exact: true }).click();
    }
    await page.keyboard.press('Escape');
    await page.locator('.compare-card').nth(2).waitFor();
    await page.evaluate(() => document.fonts.ready);
    await page.screenshot({ path: path.join(work, 'compare.png'), fullPage: true });
    console.log('Actual three-role comparison captured without saving profile changes');
  } finally { await browser.close(); }
}
async function compose() {
  await captureComparison();
  const facts = JSON.parse(await fs.readFile(path.join(work, 'recording.json'), 'utf8'));
  const version = (await fs.readFile(path.join(root, 'VERSION'), 'utf8')).trim();
  const segments = [
    { start: 0, end: 6, text: '당신의 다음 역할, 선택 전에 실험하세요. AI 경력전환 시뮬레이터 NextRole.' },
    { start: 6, end: 13, text: '내 경력을 입력하면, 보유 역량과 다음 직무의 가능성이 선명해집니다.' },
    { start: 13, end: 16, text: `같은 경력, 현재 적합도 ${Math.round(facts.baseline)}점. 역량을 더하면 어떻게 달라질까요?` },
    { start: 16, end: 20, text: `Python을 선택하면 실제 적합도 ${Math.round(facts.python)}점. 즉시 다시 계산합니다.` },
    { start: 20, end: 28, text: `Kubernetes까지 더하면 ${Math.round(facts.kubernetes)}점. 무엇부터 배울지, 변화와 근거로 확인하세요.` },
    { start: 28, end: 36, text: '목표 직무를 나란히 비교하고, 나에게 맞는 전환 경로를 선택하세요.' },
    { start: 36, end: 44, text: '3개월, 6개월, 12개월. 막연한 계획을 단계별 실행으로 연결합니다.' },
    { start: 44, end: 49, text: '환경변수 네 개로 배포하고, 관리자 화면에서 서비스 운영을 설정합니다.' },
    { start: 49, end: 53, text: '개인 API 키와 MCP, 유연한 데이터 연동으로 내부 업무까지 연결하세요.' },
    { start: 53, end: 60, text: '현재의 경력에서, 다음 가능성으로. NextRole. GitHub에서 만나보세요.' },
  ];
  const scenes = [
    { start: 6, end: 13, label: '01  내 경력', title: '내 경력부터,<br>선명하게.', description: '자연어 경력을 정리하고<br>보유 역량과 근거를 확인하세요.', pill: '경력 → 역량 프로필', image: 'profile' },
    { start: 13, end: 28, label: '02  WHAT-IF SIMULATION', title: '만약 이 역량을<br>더한다면?', description: '직접 선택하고,<br>실제 점수 변화를 확인하세요.', pill: 'Python + Kubernetes', video: true },
    { start: 28, end: 36, label: '03  직무 비교', title: '가능성을<br>나란히 비교.', description: '적합도 · 부족 역량 · 준비 기간<br>하나의 화면에서 비교합니다.', pill: '나에게 맞는 다음 직무', source: '/work/compare.png' },
    { start: 36, end: 44, label: '04  나의 로드맵', title: '다음 선택을<br>오늘의 실행으로.', description: '학습과 프로젝트를 연결해<br>단계별로 준비해 보세요.', pill: '3개월 · 6개월 · 12개월', image: 'roadmap' },
    { start: 44, end: 49, label: '05  관리자 설정', title: '내부망에서도,<br>운영은 간결하게.', description: '네 개의 환경변수로 시작하고<br>관리 화면에서 설정합니다.', pill: 'Go + React · 단일 서비스 이미지', image: 'admin-general' },
    { start: 49, end: 53, label: '06  API · MCP', title: '개인의 실험을<br>조직의 도구로.', description: '개인 키 관리와 권한 설정,<br>API · MCP · 데이터 연동.', pill: '함께 연결되는 경력 도구', image: 'api' },
  ];
  const html = `<!doctype html><html lang="ko"><head><meta charset="UTF-8"><link rel="stylesheet" href="/docs/assets/fonts.css"><style>
*{box-sizing:border-box}html,body{margin:0;width:1920px;height:1080px;overflow:hidden;background:#092b2b;color:#eef9f1;font-family:'Noto Sans KR',sans-serif}body:before{content:'';position:absolute;inset:0;background:radial-gradient(ellipse at 88% 14%,#26755a80,transparent 44%),radial-gradient(ellipse at 4% 85%,#206a5c66,transparent 46%),linear-gradient(120deg,#0b2428,#0c3430);z-index:0}.grid{position:absolute;inset:0;background-image:linear-gradient(#9cdbb008 1px,transparent 1px),linear-gradient(90deg,#9cdbb008 1px,transparent 1px);background-size:80px 80px;mask-image:linear-gradient(transparent,#000)}.brand{position:absolute;top:42px;left:64px;display:flex;align-items:center;gap:16px;font-size:30px;font-weight:700;letter-spacing:-1px;z-index:10}.brand img{width:45px;height:45px}.top-note{position:absolute;right:70px;top:55px;font-size:18px;letter-spacing:3px;color:#a8c7b9;z-index:10}.scene{position:absolute;inset:0;opacity:0;pointer-events:none}.intro-copy{position:absolute;left:100px;top:235px;z-index:2}.eyebrow{font-size:22px;font-weight:700;letter-spacing:2px;color:#b7e899;margin-bottom:26px}.intro h1{font-size:84px;line-height:1.32;letter-spacing:-5px;margin:0 0 34px;font-weight:700}.intro p{font-size:31px;color:#bdd4c9;line-height:1.65;margin:0}.intro .accent{color:#bcefa4}.intro .mini-screen{position:absolute;width:800px;height:650px;right:-95px;top:188px;border-radius:28px;border:12px solid #ffffff1a;overflow:hidden;transform:rotate(-7deg);box-shadow:0 40px 100px #0008}.intro .mini-screen img{width:100%;display:block}.tags{display:flex;gap:12px;margin-top:40px}.tags span,.feature-pill{padding:12px 19px;border:1px solid #bbdfc147;border-radius:13px;color:#dbeddf;font-size:21px;background:#ffffff06}.feature-copy{position:absolute;left:64px;top:215px;width:448px}.feature h2{font-size:53px;line-height:1.35;letter-spacing:-2.8px;margin:0 0 30px;word-break:keep-all}.feature-copy>p{font-size:25px;line-height:1.8;color:#b5d0c1;margin:0 0 30px}.feature-pill{font-size:20px;display:inline-block;max-width:440px;line-height:1.5}.browser-frame{position:absolute;left:550px;top:170px;width:1306px;height:780px;border-radius:22px;overflow:hidden;background:#fff;box-shadow:0 30px 85px #0006;border:1px solid #d8e5da66}.chrome{height:44px;display:flex;align-items:center;gap:9px;padding:0 20px;background:#eaf0e9;color:#576b61;border-bottom:1px solid #dce6dc;font-size:15px}.chrome i{width:10px;height:10px;border-radius:50%;background:#b0beb4}.chrome i:first-child{background:#bda593}.chrome i:nth-child(2){background:#c9c39d}.chrome span{margin:auto}.chrome b{color:#237760}.screen{width:1306px;height:736px;overflow:hidden;position:relative;background:#f9fbf8}.screen>img{display:block;width:1306px;height:auto;transform-origin:top center}.screen video{width:1306px;height:736px;display:block;object-fit:fill}.actual-score{font-size:56px;font-weight:700;letter-spacing:-2px;margin-top:37px;color:#c3f19c;white-space:nowrap}.feature-copy .evidence{font-size:17px;line-height:1.65;margin-top:20px;color:#9eb9a9}.outro{text-align:center}.outro .logo{position:absolute;top:166px;left:901px;width:118px}.outro h2{position:absolute;top:298px;width:100%;font-size:71px;letter-spacing:-3px;line-height:1.35;margin:0}.outro .product{position:absolute;top:481px;width:100%;font-size:40px;font-weight:700;color:#d8ecd7;letter-spacing:1px}.outro .cta{position:absolute;top:589px;left:543px;width:834px;padding:24px 20px;background:#c4eca1;color:#143c31;font-size:36px;font-weight:700;border-radius:18px;letter-spacing:-1px;box-shadow:0 14px 60px #103829}.outro .sub{position:absolute;top:714px;width:100%;font-size:24px;color:#bad0c0}.caption{position:absolute;left:62px;right:62px;bottom:38px;height:60px;border-radius:14px;background:#061d20e8;display:flex;align-items:center;justify-content:center;padding:10px 24px;font-size:26px;line-height:1.5;text-align:center;color:#f6fff7;z-index:20}.progress{position:absolute;bottom:0;left:0;height:5px;background:#b8e69c;z-index:30;width:0}.scene-number{position:absolute;left:66px;bottom:126px;font-size:18px;color:#83a996;letter-spacing:3px}.corner{position:absolute;right:67px;bottom:115px;font-size:16px;color:#9fc4ae}.orb{position:absolute;width:400px;height:400px;border-radius:50%;border:1px solid #9fd9a31a;right:-100px;bottom:-100px}
</style></head><body><div class="grid"></div><div class="orb"></div><div class="brand"><img src="/docs/assets/logo.svg" alt=""><span>NextRole</span></div><div class="top-note">YOUR NEXT POSSIBILITY</div>
<section class="scene intro" data-index="0" data-start="0" data-end="6"><div class="intro-copy"><div class="eyebrow">AI 경력전환 시뮬레이터</div><h1>당신의 다음 역할,<br><span class="accent">선택 전에 실험하세요.</span></h1><p>현재의 경력에서,<br>다음 가능성으로.</p><div class="tags"><span>경력 분석</span><span>역량 실험</span><span>경로 비교</span></div></div><div class="mini-screen"><img src="/docs/screenshots/dashboard.png" alt="NextRole 대시보드"></div></section>
${scenes.map((scene, i) => sceneMarkup(scene, i + 1)).join('')}
<section class="scene outro" data-index="7" data-start="53" data-end="60"><img class="logo" src="/docs/assets/logo.svg" alt="NextRole"><h2>나에게 맞는 다음 경력,<br>직접 확인하세요.</h2><div class="product">NextRole</div><div class="cta">github.com/hkjang/nextRole</div><p class="sub">오프라인 배포 · 한국어 UI · 사용자·관리자 가이드</p></section>
<div class="caption"></div><div class="progress"></div><div class="corner">NextRole v${version} · 실제 서비스 화면</div>
<script>const captions=${JSON.stringify(segments)};const facts=${JSON.stringify(facts)};const elements=[...document.querySelectorAll('.scene')];let started=0,done=false;const video=document.querySelector('#whatif');
window.startPromo=()=>{started=performance.now();requestAnimationFrame(frame)};
function frame(now){const t=Math.min(60,(now-started)/1000);let active=0;elements.forEach((el,i)=>{const start=Number(el.dataset.start);if(t>=start)active=i;});elements.forEach((el,i)=>{const elapsed=t-Number(el.dataset.start);let opacity=0;if(i===active)opacity=Math.min(1,Math.max(0,elapsed/.65));else if(i===active-1)opacity=1-Math.min(1,Math.max(0,(t-Number(elements[active].dataset.start))/.65));if(t>59.35)opacity*=Math.max(0,(60-t)/.65);el.style.opacity=opacity;const img=el.querySelector('.screen>img');if(img){const pan=Math.max(0,Math.min(38,elapsed*3.8));img.style.transform='translateY(-'+pan+'px)';}});if(t>=13&&t<28){if(video.paused)video.play();const value=t<16?facts.baseline:t<20?facts.python:facts.kubernetes;document.querySelector('#actual-score').textContent=Math.round(facts.baseline)+' → '+Math.round(value)+'점';}else if(!video.paused)video.pause();const current=captions.find(c=>t>=c.start&&t<c.end);document.querySelector('.caption').textContent=current?current.text:'';document.querySelector('.progress').style.width=(t/60*100)+'%';if(t<60)requestAnimationFrame(frame);else{done=true;window.promoFinished=true;}}
</script></body></html>`;
  await fs.writeFile(path.join(work, 'story.html'), html);
  const mime = { '.html': 'text/html; charset=utf-8', '.css': 'text/css', '.svg': 'image/svg+xml', '.png': 'image/png', '.woff2': 'font/woff2', '.mp4': 'video/mp4' };
  const server = createServer(async (req, res) => {
    try {
      const pathname = new URL(req.url, 'http://localhost').pathname;
      const base = pathname.startsWith('/docs/') ? path.join(root, 'docs') : work;
      const relative = pathname.startsWith('/docs/') ? pathname.slice(6) : pathname === '/' ? 'story.html' : pathname.slice('/work/'.length);
      const target = path.resolve(base, relative);
      if (!target.startsWith(base + path.sep)) { res.writeHead(403); return res.end(); }
      const data = await fs.readFile(target);
      res.setHeader('Content-Type', mime[path.extname(target)] || 'application/octet-stream');
      res.setHeader('Content-Length', data.length);
      res.end(data);
    } catch { res.writeHead(404); res.end('Not found'); }
  });
  await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
  const base = `http://127.0.0.1:${server.address().port}`;
  const browser = await chromium.launch({ headless: true });
  let rawVideo;
  try {
    const context = await browser.newContext({ viewport: { width: 1920, height: 1080 }, locale: 'ko-KR', recordVideo: { dir: path.join(work, 'story'), size: { width: 1920, height: 1080 } } });
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    page.on('response', response => { if (response.status() >= 400) errors.push(`${response.status()} ${new URL(response.url()).pathname}`); });
    await page.goto(base, { waitUntil: 'networkidle' });
    await page.evaluate(async () => {
      await document.fonts.ready;
      await Promise.all([...document.images].map(image => image.decode()));
      const video = document.querySelector('video');
      if (video.readyState < 2) await new Promise(resolve => video.addEventListener('loadeddata', resolve, { once: true }));
    });
    const start = Date.now();
    await page.evaluate(() => window.startPromo());
    for (let second = 0; second < duration; second += 10) {
      await sleep(10000);
      console.log(`Composing ${Math.min(second + 10, duration)}/${duration}s`);
    }
    await page.waitForFunction(() => window.promoFinished === true, null, { timeout: 5000 });
    const span = (Date.now() - start) / 1000;
    const video = page.video();
    await context.close();
    rawVideo = await video.path();
    if (errors.length) throw new Error(`Story resource/render errors: ${errors.join(', ')}`);
    const metadata = await probe(rawVideo);
    await fs.writeFile(path.join(work, 'timing.json'), JSON.stringify({ offset: Math.max(0, Number(metadata.format.duration) - span), span }));
  } finally { await browser.close(); await new Promise(resolve => server.close(resolve)); }
  // An original synthesized pad, generated here; no samples, recording, melody,
  // third-party music, voice clone, or external audio service is used.
  const audioScript = `import math,struct,wave\nsr=32000\nchords=[(146.832,174.614,220,261.626),(116.541,146.832,174.614,220),(130.813,164.814,196,246.942),(130.813,146.832,196,261.626)]\nwith wave.open(${JSON.stringify(path.join(work, 'original-pad.wav'))},'wb') as out:\n out.setnchannels(2);out.setsampwidth(2);out.setframerate(sr)\n for block in range(60):\n  samples=bytearray()\n  for j in range(sr):\n   t=block+j/sr; section=int(t/7.5); local=t%7.5; chord=chords[section%4]; previous=chords[(section-1)%4]; blend=min(1,local/1.5); fade=min(1,t/2,(60-t)/2.5); left=0;right=0\n   for k in range(4):\n    wave1=math.sin(2*math.pi*chord[k]*t)+.09*math.sin(4*math.pi*chord[k]*t);wave0=math.sin(2*math.pi*previous[k]*t)+.09*math.sin(4*math.pi*previous[k]*t);value=(wave1*blend+wave0*(1-blend))*.024*fade;pan=.45+.2*math.sin(.12*t+k);left+=value*pan;right+=value*(1-pan)\n   samples.extend(struct.pack('<hh',int(left*32767),int(right*32767)))\n  out.writeframes(samples)\n`;
  await fs.writeFile(path.join(work, 'audio.py'), audioScript);
  await run('python3', [path.join(work, 'audio.py')]);
  const timing = JSON.parse(await fs.readFile(path.join(work, 'timing.json'), 'utf8'));
  const target = path.join(root, 'docs/assets/nextrole-promo.mp4');
  await run('ffmpeg', ['-y', '-ss', timing.offset.toFixed(3), '-i', rawVideo, '-i', path.join(work, 'original-pad.wav'), '-t', String(duration), '-map', '0:v:0', '-map', '1:a:0', '-vf', 'fps=30,scale=1920:1080', '-c:v', 'libx264', '-preset', 'medium', '-crf', '21', '-maxrate', '3200k', '-bufsize', '6400k', '-pix_fmt', 'yuv420p', '-profile:v', 'high', '-level:v', '4.2', '-c:a', 'aac', '-b:a', '128k', '-ar', '48000', '-movflags', '+faststart', '-metadata', 'title=NextRole — 다음 경력을 실험하세요', '-metadata', 'comment=Actual NextRole UI; original synthesized ambient audio; no external music samples', target]);
  const vtt = 'WEBVTT\n\n' + segments.map((segment, i) => `${i + 1}\n${stamp(segment.start)} --> ${stamp(segment.end)}\n${segment.text}\n`).join('\n');
  await fs.writeFile(path.join(root, 'docs/assets/nextrole-promo.ko.vtt'), vtt);
  const transcript = `# NextRole 홍보 영상 대본\n\n[영상 보기](assets/nextrole-promo.mp4) · [한국어 자막](assets/nextrole-promo.ko.vtt)\n\n- 분량: 60초\n- 화면: 1920 × 1080, 30fps\n- 공개 위치: GitHub Pages 문서 사이트. Docker 릴리스 첨부 파일과 분리합니다.\n- 실제 촬영: ${facts.capturedAt} / NextRole v${version}\n- 자료: 실행 중인 NextRole 실제 UI와 최종 페이지 캡처\n- 오디오: 이 영상용으로 코드에서 직접 합성한 오리지널 앰비언트. 외부 음악·샘플·음성 사용 없음\n- 접근성: 큰 한국어 화면 자막, WebVTT 자막, 아래 전체 대본 제공\n\n## 전체 대본과 화면 설명\n\n| 구간 | 화면 | 자막 |\n| --- | --- | --- |\n${segments.map(s => `| ${stamp(s.start).slice(3, 8)}–${stamp(s.end).slice(3, 8)} | ${s.start < 6 ? 'NextRole 로고와 실제 대시보드' : s.start < 13 ? '내 경력 프로필' : s.start < 28 ? '실제 What-if 조작 화면' : s.start < 36 ? '직무 비교' : s.start < 44 ? '나의 로드맵' : s.start < 49 ? '관리자 일반 설정' : s.start < 53 ? 'API · MCP 안내' : '로고와 GitHub 주소'} | ${s.text} |`).join('\n')}\n\n## What-if 시연 검증\n\n시뮬레이션 장면은 현재 실행 중인 서비스에서 Python, Kubernetes 체크박스를 실제 클릭한 녹화입니다. 화면을 임의 편집해 점수를 올리지 않았으며 다음 API 응답을 검증하고 촬영합니다. 사용자 프로필을 저장·변경하거나 새 시뮬레이션을 저장하지 않습니다.\n\n| 상태 | 실제 계산 점수 | 영상 표시 |\n| --- | ---: | ---: |\n| 추가 역량 없음 | ${facts.baseline} | ${Math.round(facts.baseline)}점 |\n| Python 목표 수준 | ${facts.python} | ${Math.round(facts.python)}점 |\n| Python + Kubernetes 목표 수준 | ${facts.kubernetes} | ${Math.round(facts.kubernetes)}점 |\n\n합성 경력·직무 예시로 시연합니다. 적합도 점수는 취업 확률이나 채용 보장이 아닙니다. 영상에는 실제 API 키·비밀번호·개인정보를 노출하지 않습니다. 외부 SSO·AI·고용24는 해당 서비스 접근이 가능한 망에서 연동하며, 폐쇄망에서는 내부 서비스와 반입 데이터를 사용합니다.\n\n## 재생성\n\nNode.js 24, 저장소의 Playwright, Chromium, ffmpeg/ffprobe, Python 3이 필요합니다. 별도 테스트 인스턴스와 합성 경력 프로필을 사용하세요.\n\n\`\`\`bash\nnode scripts/create-promo.mjs record /안전한/테스트-config.json http://127.0.0.1:8080\n# docs/screenshots의 최종 캡처 확인 후\nnode scripts/create-promo.mjs compose /안전한/테스트-config.json http://127.0.0.1:8080\n\`\`\`\n\n테스트 config의 BOOTSTRAP_ADMIN / BOOTSTRAP_ADMIN_PASSWORD로 로컬 서비스에 로그인합니다. 자격증명은 녹화 파일·대본·로그에 저장하지 않습니다. 중간 렌더링 파일은 /tmp/nextrole-promo에 생성합니다. MP4는 H.264/AAC, yuv420p, faststart이며 최대 30 MB를 확인합니다.\n`;
  await fs.writeFile(path.join(root, 'docs/promo-script.md'), transcript);
  const metadata = await probe(target);
  const stat = await fs.stat(target);
  const video = metadata.streams.find(s => s.codec_type === 'video');
  const audio = metadata.streams.find(s => s.codec_type === 'audio');
  if (video.codec_name !== 'h264' || video.width !== 1920 || video.height !== 1080 || audio?.codec_name !== 'aac' || stat.size > 30_000_000 || Math.abs(Number(metadata.format.duration) - duration) > .2) throw new Error('Output video does not meet delivery requirements');
  console.log(JSON.stringify({ file: 'docs/assets/nextrole-promo.mp4', seconds: Number(metadata.format.duration), bytes: stat.size, codec: video.codec_name, audio: audio.codec_name, width: video.width, height: video.height, framesPerSecond: video.r_frame_rate }, null, 2));
}
if (mode === 'record' || mode === 'all') await recordLiveSimulation();
if (mode === 'compose' || mode === 'all') await compose();
if (!['record', 'compose', 'all'].includes(mode)) throw new Error('Use mode record, compose or all');
