#!/usr/bin/env bash
set -euo pipefail

nextrole_image="${1:?이미지 이름을 지정하세요. 예: nextrole:v1.0.0}"
nextrole_suffix="$$-$RANDOM"
nextrole_network="nextrole-smoke-$nextrole_suffix"
nextrole_database="nextrole-pg-$nextrole_suffix"
nextrole_container="nextrole-app-$nextrole_suffix"
nextrole_pg_image='postgres:17-bookworm@sha256:639ab7ceb90e13123085b741fb31ef493fba25463002f6da665352e7b534b652'
cleanup() {
  docker rm -f "$nextrole_container" "$nextrole_database" >/dev/null 2>&1 || true
  docker network rm "$nextrole_network" >/dev/null 2>&1 || true
}
trap cleanup EXIT

# Pull the CI database before creating the network that blocks internet egress.
docker pull "$nextrole_pg_image" >/dev/null
docker network create --internal "$nextrole_network" >/dev/null
nextrole_smoke_password="$(openssl rand -hex 24)"
docker run --detach --name "$nextrole_database" --network "$nextrole_network" \
  --network-alias nextrole-test-db \
  --env POSTGRES_DB=nextrole --env POSTGRES_USER=nextrole \
  --env "POSTGRES_PASSWORD=$nextrole_smoke_password" "$nextrole_pg_image" >/dev/null
for nextrole_attempt in {1..40}; do
  if docker exec "$nextrole_database" pg_isready --username nextrole --dbname nextrole >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
docker exec "$nextrole_database" pg_isready --username nextrole --dbname nextrole >/dev/null

export POSTGRES_DSN="postgres://nextrole:$nextrole_smoke_password@nextrole-test-db:5432/nextrole?sslmode=disable"
export BOOTSTRAP_ADMIN='smoke@example.test'
export BOOTSTRAP_ADMIN_PASSWORD="$nextrole_smoke_password"
export ENCRYPTION_KEY="$(openssl rand -base64 32)"
docker run --detach --name "$nextrole_container" --network "$nextrole_network" \
  --read-only \
  --tmpfs /tmp:size=128m,mode=1777,noexec,nosuid \
  --cap-drop ALL --security-opt no-new-privileges:true \
  --env POSTGRES_DSN --env BOOTSTRAP_ADMIN --env BOOTSTRAP_ADMIN_PASSWORD --env ENCRYPTION_KEY \
  "$nextrole_image" >/dev/null
# Internal Docker networks do not publish host ports on all Engine versions.
# Probe through the service's own loopback interface so internet isolation is
# preserved without relying on host-port forwarding.
for nextrole_attempt in {1..40}; do
  if docker exec "$nextrole_container" curl --fail --silent --max-time 3 http://127.0.0.1:8080/healthz >/dev/null; then
    break
  fi
  sleep 1
done
docker exec "$nextrole_container" curl --fail --silent --max-time 3 http://127.0.0.1:8080/healthz >/dev/null

python3 - "$nextrole_container" <<'PY'
import json, os, re, subprocess, sys
container = sys.argv[1]
base = 'http://127.0.0.1:8080'
def fetch(path, data=None, method=None):
    args = ['docker', 'exec', '-i', container, 'curl', '--fail', '--silent', '--show-error', '--max-time', '10',
            '--cookie', '/tmp/nextrole-smoke-cookies', '--cookie-jar', '/tmp/nextrole-smoke-cookies',
            '--header', 'Content-Type: application/json', '--header', 'Origin: ' + base]
    body = None
    if data is not None:
        body = json.dumps(data).encode()
        args += ['--data-binary', '@-']
    if method:
        args += ['--request', method]
    args += [base + path]
    return subprocess.run(args, input=body, stdout=subprocess.PIPE, check=True).stdout
def request(path, data=None, method=None):
    return json.loads(fetch(path, data, method))
health = request('/healthz')
assert health['status'] == 'ok'
public = request('/api/v1/public')
assert public['version'] == health['version']
request('/api/v1/auth/login', {'email':os.environ['BOOTSTRAP_ADMIN'],'password':os.environ['BOOTSTRAP_ADMIN_PASSWORD']})
assert request('/api/v1/me')['role'] == 'admin'
profile = request('/api/v1/profile/parse', {'text':'Java와 Spring 백엔드 개발 10년, Docker와 Linux 서버 운영 경험'})
assert profile.get('skills'), 'offline career parser returned no skills'
request('/api/v1/profile', profile, 'PUT')
jobs = request('/api/v1/jobs')
assert jobs, 'offline catalog is empty'
sim = request('/api/v1/simulate', {'jobId':jobs[0]['id'],'months':6,'addedSkills':[]})
assert 0 <= sim['score'] <= 100 and sim['factors'], 'invalid simulation'
html = fetch('/simulator').decode()
assert '<html' in html.lower(), 'SPA refresh did not return HTML'
assets = re.findall(r'(?:src|href)=["\'](/[^"\']+)["\']', html)
assert any('/assets/' in path for path in assets), 'frontend bundle missing'
for path in assets:
    data = fetch(path)
    assert data, 'static asset empty'
    if path.endswith('.css'):
        css = data.decode()
        assert 'fonts.googleapis.com' not in css and 'fonts.gstatic.com' not in css, 'external font dependency'
        for font in set(re.findall(r'url\(["\']?(/assets/[^)"\']+)', css)):
            assert fetch(font), 'bundled font missing'
print('오프라인 이미지 검증 통과: 로그인, 경력 파싱, 프로필 저장, 시뮬레이션, 새로고침, 정적 리소스')
PY
test "$(docker network inspect "$nextrole_network" --format '{{.Internal}}')" = true
test "$(docker exec "$nextrole_container" id -u)" = 10001
docker exec "$nextrole_container" pdftotext -v >/dev/null 2>&1
