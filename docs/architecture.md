# NextRole 구현 계약

Go 1.26 HTTP server + React/TypeScript/Mantine/Vite. PostgreSQL 영속화. 서비스 포트 8080. 런타임 환경변수는 POSTGRES_DSN, BOOTSTRAP_ADMIN, BOOTSTRAP_ADMIN_PASSWORD, ENCRYPTION_KEY (32바이트 base64) 네 개만 사용한다. 서버 설정은 DB에 암호화 저장. UI 번들과 한글 폰트는 이미지 내장. VERSION 1.0.0.

## 공통 API

모든 API /api/v1, JSON camelCase. 오류 {error:string}. 세션 HttpOnly SameSite=Lax cookie. 같은 origin mutation만 허용. API 키는 Bearer. 관리자/일반사용자/검토자 역할 admin/user/reviewer.

- GET /public: {name,version,registrationEnabled,approvalEnabled,providers:[{id,name,type}]} 공개 설정
- POST /auth/login {email,password} -> {user}
- POST /auth/logout -> {ok:true}
- POST /auth/register {email,password,name} -> {user}
- GET /me -> {id,email,name,role,preferences:{},version}
- PUT /me {name,preferences} -> user
- POST /me/password {currentPassword,newPassword}
- GET /auth/sso/{id}/start : SSO redirect
- GET /auth/sso/{id}/callback : callback

## 경력 API (internal/career에서 순수 엔진 및 데이터 타입 구현)

Profile: {name,currentRole,yearsExperience:number,education,region,domain,narrative,skills:[{name,level:number(0-5),years:number,confidence:'explicit'|'inferred'|'review'}],certifications:string[],preferences:string[],weeklyHours:number,careerBreakMonths:number}

Job: {id,title,category,description,skills:[{name,level,weight}],minExperience,domain,education,regions:string[],source:{name,url,synthetic:boolean},salaryRange,bridgeIds:string[]}

- GET /profile -> Profile (빈 사용자 프로필)
- PUT /profile Profile -> Profile
- POST /profile/parse {text} -> Profile (룰 기반 오프라인 파서; AI 스트림은 별도)
- POST /profile/upload multipart file -> Profile (PDF/DOCX/TXT 실제 추출)
- GET /jobs?q= -> Job[]
- GET /recommendations -> Simulation[] (Top 5)
- POST /simulate {jobId,months:3|6|12,addedSkills:[{name,level}],certifications?:string[],project?:string,save?:boolean} -> Simulation
- GET /simulations -> 저장한 Simulation[]
- DELETE /simulations/{id}
- GET /market?jobId= -> 실제·미만료 공고 표본의 지역/역량 빈도/급여 원문/일별 수집 추세 (과거 기준점 없으면 빈 추세)
- GET /opportunities?jobId= -> {jobs:Opportunity[],training:Opportunity[],skillFrequency:[{name,count}],synthetic:boolean}
- POST /ai/stream {task:'parse'|'explain'|'plan',text?,jobId?} -> SSE: event delta data {text}, event done data {mode:'offline'|'ai'}, event error data {error}; AI 호출 스트리밍 기본, maxTokens 1..262144
- POST /feedback {jobId,preference:'like'|'dislike'|'neutral'}
- GET /roadmap -> {completed:string[]}; PUT /roadmap {completed:string[]}

Simulation: {id?,job:Job,score:number,baselineScore:number,difficulty:number,estimatedMonths:number,gaps:[{name,required,current,gap,weight}],factors:[{name,score,max,reason}],roi:[{name,before,after,gain}],paths:[{id,title,roles:string[],months,skills:string[],reuse:number,cost:number,reason}],plan:[{month,title,tasks:string[],skills:string[]}],strengths:string[],warnings:string[],explanation:string,source:Source,createdAt?,scenario?:{jobId,months,addedSkills,certifications,project},marketDemand?,rankingReason?}

Opportunity: {id,title,organization,region,url,skills:string[],source:Source,description,cost?:number,salary?:string,deadline?:string}

## 키/승인/API

- GET /keys -> [{id,name,prefix,scopes:string[],createdAt,expiresAt,lastUsedAt,revokedAt}]
- POST /keys {name,scopes,expiresInDays:number} -> {key:one-time-string,record:Key}
- PUT /keys/{id} {name,scopes} -> Key
- POST /keys/{id}/rotate -> {key,record} (기존 키 즉시 폐기)
- DELETE /keys/{id} -> {ok:true}
- GET /key-scopes -> {available:string[],allowed:string[]}
- GET /approvals -> Approval[] (비활성시 404)
- POST /approvals {simulationId,note} -> Approval
- PUT /approvals/{id} {status:'approved'|'rejected',note} (reviewer/admin, 자기 승인 금지)
- POST /mcp : JSON-RPC 2.0 MCP initialize, tools/list, tools/call; GET /api/v1/openapi.json

## 관리자 API

- GET/PUT /admin/settings -> {general:{name,baseUrl,registrationEnabled,demoEnabled,approvalEnabled},ai:{enabled,baseUrl,model,apiKey,maxTokens,temperature,contextWindow,tokenParameter,sendTemperature,timeoutSeconds},security:{sessionHours,keyMaxDays,allowedKeyScopes:string[]},scoring:{skill,transfer,experience,domain,education,preference}}
- GET/POST /admin/users; PUT /admin/users/{id} {name,role,disabled,password?}; 사용자 POST {email,name,password,role}
- GET/POST /admin/providers; PUT/DELETE /admin/providers/{id}; Provider {id,name,type:'google'|'linkedin'|'kakao'|'naver'|'keycloak'|'oidc'|'oauth2',enabled,clientId,clientSecret,issuer,authorizationUrl,tokenUrl,userInfoUrl,scopes,subjectField,emailField,nameField} Secret read is empty + hasSecret boolean; blank update preserves secret.
- GET/POST /admin/connectors; PUT/DELETE /admin/connectors/{id}; Connector {id,name,type:'json'|'xml'|'csv'|'postgres'|'mysql',enabled,endpoint,apiKey,authHeader,dsn,query,rootPath,mapping:{[target:string]:sourcePath},dataset:'jobs'|'training'|'occupations',lastSync?,lastError?}. Credentials write-only.
- POST /admin/connectors/{id}/test -> {ok,count,preview}; POST /admin/connectors/{id}/sync -> {ok,count}
- POST /admin/import {dataset,records:object[]} -> {ok,count}
- GET /admin/audit -> [{id,actor,action,target,createdAt,details}]
- GET /admin/status -> {version,database,users,simulations,connectors,mode}
- POST /admin/ai/test -> {ok,message}

Secret management AES-GCM; bcrypt passwords; SHA256 random session/API token digests. No auto-link SSO accounts by unverified email. Role permissions on server. Imported data provenance retained. Scores explainable and deterministic; not hiring probabilities.


## 검증과 운영 경계

- `go test -race ./...` 및 `go vet ./...`; `TEST_POSTGRES_DSN`을 제공하면 무작위 격리 스키마에서 실제 PostgreSQL 통합 테스트를 실행하며 종료 시 삭제한다. 서비스가 읽는 런타임 변수는 앞의 네 개뿐이다.
- 브라우저: `web/e2e/service.spec.ts` — 로그인/프로필/시뮬레이션/모든 메뉴 새로고침/모바일/키 발급·회전·폐기/이력서 업로드/SSE. 검증 계정은 테스트용 환경변수로만 전달한다.
- SSO: 로컬 RSA/JWKS IdP로 서명·issuer·audience·만료·nonce·PKCE·state·replay·계정충돌·비활성 계정을 검증한다. 외부 공급자의 실계정 로그인은 해당 기관이 Client ID/Secret/콜백을 등록한 뒤 확인한다.
- 실제 공고·교육은 관리자 데이터 연동을 통해 수집한다. 기본 카탈로그와 시연 기회는 합성 표시한다. 외부 API 인증키나 외부 모델 가중치는 배포 이미지에 포함하지 않는다.
- 추천 순서는 적합도 + 선호(좋아요 +3) + 유효 실제 공고 수요(min(3,log2(N+1)))로 정렬하고, 싫어요 직무는 제외한다. 이 보정은 표시되는 여섯 요소 적합도 자체를 바꾸지 않는다.
- 이미지 검증은 GitHub Actions의 외부 통신을 차단한 Docker 내부 네트워크에서 실행한다. 서비스 이미지 하나만 릴리즈하고 PostgreSQL은 DSN의 별도 서버로 연결한다.
