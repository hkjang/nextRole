# NextRole 구현 계약

Go 1.26 HTTP server + React/TypeScript/Mantine/Vite. PostgreSQL 영속화. 서비스 포트 8080. 런타임 환경변수는 POSTGRES_DSN, BOOTSTRAP_ADMIN, BOOTSTRAP_ADMIN_PASSWORD, ENCRYPTION_KEY (32바이트 base64) 네 개만 사용한다. 서버 설정은 DB에 암호화 저장. UI 번들과 한글 폰트는 이미지 내장. VERSION 1.1.0.

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

Profile: {name,currentRole,yearsExperience:number,education,region,domain,narrative,skills:[{name,level:number(0-5),years:number,confidence:'explicit'|'inferred'|'review'}],certifications:string[],preferences:string[],weeklyHours:number,careerBreakMonths:number,source:Source} — 분석용 name은 빈 값으로 저장하며 계정 표시 이름은 /me에서 별도 관리

Job: {id,title,category,description,skills:[{name,level,weight}],minExperience,domain,education,regions:string[],source:Source,salaryRange,bridgeIds:string[],occupationCode?,recruitmentCodes?:string[],ncsCodes?:string[],sourceCode?,skillAliases?:{alias:canonical}}

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
- POST /ai/stream {task:'parse'|'explain'|'plan',text?,jobId?,months?,addedSkills?,certifications?,project?} -> SSE: event delta data {text}, event done data {mode:'offline'|'ai'}, event error data {error}; AI 호출 스트리밍 기본, maxTokens 1..262144
- POST /feedback {jobId,preference:'like'|'dislike'|'neutral'}
- GET /roadmap -> {completed:string[]}; PUT /roadmap {completed:string[]}

Simulation: {id?,job:Job,score:number,baselineScore:number,difficulty:number,estimatedMonths:number,gaps:[{name,required,current,gap,weight}],factors:[{name,score,max,reason}],roi:[{name,before,after,gain}],paths:[{id,title,roles:string[],months,skills:string[],reuse:number,cost:number,reason}],plan:[{month,title,tasks:string[],skills:string[]}],strengths:string[],warnings:string[],explanation:string,source:Source,createdAt?,scenario?:{jobId,months,addedSkills,certifications,project},marketDemand?,rankingReason?}

Opportunity: {id,title,organization,region,url,skills:string[],source:Source,description,cost?:number,salary?:string,deadline?:string,raw?:object,code?,occupationCode?,ncsCode?,courseId?,courseRound?,startDate?,endDate?,tuitionReference?,skillsOrigin?}

## 출처·공공 원천·검토 모델

Source: `{kind:'public_api'|'user_input'|'synthetic'|'derived'|'external',name,url,synthetic:boolean,provider?,dataset?,recordId?,retrievedAt?,fields?:string[],version?,basis?,sourceIds?:string[]}`. 과거 출처에 kind가 없으면 external로 취급하며 synthetic=true가 우선한다.

SourceRecord: `{id,title,description?,occupationCode?,ncsCode?,officialLevel?,source:Source,raw:object}`. 원천 자료는 `nr_records.kind=occupation_details|ncs_units`에, 공고·훈련은 `jobs|training`에 저장한다. `raw`에는 선택한 공식 필드만 있다. 표준직무기술서의 11개 공식 항목에는 수준이 없어 프리셋이 officialLevel을 만들지 않는다.

- 고용24 직업상세·NCS 원천은 곧바로 직무 수준 모델이 되지 않는다. `skill_mapping`의 유효한 published 버전만 `mapping:{id}` 직무로 계산한다.
- `SkillMapping`은 원천 직무 ID, NCS 원천 ID, 내부 역량 ID·이름·별칭·0 초과~5 수준·가중치·각 원천 참조·근거·버전을 가진다. `mapping_history`는 수정·게시·삭제 버전을 보존한다.
- 원천 지문이 달라지거나 참조 자료가 없어지면 계산에서 제외한다. 수집 시각과 선택 필드 나열 순서만 달라지는 재수집은 제외 사유가 아니다.
- `jobCd` 직업코드와 `jobsCd` 채용 직종코드를 등치하지 않는다. 관리자가 검토한 `jobCodes` → `Job.recruitmentCodes`만 코드 연계에 사용한다.
- 훈련은 게시 NCS의 코드와 원문 `job_sdvn_cd`를 과정 `ncsCd`와 정확히 비교한다. 코드 접두어·제목에서 관계를 만들지 않는다.
- 합성 모드를 끄면 seed와 저장된 synthetic 직무·기회 모두 제외한다. public_api/derived로 반입한 occupations는 게시 매핑 검증을 우회할 수 없다.
- derived 모델에 요구 경력·학력 기준이 없으면 해당 점수는 0이며 이유를 표시한다. 나머지 배점으로 자동 재분배하지 않는다.

## 개인정보·동의 API

- GET /privacy (`profile:read`) → `{notice,version,consent:null|{version,acceptedAt,noticeHash}}`
- POST /privacy/consent (`profile:write`) `{version,accepted:true}` → 현재 안내와 동의. 현재 버전이 아니거나 accepted=false면 409.
- DELETE /privacy/consent (`profile:write`) → 동의와 본인 profile/simulation/roadmap/feedback/본인 approval 삭제. 계정·키·감사 기록은 유지.

현재 안내 버전·본문 해시와 동의가 일치해야 경력 쓰기·파싱·업로드, 추천·기회 조회, 시뮬레이션, AI, 선호·로드맵·승인 요청을 처리한다. MCP career_simulate/career_opportunities도 같은 검사를 적용한다. 조회와 도구 검색의 기존 권한은 별도로 적용한다. 동의 철회 중 저장 요청은 동의 행 잠금을 사용해 철회 완료 뒤 개인 자료를 다시 만들지 않는다.

분석용 name은 비우고 입력과 분석 결과의 이메일·전화·주민등록번호·식별정보 표시 줄을 제외한다. 알려진 프로필 이름도 내용에서 제거한다. 이력서 원본은 저장하지 않는다. 정규식 기반이므로 완전한 익명화를 보장하지 않으며 이용자 확인이 필요하다. 공공 API jobCont에는 개인 프로필을 자동 전송하지 않는다. 명시적으로 실행한 AI 기능은 제외 처리를 거친 내용을 설정된 AI 서버로 보낼 수 있다.

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
- GET/POST /admin/connectors; PUT/DELETE /admin/connectors/{id}; Connector {id,name,type:'json'|'xml'|'csv'|'postgres'|'mysql',enabled,endpoint,apiKey,authHeader,dsn,query,rootPath,mapping:{[target:string]:sourcePath},dataset:'jobs'|'training'|'occupations'|'occupation_details'|'ncs_units',presetId?,selectedFields?:string[],lastSync?,lastError?}. Credentials write-only.
- POST /admin/connectors/{id}/test -> {ok,count,preview}; POST /admin/connectors/{id}/sync -> {ok,count}
- POST /admin/import {dataset,records:object[]} -> {ok,count}
- GET /admin/connector-presets → `{presets:Connector[],metadata:[{id,name,dataset,description,specUrl,endpoint,rootPath,format,fields:[{name,label,description,required,defaultSelected}],defaultSelectedFields,requiredParams,defaultParams}]}`
- GET/PUT /admin/data-policy `{notice,version}`. 본문 변경 시 새 버전 필수. notice 30~20000바이트, version 1~80바이트.
- GET /admin/data → `{datasets:[{dataset,kind,provider,count,updatedAt}],mappingCount,publishedMappingCount}` (published count는 저장 상태 수, 개별 valid 확인 필요)
- GET /admin/source-records?dataset=&q=&offset=0&limit=50 → `{records:[{id,dataset,title,source,raw,occupationCode?,ncsCode?,officialLevel?,updatedAt}],total,offset,limit}`. limit 1~200. 개인 owner의 레코드는 제외.
- DELETE /admin/source-records/{dataset}/{id} → `{ok:true}`
- GET/POST /admin/mappings; PUT/DELETE /admin/mappings/{id}
- GET /admin/mappings/{id}/history → `SkillMapping[]` (최신순, 최대 500버전)
- POST /admin/mappings/{id}/publish `{version}` → 검토자를 기록하고 version+1 게시. 오래된 버전은 409.
- GET /admin/audit -> [{id,actor,action,target,createdAt,details}]
- GET /admin/status -> {version,database,users,simulations,connectors,mode}
- POST /admin/ai/test -> {ok,message}

SkillMapping: `{id,title,occupationRecordId,occupationCode,jobCodes:string[],ncsRecordIds:string[],skills:[{internalSkillId,name,aliases:string[],level,weight,sourceRecordIds:string[],basis}],basis,version,status:'draft'|'published'|'deleted',reviewedBy?,reviewedAt?,updatedAt,sourceHashes?,valid,validationWarning?}`. POST 새 초안은 version=1. PUT은 현재 version을 제출하며 수정 후 draft/version+1. 게시·삭제도 새 이력을 남긴다. sourceRecordIds는 선택한 원천 직무 또는 NCS의 내부 namespace ID여야 한다.

Secret management AES-GCM; bcrypt passwords; SHA256 random session/API token digests. No auto-link SSO accounts by unverified email. Role permissions on server. Imported data provenance retained. Scores explainable and deterministic; not hiring probabilities.


고용24 프리셋은 GET/authKey와 공식 rootPath·형식·요청 고정값을 검증하고 `selectedFields`로 투영한다. `authKey`를 일반 params·endpoint에 저장할 수 없다. 한 번에 한 페이지를 조회하며 자동 순회/스케줄러는 없다. 일반 연결의 `mapping`과 공공 원천의 `selectedFields`는 별개다.

## 검증과 운영 경계

- `go test -race ./...` 및 `go vet ./...`; `TEST_POSTGRES_DSN`을 제공하면 무작위 격리 스키마에서 실제 PostgreSQL 통합 테스트를 실행하며 종료 시 삭제한다. 서비스가 읽는 런타임 변수는 앞의 네 개뿐이다.
- 브라우저: `web/e2e/service.spec.ts` — 로그인/프로필/시뮬레이션/모든 메뉴 새로고침/모바일/키 발급·회전·폐기/이력서 업로드/SSE. 검증 계정은 테스트용 환경변수로만 전달한다.
- SSO: 로컬 RSA/JWKS IdP로 서명·issuer·audience·만료·nonce·PKCE·state·replay·계정충돌·비활성 계정을 검증한다. 외부 공급자의 실계정 로그인은 해당 기관이 Client ID/Secret/콜백을 등록한 뒤 확인한다.
- 고용24 4종은 공식 응답 형태의 mock으로 검증한다. 실제 승인 API 키가 없어 실제 조회 성공을 주장하지 않는다. 실제 공고·교육은 관리자 데이터 연동을 통해 수집한다. 기본 카탈로그와 시연 기회는 합성 표시한다. 외부 API 인증키나 외부 모델 가중치는 배포 이미지에 포함하지 않는다.
- 추천 순서는 적합도 + 선호(좋아요 +3) + 유효 실제 공고 수요(min(3,log2(N+1)))로 정렬하고, 싫어요 직무는 제외한다. 이 보정은 표시되는 여섯 요소 적합도 자체를 바꾸지 않는다.
- 이미지 검증은 GitHub Actions의 외부 통신을 차단한 Docker 내부 네트워크에서 실행한다. 서비스 이미지 하나만 릴리즈하고 PostgreSQL은 DSN의 별도 서버로 연결한다.
