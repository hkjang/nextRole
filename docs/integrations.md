# NextRole 데이터 연동 가이드

**v1.1.0** · [공식 전체 항목·활용 목적](work24-data-plan.md) · [관리자 검토 절차](admin-guide.html#직무역량-매핑-검토와-게시)

관리자는 **관리자 → 데이터 연동**에서 데이터 수집 위치, 인증키, 조회 조건과 필드 매핑을 저장합니다. 배포 환경변수를 추가하지 않습니다. 설정을 저장한 뒤 **연결 테스트**에서 변환된 레코드를 확인하고 **동기화**로 가져옵니다. API 실패 시 합성 채용정보나 가짜 훈련과정을 대신 생성하지 않습니다.

## 연동 방식과 범위

| 형식 | 접근 방식 | 적합한 데이터 |
| --- | --- | --- |
| JSON | HTTP(S) GET/POST | 사내 인사·채용 API, NCS 변환 게이트웨이 |
| XML | HTTP(S) GET/POST, UTF-8 | 고용24 채용·훈련 API, 레거시 XML 서비스 |
| CSV | HTTP(S) GET/POST, UTF-8/BOM 허용 | 사내 파일 배포 서버, 정기 반입 데이터 |
| PostgreSQL | 전용 SELECT 계정과 읽기 전용 트랜잭션 | 사내 직무 사전 뷰, 채용 데이터 마트 |
| MySQL 8 | 전용 SELECT 계정과 읽기 전용 트랜잭션 | 교육/LMS 뷰, 채용 시스템 뷰 |
| 직접 가져오기 | 관리자 JSON 가져오기 | 폐쇄망으로 반입한 승인 데이터 |

데이터 종류는 `occupation_details`(직업상세 원천), `ncs_units`(NCS 능력단위 원천), `jobs`(채용공고), `training`(훈련과정), `occupations`(별도 승인 사내 직무)입니다. 채용공고와 직무 사전은 별도 데이터입니다. 채용 API에서 받은 공고 제목을 목표 직무와 역량 수준으로 임의 변환하지 않습니다.

한 번의 테스트·동기화는 설정한 API **한 페이지** 또는 SQL **한 조회**를 읽습니다. 전체 페이지 자동 순회나 주기 스케줄러는 포함하지 않습니다. 고용24의 `startPage` / `pageNum` 같은 조회 매개변수를 변경해 필요한 페이지를 가져오거나, 사내 수집 게이트웨이에서 페이지 순회·증분 처리를 수행하세요.

## 공통 설정

| 항목 | 의미 |
| --- | --- |
| `name` | 사용자에게 표시할 데이터 출처 이름 |
| `type` | `json`, `xml`, `csv`, `postgres`, `mysql` |
| `dataset` | `jobs`, `training`, `occupations`, `occupation_details`, `ncs_units` |
| `presetId` | 고용24 프리셋 ID. 일반 연동은 비움 |
| `selectedFields` | 고용24에서 보관할 공식 필드 이름 목록. 일반 mapping과 별개 |
| `enabled` | 연동 사용 여부. 비활성 설정도 관리자 테스트 가능 |
| `endpoint` | API의 최종 HTTP(S) 주소. 인증정보는 별도 API 키 필드 사용 |
| `method` | `GET`(기본) 또는 `POST` |
| `params` | 비밀 아닌 쿼리 매개변수 객체. 기존 URL 매개변수와 합침 |
| `body` | POST 고정 요청 본문. 최대 1 MiB |
| `apiKey` | 인증키. 화면 조회 응답에는 원문 대신 등록 여부 반환 |
| `authHeader` | 인증 헤더 이름. 기본 `Authorization` |
| `apiKeyParam` | API 키를 query로 받는 서비스의 매개변수 이름. 예: `authKey` |
| `dsn` | DB 연결 문자열. 비밀번호가 포함되므로 비밀값으로 취급 |
| `query` | SELECT 또는 읽기 전용 WITH SQL |
| `rootPath` | 응답에서 레코드 목록까지의 경로. CSV/DB에는 비워 둠 |
| `mapping` | `{ "NextRole 대상 필드": "원본 필드 경로" }` |

`apiKeyParam`이 있으면 API 키를 URL query로 전달하고, 없으면 `authHeader`로 전달합니다. Bearer 인증은 API 키 값에 `Bearer 발급받은값`을 입력합니다. `params`, `body`, `endpoint`에는 자격증명을 직접 넣지 마세요. 이 필드들은 관리자가 다시 조회할 수 있습니다. 저장 후 비밀값을 비운 상태로 수정하면 기존 값을 유지하는 방식으로 관리합니다.

HTTP 응답 10 MiB, 최대 5,000건, 전체 연결 20초, SQL 서버 실행 15초 제한을 적용합니다. XML DTD와 외부 엔티티를 허용하지 않습니다. HTTP 리디렉션도 따르지 않으므로 서비스가 이동했다면 최종 URL을 입력합니다. URL의 사용자명/비밀번호와 `#fragment`는 지원하지 않습니다.

## 필드 매핑

JSON `{"data":{"items":[...]}}`이면 `rootPath`는 `data.items`입니다. XML `<wantedRoot><wanted>...</wanted></wantedRoot>`이면 `wantedRoot.wanted`입니다. XML 속성은 `@code`, 텍스트와 속성이 함께 있는 요소의 본문은 `#text`로 접근합니다. XML 네임스페이스 접두사는 제외하고 요소 이름을 사용합니다.

| 매핑 예 | 동작 |
| --- | --- |
| `"title": "job.name"` | 중첩 객체의 문자열 |
| `"title": "items[0].name"` | 배열 첫 번째 객체의 문자열 |
| `"skills": "tags[].label"` | 배열 전체에서 label 목록 추출 |
| `"skills[].name": "requirements[].label"` | 배열 객체의 name 필드 생성 |
| `"skills[].level": "requirements[].rank"` | 같은 배열 객체에 수준 추가 |
| `"source.name": "=사내 직무 사전"` | 문자열 상수 |
| `"source.synthetic": "=false"` | JSON 불리언 상수 |
| `"skills": "=[]"` | 빈 배열 상수 |
| `"id": "concat:trprId,trprDegr"` | 과정 ID와 회차를 조합해 고유 ID 생성 |

매핑을 비워 두면 원본 객체 필드 이름을 그대로 사용합니다. 매핑을 지정하면 선택한 필드만 가져옵니다. 원본 필드가 없거나 대상 경로가 충돌하면 전체 수집을 실패 처리하므로 테스트에서 구조를 확인하세요. 임의 JavaScript, 셸 명령 또는 계산식은 실행하지 않습니다.

`concat:`은 2~8개 원본 필드의 문자열·숫자 값을 URL 인코딩한 후 콜론으로 연결합니다. 예를 들어 과정 `C100`과 회차 `2`는 `C100:2`입니다. 원본 값의 콜론 등 특수문자는 인코딩해 조합 충돌을 방지합니다. 비어 있는 값과 배열은 허용하지 않습니다. 한 가져오기 배치 안에 ID가 중복되면 전체 배치를 저장하지 않으며, 다른 날짜에 같은 ID를 가져오면 기존 항목을 갱신합니다.

CSV와 DB의 숫자 문자열은 `cost`, `minExperience`, 직무 역량의 `level`·`weight` 필드에서 숫자로 변환합니다. 채용·훈련 `skills`와 직무의 `regions`, `bridgeIds`는 JSON 배열 또는 세미콜론·파이프 구분 문자열을 사용할 수 있습니다. 직무의 `skills`는 `{name,level,weight}` 객체 배열이어야 합니다. NextRole의 수준(0~5)과 가중치로 변환하는 규칙은 기관의 직무 담당자가 검증해야 합니다.

출처가 없으면 커넥터 이름과 query를 제거한 endpoint 주소를 보완하며, 출처 kind가 없는 일반 자료는 external로 구분합니다. source.synthetic를 생략했다는 사실은 실제 데이터의 진위나 이용 권한을 증명하지 않습니다. 교육·채용 원문 링크는 원본에서 받아야 하며, 존재하지 않는 링크를 생성하지 않습니다.

## 고용24 프리셋과 필드 선택

관리자 화면에서 서버 제공 프리셋을 선택하면 공식 엔드포인트·형식·목록 구조·필수 매개변수가 채워집니다. 프리셋은 비활성·인증키 없는 상태로 제공하며 실제 `jobCd`·`jobCont`·날짜 조건은 관리자가 입력합니다. `GET /api/v1/admin/connector-presets`로 같은 메타데이터와 필수/기본 선택 필드를 조회할 수 있습니다.

| presetId | dataset | 응답 구조 | 조회 조건 |
| --- | --- | --- | --- |
| `work24-occupation-summary` | `occupation_details` | XML `jobSum` 객체 | `jobCd`, 고정 `target=JOBDTL`, `jobGb=1`, `dtlGb=1` |
| `work24-ncs` | `ncs_units` | JSON `result` 안 능력단위명별 객체 | 관리자 수행직무 `jobCont`, 선택 `limit`(1~5000) |
| `work24-jobs` | `jobs` | XML `wantedRoot.wanted` | `callTp=L`, `startPage`(1~1000), `display`(1~100) |
| `work24-training` | `training` | XML `HRDNet.srchList.scn_list` | `outType=1`, `pageNum`(1~1000), `pageSize`(1~100), 시작일 검색 범위, 정렬 |

고용24 OPEN-API는 이용 신청·승인된 인증키가 필요합니다. [공식 이용 안내](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiIntro.do)와 각 API 이용 범위를 확인하세요. 개발에는 실제 승인 키가 제공되지 않아 실서비스 조회는 아직 검증하지 않았습니다. 테스트는 공식 정상·오류 형태의 모의 응답을 사용합니다.

프리셋은 GET과 `apiKeyParam=authKey`를 사용합니다. 인증키는 `apiKey` 비밀 필드에만 입력합니다. URL과 일반 매개변수 안 `authKey`는 거부합니다. 형식·dataset·rootPath·고정 요청값을 임의로 바꾸면 저장 검증에서 거부합니다. 승인된 내부 프록시를 사용할 때는 `presetId`를 유지하고 동일한 공식 응답 계약을 제공해야 합니다.

`selectedFields`에 고른 항목만 `raw`에 보존합니다. 필수 식별·표시 항목은 해제할 수 없고, 미지원·중복 이름도 거부합니다. 선택하지 않은 민감하거나 불필요한 세부 필드를 응답 전체와 함께 보관하지 않습니다. 관련전공·자격·관련직업 같은 중첩 그룹도 명세에 있는 하위 항목만 보존합니다. 날짜와 임금 원문을 해석할 수 없으면 임의로 값을 만들어 넣지 않습니다.

공통 출처에는 `kind=public_api`, `provider=한국고용정보원`, API별 dataset, 공식 명세 URL, 원천 ID, 수집 시각과 선택 필드 목록을 기록합니다. 공식 원문 전체 항목 목록은 [고용24 데이터 활용·출처 관리](work24-data-plan.md)에 유지하며, 실제 저장 필드와 구분합니다.

### 채용 프리셋 예제

아래 ID·조건은 설정 방법을 설명하는 예입니다. 인증키와 조회 범위는 승인된 운영 값으로 입력합니다.

```json
{
  "name": "고용24 채용정보",
  "presetId": "work24-jobs",
  "type": "xml", "dataset": "jobs", "enabled": false,
  "method": "GET",
  "endpoint": "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do",
  "apiKey": "관리자 비밀 필드에 입력", "apiKeyParam": "authKey",
  "params": {"callTp":"L","returnType":"XML","startPage":"1","display":"100"},
  "rootPath": "wantedRoot.wanted",
  "selectedFields": ["wantedAuthNo","title","company","wantedInfoUrl","jobsCd","region","sal","closeDt"],
  "mapping": {}
}
```

채용 목록에는 요구기술 목록이 없습니다. `skills`를 비우고 제목에서 기술을 추론하지 않습니다. 과거 자동 추론 레코드도 역량 빈도·추세에 사용하지 않습니다. 일반 사내 데이터에서 실제 명시된 `skills`를 반입한 경우에만 관리자 확인 자료(`skillsOrigin=admin_reviewed`)로 구분합니다. 고용24 공고만 수집한 환경에서 공고 수·지역·급여 원문은 있어도 역량 빈도가 비어 있을 수 있습니다.

### 훈련 조건과 비용

`srchTraStDt`와 `srchTraEndDt`는 각각 `YYYYMMDD` 형식의 **훈련 시작일 검색 범위**이며 시작일이 종료일보다 늦으면 거부합니다. 개별 과정의 종료일 조건을 뜻하지 않습니다. `sort`는 ASC/DESC, `sortCol`은 1·2·3·5 중 하나입니다. 서비스 프리셋은 XML을 사용합니다.

과정 ID `trprId`와 회차 `trprDegr`를 결합한 고유 ID를 사용합니다. `courseId`, `courseRound`, `startDate`, `endDate`, `ncsCode`와 선택 원문을 보존합니다. 원문 `courseMan`(수강비)·`realMan`(실제 훈련비)은 `tuitionReference` 문자열로 표시하며 **개인의 확정 본인부담금이 아닙니다**. 중단되어 Null을 반환하는 `eiEmplCnt3Gt10`은 저장하지 않습니다.

## 공공 원천에서 검토된 직무 모델 만들기

1. 직업상세·NCS 프리셋으로 `occupation_details`·`ncs_units`를 가져옵니다. NCS 조회에는 관리자가 작성한 직무 설명을 사용하며 개인 이력서를 자동 전송하지 않습니다.
2. **원천·가공 데이터**에서 원문·공식 출처·코드·선별 필드를 확인합니다.
3. **역량 매핑 검토**에서 목표 직무, 원천 레코드 ID, 내부 역량 ID·이름·별칭·수준·가중치·근거를 작성합니다.
4. 초안을 검토하고 현재 버전을 게시합니다. 게시·유효한 모델만 `derived` 출처의 목표 직무로 제공됩니다.

공공 원문의 `jobAbil`, `knowldg`, `knwg_tchn_attd`와 업무활동 중요도·수준은 NextRole의 내부 0~5 요구 수준이 아닙니다. 표준직무기술서의 공식 11항목에는 별도 수준 필드도 없습니다. 원문에 없는 수준이나 가중치를 자동 생성하지 않습니다. 별칭 연결은 내부 표현 정규화이며 공식 NCS 동등성을 보증하지 않습니다.

원천 레코드를 수정·삭제하면 해당 내용에 근거한 매핑은 무효해져 다시 검토·게시해야 합니다. 수집 시각이나 선택 필드 나열 순서만 바뀌면 내용이 바뀐 것으로 보지 않습니다. 수정한 매핑은 초안으로 돌아가며 삭제 후에도 버전 이력은 남습니다.

직업상세 `jobCd`와 채용 `jobsCd`는 다른 코드 체계일 수 있습니다. 관리자 매핑의 `jobCodes`에 검토한 채용 직종코드를 명시해야 코드 기반으로 연결합니다. 코드 접두어를 잘라 추정하지 않습니다. 훈련은 게시한 NCS 원천의 코드와 `job_sdvn_cd` 세분류코드를 과정의 `ncsCd`와 정확히 비교합니다. 해당 원천 필드를 선택하지 않으면 없는 연결을 만들지 않습니다.

기존 `occupations`는 기관이 별도로 검증한 외부 직무 모델을 반입하는 형식입니다. 공공 원천의 kind를 `public_api` 또는 `derived`로 지정해 이 경로로 넣어도 게시된 매핑을 우회하여 계산하지 않습니다. 필드 구조는 [구현 계약](architecture.md)을 확인하세요.

## 사내 PostgreSQL / MySQL

서비스 자체 PostgreSQL 계정과 연동용 DB 계정을 분리하세요. 연동용 계정에는 승인된 뷰의 SELECT 권한만 부여하고 테이블 변경·확장·파일·외부 접속 함수 권한을 부여하지 않습니다. SQL 문자열 검사와 읽기 전용 트랜잭션은 DB 권한 설정을 대체하지 않습니다.

```json
{
  "name": "사내 직무 사전",
  "type": "postgres",
  "dataset": "occupations",
  "enabled": true,
  "dsn": "postgres://nextrole_reader:비밀번호@db.internal:5432/hr?sslmode=verify-full",
  "query": "SELECT id, title, description, skills, min_experience AS \"minExperience\", regions, source FROM nextrole_approved_occupations LIMIT 5000",
  "mapping": {}
}
```

MySQL 8 연결 예제: `nextrole_reader:비밀번호@tcp(mysql.internal:3306)/hr?tls=true&parseTime=true`. MariaDB의 실행 제한 변수는 다르므로 이 MySQL 어댑터의 호환 대상으로 간주하지 않습니다. PostgreSQL 전용 인증서가 필요하면 신뢰 저장소 또는 DSN의 TLS 인증서 파일 경로를 운영 환경에서 제공하세요.

SQL은 단일 SELECT 또는 읽기 전용 WITH만 허용합니다. 주석, 여러 명령, 파일 읽기, 잠금, 지연 함수, 쓰기 명령은 차단합니다. 문자열 안의 세미콜론은 허용하며 SQL 인용은 표준 작은따옴표 중복 방식(`'it''s'`)을 사용합니다. 임의 사용자 정의 함수까지 정적 검사로 보장하지 않으므로 관리자가 검토한 뷰 조회를 사용하세요.

## 폐쇄망 운영

외부망 연결이 없는 환경에서 고용24, Google, LinkedIn 등 인터넷 서비스에 직접 연결할 수는 없습니다. 외부 데이터는 승인된 절차로 반입해 내부 HTTP 서버·DB·JSON 가져오기로 공급합니다. 외부 API 키가 없어도 내장된 합성 데이터 모드와 로컬 계산 엔진은 사용할 수 있으며, 결과 화면에 합성 출처가 표시됩니다.

사내 Keycloak과 로컬 OpenAI 호환 AI 서버를 설정하면 인증·AI도 내부망에서 처리할 수 있습니다. 데이터 수집 서버의 외부 접근이 허용되는 환경에서는 관리자 커넥터로 최신 데이터를 가져오고, 수집 시점·출처와 이용 허가 범위를 함께 관리하세요. 등록한 커넥터는 내부 주소 접근 권한을 가지므로 관리자 권한은 신뢰하는 운영자에게만 부여합니다.

## 문제 해결

| 증상 | 확인 방법 |
| --- | --- |
| HTTP 301/302 | 이동한 최종 API URL을 endpoint에 입력 |
| HTTP 401/403 또는 실패 코드 | 키 승인 여부, 인증 헤더/매개변수, 서비스 이용 범위 확인 |
| 목록 경로에서 데이터 없음 | 오류 응답 여부와 `rootPath` 대소문자 확인 |
| 매핑 필드 누락 | 실제 응답에 있는 필드만 매핑. 상수는 `=` 사용 |
| XML UTF-8 오류 | 내부 게이트웨이에서 UTF-8로 변환 |
| DB 연결 실패 | DNS·방화벽·TLS·연동 전용 계정의 SELECT 권한 확인 |
| DB 실행 제한 | 인덱스·뷰 최적화, 기간 조건과 LIMIT 추가 |
| 10 MiB / 5,000건 초과 | 페이지 크기·기간·열 수 축소 |
| 고용24 설정 저장 거부 | 고정 요청 조건, 필수 식별 필드, authKey 전용 입력, 날짜·페이지 범위 확인 |
| 원천은 있는데 목표 직무가 없음 | 초안이 아닌 유효한 게시 매핑이 필요. 원천 변경·삭제 경고 확인 |
| 훈련 연결이 없음 | 검토된 NCS와 job_sdvn_cd 선택, 과정 ncsCd 정확 일치 여부 |
| 실제 공고 역량 빈도가 없음 | 고용24 목록에는 요구기술 항목이 없으므로 정상일 수 있음 |
| 원문 링크가 열리지 않음 | 원본 데이터 URL과 사용자 단말의 네트워크 접근 확인 |

오류 메시지는 원격 응답 본문, 인증키, DSN을 표시하지 않습니다. 자세한 원격 장애 분석은 원본 시스템의 승인된 관리 도구에서 수행합니다.
