#!/usr/bin/env bash
set -euo pipefail

nextrole_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$nextrole_root"
nextrole_version="$(tr -d '\r\n' < VERSION)"
if [[ ! "$nextrole_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo 'VERSION은 1.0.0 형식이어야 합니다.' >&2
  exit 1
fi
if [[ "${1:-}" != "" && "$1" != "v$nextrole_version" ]]; then
  echo '릴리스 태그와 VERSION 파일이 일치하지 않습니다.' >&2
  exit 1
fi
command -v docker >/dev/null
mkdir -p dist
nextrole_image="nextrole:v$nextrole_version"
nextrole_asset="dist/nextrole-v$nextrole_version.tar.gz"
docker build --platform linux/amd64 --tag "$nextrole_image" \
  --label "org.opencontainers.image.version=$nextrole_version" .
bash scripts/smoke-image.sh "$nextrole_image"
# Only the service image is included. Database and other dependency images are
# CI prerequisites and must never be added to the release archive.
docker image save "$nextrole_image" | gzip -n -9 > "$nextrole_asset"
gzip --test "$nextrole_asset"
nextrole_archive_sha="$(sha256sum "$nextrole_asset" | cut -d ' ' -f 1)"
nextrole_image_sha="$(docker image inspect "$nextrole_image" --format '{{.Id}}')"
cat > dist/release-notes.md <<EOF
NextRole v$nextrole_version — AI 경력전환 시뮬레이터

서비스 Docker 이미지 하나를 포함한 오프라인 설치용 압축 파일입니다. 대상은 Linux amd64입니다. PostgreSQL 16+는 기존 내부망 데이터베이스를 사용합니다.

- 이미지: \`$nextrole_image\`
- 첨부 파일: \`nextrole-v$nextrole_version.tar.gz\`
- SHA-256: \`$nextrole_archive_sha\`
- Docker 이미지 ID: \`$nextrole_image_sha\`
- 검증: Go 테스트·프런트엔드 빌드 및 외부 통신이 차단된 Docker 네트워크에서 로그인·프로필·시뮬레이션·정적 리소스 확인

오프라인 서버에서 \`docker load -i nextrole-v$nextrole_version.tar.gz\`로 이미지를 불러옵니다. 저장소의 compose.yaml과 .env.example을 함께 반입하고 네 환경변수를 설정한 후 \`docker compose up -d\`를 실행합니다.

[변경 기록](https://github.com/hkjang/nextRole/blob/v$nextrole_version/CHANGELOG.md) · [설치·운영 가이드](https://github.com/hkjang/nextRole/blob/v$nextrole_version/README.md) · [데이터 연동](https://github.com/hkjang/nextRole/blob/v$nextrole_version/docs/integrations.md)

외부 SSO·AI·고용24 API는 해당 서비스에 접근할 수 있는 망에서만 동작합니다. 폐쇄망에서는 내부 OIDC, 로컬 AI, 반입 데이터를 설정합니다. 인증키가 필요한 외부 사업자와의 실계정 연동은 배포 기관의 설정 후 검증이 필요합니다.

GitHub가 자동 제공하는 소스 코드 아카이브를 제외하면 배포 첨부 파일은 서비스 이미지 tar.gz 한 개입니다.
EOF
echo "완료: $nextrole_asset"
echo "SHA-256: $nextrole_archive_sha"
