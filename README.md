# NextRole

**현재의 경력을 평가하는 AI를 넘어, 미래의 경력을 실험하는 경력전환 시뮬레이터.**

NextRole은 경력·보유 역량에서 목표 직무의 부족 역량과 점수 근거를 계산하고, 역량을 추가하거나 목표를 바꾸면서 경로·기간·학습 우선순위를 비교하는 Go + React 서비스입니다. 인터넷 없이 실행할 수 있는 단일 서비스 이미지에 프런트엔드, 한국어 폰트, 이력서 PDF 추출 도구를 포함합니다.

[서비스 소개](https://hkjang.github.io/nextRole/) · [릴리스 다운로드](https://github.com/hkjang/nextRole/releases) · [데이터 연동 가이드](docs/integrations.md)

![NextRole 경력 시뮬레이터 화면](docs/screenshots/simulator.png)

| 문서 | Markdown | HTML | PDF |
| --- | --- | --- | --- |
| 사용자 가이드 | [보기](docs/user-guide.md) | [보기](https://hkjang.github.io/nextRole/user-guide.html) | [다운로드](https://hkjang.github.io/nextRole/user-guide.pdf) |
| 관리자 가이드 | [보기](docs/admin-guide.md) | [보기](https://hkjang.github.io/nextRole/admin-guide.html) | [다운로드](https://hkjang.github.io/nextRole/admin-guide.pdf) |

[전체 화면 캡처](https://hkjang.github.io/nextRole/screenshots/) · [시장 데이터 집계 기준](docs/market-data.md)

[홍보 영상 보기·다운로드](https://hkjang.github.io/nextRole/#video) · [한국어 영상 대본](docs/promo-script.md)

홍보 영상은 실제 서비스 화면과 What-if 조작 장면으로 제작했습니다. 한국어 자막을 포함하며, 영상과 가이드·화면 캡처는 소개 사이트에서 제공합니다. GitHub 릴리스 첨부 파일에는 오프라인 서비스 이미지 `tar.gz`만 포함합니다.

## 제공 기능

- 자연어 경력 파싱, PDF·DOCX·TXT 이력서 텍스트 추출, 역량 수준·신뢰 상태 관리
- 직무 후보 탐색, 2~3개 목표 비교, 부족 역량과 설명 가능한 적합도 계산
- 역량·프로젝트 추가 What-if, 역량별 점수 상승폭, 3·6·12개월 계획과 중간직무 경로
- 실제 채용·훈련 데이터 연동과 출처 표시, 독립적인 합성 데이터 데모 모드
- 유효한 실제 공고의 지역·역량 빈도·공개 급여 표본, 일별 수집 스냅샷 기반 변화 추세
- 관리자 설정과 개인화 분리, 개인 API 키 발급·회전·폐기·권한 변경
- REST API, MCP 도구, AI 스트리밍과 내부 OpenAI 호환 서버 설정
- Google·LinkedIn·Kakao·Naver 프리셋, Keycloak·일반 OIDC/OAuth2 설정
- 선택적 팀장 검토·승인. 기본 설정에서는 승인 메뉴와 절차 비활성
- 한국어 기본 UI, 읽기 쉬운 글자, 모바일 대응, URL 기반 메뉴 복원, 로그인·프로필의 버전 표시

적합도와 예상 기간은 입력과 설정에 따른 의사결정 참고값이며 취업 확률이나 실제 채용 보장이 아닙니다. AI 설명과 별개로 점수는 구조화된 역량·경력·선호 조건을 사용해 계산합니다. 실제 데이터가 연결되지 않은 환경에서는 합성 출처를 명시합니다.

## 기술 구성

| 영역 | 선택 |
| --- | --- |
| 서버 | Go 1.26, 표준 HTTP 서버, PostgreSQL |
| 화면 | React 19, TypeScript, Vite, Mantine 8 |
| 디자인 | 접근성 있는 폼·모달·반응형 레이아웃과 한국어 로컬 폰트 |
| 인증 | 서버 세션, bcrypt, OIDC/OAuth2, 개인 Bearer API 키 |
| 비밀 설정 | PostgreSQL에 AES-256-GCM 암호화 저장 |
| AI | OpenAI 호환 Chat Completions SSE, 최대 출력 설정 262,144토큰 |
| 연동 | JSON/XML/CSV HTTP, PostgreSQL/MySQL 8 읽기 전용 조회, 직접 JSON 반입 |
| 배포 | Linux amd64 서비스 Docker 이미지 + 기존 PostgreSQL |

Mantine을 UI 프레임워크로 선택했습니다. 입력·설정 화면이 많은 서비스에 필요한 접근성, 일관된 컴포넌트, 반응형 구조를 갖추고 있으며 모든 정적 리소스를 번들에 포함할 수 있습니다.

## 오프라인 서버에 설치

필요한 것은 Docker Engine, Docker Compose v2와 접근 가능한 PostgreSQL 16+ 데이터베이스입니다. 2 vCPU·2 GiB 메모리를 시작점으로 잡고 동시 사용자·PDF 크기·데이터 규모에 맞춰 조정하세요. PostgreSQL 이미지와 AI 모델은 서비스 릴리스에 포함하지 않습니다. 해당 시스템은 기존 내부망 자원을 사용합니다.

인터넷에 접근 가능한 환경에서 릴리스의 `nextrole-v1.0.0.tar.gz`와 저장소의 `compose.yaml`, `.env.example`을 받아 승인된 경로로 반입합니다. 릴리스 본문에 있는 SHA-256 값과 반입 파일을 비교합니다.

```bash
sha256sum nextrole-v1.0.0.tar.gz
docker load -i nextrole-v1.0.0.tar.gz
docker image inspect nextrole:v1.0.0
cp .env.example .env
chmod 600 .env
openssl rand -base64 32
```

마지막 명령으로 생성한 값을 `.env`의 `ENCRYPTION_KEY`에 넣습니다. `.env`의 네 값을 운영 환경에 맞게 편집합니다.

| 환경변수 | 값과 용도 |
| --- | --- |
| `POSTGRES_DSN` | 서비스 저장용 PostgreSQL DSN. 예: `postgres://nextrole:비밀번호@postgres.internal:5432/nextrole?sslmode=verify-full` |
| `BOOTSTRAP_ADMIN` | 첫 실행 시 만들 관리자 이메일 |
| `BOOTSTRAP_ADMIN_PASSWORD` | 초기 관리자 비밀번호. 12자 이상 권장 |
| `ENCRYPTION_KEY` | 32바이트 난수를 base64로 인코딩한 암호화 마스터키 |

PostgreSQL에는 NextRole 전용 DB·계정을 준비합니다. 첫 실행 시 애플리케이션이 테이블을 생성하므로 해당 스키마의 생성·읽기·쓰기 권한이 필요합니다. DB TLS 설정에 맞는 DSN을 입력하고, 사설 CA를 쓰는 경우 인증서 파일을 읽기 전용으로 마운트해 DSN의 `sslrootcert`에 경로를 지정할 수 있습니다. DSN 비밀번호에 예약 문자가 포함되면 URL 인코딩하세요.

```bash
docker compose up -d
docker compose ps
curl --fail http://localhost:8080/healthz
```

브라우저에서 `http://서버주소:8080`에 접속해 초기 관리자 계정으로 로그인합니다. 운영 HTTPS는 기관의 리버스 프록시에서 종료하고 관리자 **일반 설정 → 서비스 기본 URL**에 실제 외부 URL을 저장합니다. AI·SSO·키 권한·연동·점수 가중치·승인 흐름은 이후 관리자 화면에서 변경합니다.

컨테이너는 UID/GID 10001로 실행하며 읽기 전용 루트 파일시스템과 임시 `/tmp`를 사용합니다. 영속 데이터는 PostgreSQL에 저장합니다. `compose.yaml`은 `pull_policy: never`로 인터넷 이미지 pull을 시도하지 않습니다.

## 첫 설정 순서

1. 관리자 **일반 설정**에서 서비스 URL, 가입 허용 여부와 합성 데이터 사용 여부를 정합니다.
2. **사용자 관리**에서 사용자·검토자·관리자 역할을 지정합니다.
3. 실제 채용·훈련·직무 데이터가 있으면 **데이터 연동**에서 설정 후 연결 테스트·동기화를 실행합니다. [예제 및 매핑 규칙](docs/integrations.md)을 참고하세요.
4. 내부 AI가 있으면 **AI 설정**에 호환 API 기본 URL, 모델, 키를 입력하고 연결 테스트를 실행합니다. AI가 없어도 오프라인 경력 파싱·시뮬레이션을 사용할 수 있습니다.
5. SSO가 필요하면 **SSO 설정**에서 제공자와 Client ID/Secret을 입력하고 화면의 callback URL을 공급자에 등록합니다.
6. 팀장 검토가 필요한 조직만 승인 절차를 활성화하고 검토자 역할을 배정합니다.

Bootstrap 계정은 사용자 테이블이 비어 있을 때 생성됩니다. 이후 `.env`의 bootstrap 비밀번호를 바꿔도 기존 계정 비밀번호는 재설정되지 않습니다. 로그인 후 개인 설정 또는 사용자 관리에서 변경하세요. `ENCRYPTION_KEY`는 설정 복호화에 계속 필요하므로 PostgreSQL 백업과 별도로 안전하게 보관합니다. 이 키를 임의로 바꾸는 것은 개인 API 키 회전과 다르며 기존 설정을 읽을 수 없게 합니다.

## SSO와 AI

Google·LinkedIn·Kakao·Naver는 기본 엔드포인트와 프로필 필드 프리셋을 적용합니다. 실제 공급자의 애플리케이션 등록, 동의 항목과 OIDC 사용 승인 설정은 해당 공급자 콘솔에서 필요합니다. Keycloak은 realm의 issuer URL과 Client ID/Secret을 입력합니다. 일반 OAuth2는 authorize/token/userinfo URL과 사용자 식별 필드도 설정할 수 있습니다.

서비스가 인터넷에 접근할 수 없는 환경에서는 외부 Google·LinkedIn 등의 로그인은 연결되지 않습니다. 내부 Keycloak 같은 OIDC 서버를 사용하세요. 같은 이메일이라는 이유만으로 기존 계정에 자동 연결하지 않습니다.

AI는 스트리밍으로 요청하고 표시합니다. `maxTokens`는 최대 262,144까지 저장할 수 있지만 실제 출력과 문맥 길이는 모델 서버가 지원하는 범위에 따라 제한됩니다. 관리자 화면에서 모델 문맥창과 토큰 매개변수도 맞추세요. 외부 AI 대신 내부 OpenAI 호환 서버를 지정할 수 있습니다. LLM이 만든 공고나 교육과정을 실제 데이터 목록으로 등록하지 않습니다.

## 개인 API 키와 MCP

**개인 설정 → API 키**에서 필요한 권한만 선택해 키를 발급합니다. 키 원문은 발급·회전 시 한 번만 표시됩니다. 회전하면 이전 키는 즉시 무효화됩니다. 관리자가 허용한 권한 범위와 개인 키의 권한은 매 요청 시 함께 검사합니다.

```bash
# NEXTROLE_API_KEY는 예제 클라이언트 변수이며 서비스 환경변수가 아닙니다.
curl --fail http://localhost:8080/api/v1/jobs \
  -H "Authorization: Bearer $NEXTROLE_API_KEY"
```

REST 명세는 서비스의 `/api/v1/openapi.json`에서 확인합니다. MCP HTTP endpoint는 `/mcp`이며 JSON-RPC `initialize`, `tools/list`, `tools/call`을 사용합니다. MCP 호출에는 `mcp:use`와 해당 도구의 API 권한이 필요합니다.

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2025-03-26",
    "capabilities": {},
    "clientInfo": {"name": "internal-career-client", "version": "1.0.0"}
  }
}
```

## 백업·업데이트·복구

PostgreSQL을 정기 백업하고 실제 복원을 시험하세요. 사용자 프로필·시뮬레이션·설정·키 해시가 DB에 있으며 설정 비밀은 암호화되어 있습니다. 복구에는 백업 DB와 동일한 `ENCRYPTION_KEY`가 함께 필요합니다. `.env`는 저장소에 커밋하지 않습니다.

업데이트 전 DB 백업을 확보하고 새 릴리스의 이미지 tar.gz를 반입합니다. `docker load` 후 `compose.yaml`의 `image` 태그를 새 버전으로 변경하고 `docker compose up -d`를 실행합니다. `/healthz`와 로그인·시뮬레이션을 확인합니다. 스키마 변경이 있는 릴리스는 해당 릴리스의 복구 안내에 따라 DB 백업까지 함께 복원하세요. 이전 이미지 태그만 바꾸는 것으로 모든 스키마 변경이 되돌아가지는 않습니다.

```bash
docker compose logs --tail=100 nextrole
docker compose restart nextrole
docker compose down
```

`down`은 서비스 컨테이너를 중지하며 외부 PostgreSQL 데이터를 지우지 않습니다. 관리자 비밀값은 조회 응답과 원격 연동 오류에 원문으로 표시하지 않습니다. 개인정보를 포함할 수 있는 요청 본문을 리버스 프록시 접근 로그에 기록하지 않도록 기관 정책에 맞춰 설정하세요.

## 개발·검증·릴리스

개발 도구: Go 1.26, Node.js 24, PostgreSQL 16+, PDF 추출을 위한 `pdftotext`. 프런트엔드 의존성은 `web/package-lock.json`, Go 의존성은 `go.mod`/`go.sum`으로 고정합니다.

```bash
cd web
npm ci
npm run build
cd ..
go test ./...
go vet ./...
# .env의 네 값을 개발 셸에 안전하게 설정한 뒤 실행
go run ./cmd/nextrole
```

프런트엔드 개발 서버는 `cd web && npm run dev`로 실행합니다. Vite가 `/api`와 `/mcp`를 Go 서버 8080 포트로 전달합니다. 별도 개발 DB를 사용하고, 서비스 기본 URL을 설정하는 경우 실제 개발 브라우저 주소와 일치시켜 Origin 검사를 통과하도록 합니다.

```bash
# Docker 사용 가능한 빌드 환경에서 이미지 검증과 오프라인 파일 생성
bash scripts/release.sh v1.0.0
```

`VERSION`과 태그가 일치해야 합니다. 스크립트는 Linux amd64 이미지를 만들고 외부 통신이 차단된 Docker 내부 네트워크에서 PostgreSQL과 함께 실행합니다. 로그인·경력 파싱·프로필 저장·시뮬레이션·메뉴 새로고침·정적 리소스를 검사한 뒤 **서비스 이미지 하나만** `dist/nextrole-v1.0.0.tar.gz`로 저장합니다. CI에 쓰는 PostgreSQL 이미지는 배포 파일에 포함하지 않습니다.

GitHub의 `v*` 태그 push는 Go 테스트·정적 검사·이미지 빌드·오프라인 검증을 통과한 뒤 릴리스에 이미지 압축 파일 하나를 첨부합니다. SHA-256은 릴리스 본문에 기록합니다. GitHub 자체가 제공하는 자동 소스 아카이브는 플랫폼 기본 기능입니다.

`docs/`는 GitHub Pages의 정적 사이트입니다. 저장소 Pages 설정에서 GitHub Actions 소스를 선택하면 main 브랜치의 docs 변경을 Pages 워크플로가 게시합니다. 소개 페이지와 가이드는 앱 배포 이미지와 별도로 제공됩니다.
