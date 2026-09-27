import { chromium } from 'playwright';
import { mkdir, writeFile } from 'node:fs/promises';
import path from 'node:path';
import { randomBytes } from 'node:crypto';
import { fileURLToPath } from 'node:url';
const base = process.env.NEXTROLE_URL || 'http://127.0.0.1:8080';
const email = process.env.NEXTROLE_TEST_ADMIN;
const password = process.env.NEXTROLE_TEST_PASSWORD;
if (!email || !password) throw new Error('테스트 관리자 자격증명을 환경변수로 입력하세요.');
const output = fileURLToPath(new URL('../../docs/screenshots/', import.meta.url));
await mkdir(output, {recursive:true});
const browser = await chromium.launch();
const ctx = await browser.newContext({viewport:{width:1440,height:1050}, locale:'ko-KR'});
const page = await ctx.newPage();
const errors=[]; page.on('pageerror',e=>errors.push(e.message));
async function api(url,method='GET',data){const response=await ctx.request.fetch(base+'/api/v1'+url,{method,data});if(!response.ok())throw new Error(`${method} ${url}: ${response.status()} ${await response.text()}`);return response.json();}
async function shot(name,route, mobile=false){await page.goto(base+route);await page.waitForLoadState('networkidle');await page.evaluate(()=>document.fonts.ready);await page.screenshot({path:path.join(output,`${name}${mobile?'-mobile':''}.png`),fullPage:true});if(route!='/login'){const overflow=await page.evaluate(()=>document.documentElement.scrollWidth>innerWidth+2);if(overflow)errors.push(`가로 넘침: ${name}${mobile?'-mobile':''}`)}console.log(`capture ${name}${mobile?'-mobile':''}`);}
await shot('login','/login');
await api('/auth/login','POST',{email,password});
const me=await api('/me');
await api('/me','PUT',{name:'운영 관리자',preferences:me.preferences});
const profile={name:'김도현',currentRole:'Java 백엔드 개발자',yearsExperience:10,careerBreakMonths:0,education:'대학교 졸업',region:'서울',domain:'금융',narrative:'Java 백엔드 개발자로 10년간 금융 도메인에서 일했습니다. Java, Spring, SQL을 활용한 API 개발과 Linux 운영을 담당했고 Docker 4년, Kubernetes 1년, CI/CD 4년 경험이 있습니다. Python은 기초 수준입니다. AI 플랫폼 엔지니어로 전환하고 싶습니다.',skills:[['Java',5,10],['Spring',5,8],['SQL',4,8],['Linux',4,6],['Docker',3,4],['Kubernetes',2,1],['CI/CD',4,4],['Python',1,1]].map(([name,level,years])=>({name,level,years,confidence:'explicit'})),certifications:[],preferences:['AI 플랫폼 엔지니어'],weeklyHours:10};
await api('/profile','PUT',profile);
const previous=await api('/simulations');for(const s of previous)await api('/simulations/'+s.id,'DELETE');
for(const jobId of ['ai-platform','data-engineer','devops'])await api('/simulate','POST',{jobId,months:6,addedSkills:jobId==='ai-platform'?[{name:'Python',level:4},{name:'Kubernetes',level:4}]:[],save:true});
let settings=await api('/admin/settings');const original={...settings.general};settings.general.approvalEnabled=true;settings.general.registrationEnabled=true;await api('/admin/settings','PUT',settings);
const demoPassword=randomBytes(24).toString('base64url');
const demoEmail='demo.user@nextrole.example';
const users=await api('/admin/users');const existing=users.find(u=>u.email===demoEmail);
if(existing)await api('/admin/users/'+existing.id,'PUT',{name:'김도현',role:'user',disabled:false,password:demoPassword});
else await api('/admin/users','POST',{email:demoEmail,name:'김도현',role:'user',password:demoPassword});
const userCtx=await browser.newContext();
async function userAPI(url,data,method='POST'){const res=await userCtx.request.fetch(base+'/api/v1'+url,{method,data});if(!res.ok())throw new Error('demo fixture failed: '+url);return res.json();}
await userAPI('/auth/login',{email:demoEmail,password:demoPassword});await userAPI('/profile',profile,'PUT');
const reviewSim=await userAPI('/simulate',{jobId:'ai-platform',months:6,addedSkills:[{name:'Python',level:4}],save:true});
await userAPI('/approvals',{simulationId:reviewSim.id,note:'주당 10시간 학습 계획과 DevOps 경유 경로를 검토해 주세요. 합성 시연 경력입니다.'});await userCtx.close();
const key=await api('/keys','POST',{name:'나의 커리어 도구',scopes:['profile:read','jobs:read','simulate:write','mcp:use'],expiresInDays:30});
await api('/keys/'+key.record.id+'/rotate','POST'); // The raw credential is never captured.
const routes=[['dashboard','/'],['profile','/profile'],['discover','/discover'],['compare','/compare'],['simulator','/simulator?job=ai-platform'],['roadmap','/roadmap?job=ai-platform'],['opportunities','/opportunities?job=ai-platform'],['saved','/saved'],['keys','/keys'],['api','/api'],['preferences','/preferences'],['approvals','/approvals'],['admin','/admin'],['admin-users','/admin/users'],['admin-general','/admin/general'],['admin-ai','/admin/ai'],['admin-security','/admin/security'],['admin-scoring','/admin/scoring'],['admin-providers','/admin/providers'],['admin-connectors','/admin/connectors'],['admin-audit','/admin/audit']];
for(const [name,route] of routes)await shot(name,route);
await page.goto(base+'/simulator?job=ai-platform');await page.getByRole('checkbox',{name:'Python',exact:true}).check();await page.getByRole('checkbox',{name:'Kubernetes',exact:true}).check();await page.waitForTimeout(600);await page.screenshot({path:path.join(output,'simulator.png'),fullPage:true});
await page.goto(base+'/');await page.locator('.profile-button').click();await page.screenshot({path:path.join(output,'profile-menu.png'),fullPage:true});
await page.setViewportSize({width:390,height:844});
for(const [name,route] of routes)await shot(name,route,true);
await api('/auth/logout','POST',{});await shot('login','/login',true);
await api('/auth/login','POST',{email,password});settings=await api('/admin/settings');settings.general=original;await api('/admin/settings','PUT',settings);
await writeFile(path.join(output,'capture-report.json'),JSON.stringify({version:'1.0.0',capturedAt:new Date().toISOString(),desktopViewport:{width:1440,height:1050},mobileViewport:{width:390,height:844},routes:routes.map(([name,route])=>({name,route})),errors},null,2));
await browser.close();if(errors.length)throw new Error(errors.join('\n'));
