# NextRole 데이터 연동 가이드

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

데이터 종류는 `occupations`(시뮬레이션 대상 직무), `jobs`(채용공고), `training`(훈련과정)입니다. 채용공고와 직무 사전은 별도 데이터입니다. 채용 API에서 받은 공고 제목을 목표 직무와 역량 수준으로 임의 변환하지 않습니다.

한 번의 테스트·동기화는 설정한 API **한 페이지** 또는 SQL **한 조회**를 읽습니다. 전체 페이지 자동 순회나 주기 스케줄러는 포함하지 않습니다. 고용24의 `startPage` / `pageNum` 같은 조회 매개변수를 변경해 필요한 페이지를 가져오거나, 사내 수집 게이트웨이에서 페이지 순회·증분 처리를 수행하세요.

## 공통 설정

| 항목 | 의미 |
| --- | --- |
| `name` | 사용자에게 표시할 데이터 출처 이름 |
| `type` | `json`, `xml`, `csv`, `postgres`, `mysql` |
| `dataset` | `jobs`, `training`, `occupations` |
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

출처가 없으면 커넥터 이름과 query를 제거한 endpoint 주소를 보완합니다. `source.synthetic`을 생략하면 실제 연동 데이터(`false`)로 표시합니다. 교육·채용 원문 링크는 원본에서 받아야 하며, 존재하지 않는 링크를 생성하지 않습니다.

## 고용24 채용정보

고용24 OPEN-API는 신청·승인된 인증키가 필요합니다. 현재 공식 안내는 기업회원 대상 신청 절차를 설명하고 있습니다. 계정별 이용 범위는 발급 시 확인하세요. [고용24 OPEN-API 이용안내](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiIntro.do)

다음 예제는 공식 채용 목록의 XML 구조를 NextRole 채용공고로 매핑합니다. 공고 원문 URL과 구인인증번호를 보존합니다. `skills`는 목록 API가 제공하지 않는 항목이므로 빈 배열이며, 실제 요구기술 분석은 별도의 승인된 상세 데이터와 연결할 수 있습니다. 공식 규격 확인일: 2026-09-27. [고용24 채용정보 API 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId=000000000000000000000000000000)

```json
{
  "name": "고용24 채용정보",
  "type": "xml",
  "dataset": "jobs",
  "enabled": true,
  "endpoint": "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do",
  "apiKey": "관리자 화면에서 발급된 인증키 입력",
  "apiKeyParam": "authKey",
  "params": {
    "callTp": "L", "returnType": "XML", "startPage": "1",
    "display": "100", "keyword": "플랫폼 개발"
  },
  "rootPath": "wantedRoot.wanted",
  "mapping": {
    "id": "wantedAuthNo", "title": "title", "organization": "company",
    "region": "region", "url": "wantedInfoUrl", "salary": "sal",
    "deadline": "closeDt", "description": "title", "skills": "=[]",
    "source.name": "=고용24 채용정보", "source.synthetic": "=false"
  }
}
```

## 고용24 국민내일배움카드 훈련

훈련 목록은 `HRDNet.srchList.scn_list`에서 읽습니다. 기간은 훈련 시작일 기준이며 예제 날짜를 운영 시점에 맞게 수정합니다. 과정 ID와 회차를 합친 고유 `id`로 같은 과정의 여러 회차를 각각 보존합니다. [고용24 국민내일배움카드 훈련 API 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId=000000000000000000000000000004)

```json
{
  "name": "고용24 국민내일배움카드",
  "type": "xml",
  "dataset": "training",
  "enabled": true,
  "endpoint": "https://www.work24.go.kr/cm/openApi/call/hr/callOpenApiSvcInfo310L01.do",
  "apiKey": "관리자 화면에서 발급된 인증키 입력",
  "apiKeyParam": "authKey",
  "params": {
    "returnType": "XML", "outType": "1", "pageNum": "1", "pageSize": "100",
    "srchTraStDt": "20260927", "srchTraEndDt": "20261227", "sort": "ASC", "sortCol": "2"
  },
  "rootPath": "HRDNet.srchList.scn_list",
  "mapping": {
    "id": "concat:trprId,trprDegr", "title": "title", "organization": "subTitle", "region": "address",
    "url": "titleLink", "description": "contents", "skills": "=[]",
    "source.name": "=고용24 국민내일배움카드", "source.synthetic": "=false"
  }
}
```

수강비와 본인부담금은 서로 다른 값일 수 있으므로 위 예제는 비용을 추정해 넣지 않습니다. 사용자의 부담 비용을 확인한 별도 데이터가 있을 때 `cost`를 매핑하세요.

## 직무·NCS 데이터

표준직무기술서 API의 공식 요청 주소는 `https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo215L01.do`이며, `authKey`, `jobCont`와 선택 `limit`, JSON 결과 설정을 사용합니다. 이 응답의 NCS 능력단위를 NextRole 역량 수준과 동일한 값으로 해석해서는 안 됩니다. 담당자가 승인한 직무·역량·수준 매핑을 거친 JSON/CSV/DB 뷰를 `occupations` 데이터로 가져오는 방식을 권장합니다. [고용24 표준직무기술서 API 명세](https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?apiSvcId=000000000000000000000000000087&fullApiSvcId=000000000000000000000000000096%5E000000000000000000000000000084&upprApiSvcId=000000000000000000000000000086)

정규화된 직무 예제:

```json
{
  "id": "internal-platform-engineer",
  "title": "플랫폼 엔지니어",
  "category": "소프트웨어",
  "description": "기관에서 승인한 직무 정의와 역량 요구사항",
  "skills": [{"name":"Go","level":4,"weight":2},{"name":"Linux","level":3,"weight":1}],
  "minExperience": 3,
  "domain": "IT",
  "education": "무관",
  "regions": ["서울", "원격"],
  "bridgeIds": [],
  "source": {"name":"사내 직무위원회 2026-09 승인본","url":"","synthetic":false}
}
```

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
| 원문 링크가 열리지 않음 | 원본 데이터 URL과 사용자 단말의 네트워크 접근 확인 |

오류 메시지는 원격 응답 본문, 인증키, DSN을 표시하지 않습니다. 자세한 원격 장애 분석은 원본 시스템의 승인된 관리 도구에서 수행합니다.
