import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Code,
  Modal,
  MultiSelect,
  NumberInput,
  PasswordInput,
  Select,
  Switch,
  Table,
  Textarea,
  TextInput,
} from "@mantine/core";
import {
  IconActivity,
  IconArrowUpRight,
  IconCheck,
  IconCloudDownload,
  IconDatabase,
  IconDownload,
  IconEdit,
  IconKey,
  IconLink,
  IconPlus,
  IconRefresh,
  IconSettings,
  IconShieldCheck,
  IconSparkles,
  IconTrash,
  IconUpload,
  IconUsers,
} from "@tabler/icons-react";
import { api } from "./api";
import {
  Empty,
  Loading,
  number,
  PageTitle,
  Panel,
  useApp,
  useResource,
} from "./App";
export function AdminOverview() {
  const status = useResource<any>("/admin/status"),
    settings = useResource<any>("/admin/settings");
  const s = status.data;
  return (
    <>
      <PageTitle
        eyebrow="SERVICE ADMINISTRATION"
        title="NextRole 운영 현황"
        description="서비스 상태부터 사용자와 데이터 연결까지, 한곳에서 관리하세요."
        action={
          <Button
            variant="default"
            leftSection={<IconRefresh size={18} />}
            onClick={() => status.reload()}
          >
            새로고침
          </Button>
        }
      />
      <Loading loading={status.loading} error={status.error} />
      {s && (
        <>
          <div className="admin-health">
            <div className="health-icon">
              <IconShieldCheck size={32} />
            </div>
            <div>
              <h2>서비스 운영 상태</h2>
              <p>
                PostgreSQL ·{" "}
                {s.database === "ok" || s.database === "connected"
                  ? "정상 연결"
                  : String(s.database)}{" "}
                · NextRole v{s.version}
              </p>
            </div>
            <Badge
              color={
                s.database === "ok" || s.database === "connected"
                  ? "teal"
                  : "orange"
              }
              size="lg"
            >
              {s.mode === "offline"
                ? "오프라인"
                : s.mode === "connected"
                  ? "AI 연결"
                  : "서비스 운영 중"}
            </Badge>
          </div>
          <div className="stat-grid">
            <div className="stat-card">
              <span className="stat-icon mint">
                <IconUsers />
              </span>
              <div>
                <span>전체 사용자</span>
                <strong>
                  {number(s.users)}
                  <small>명</small>
                </strong>
              </div>
            </div>
            <div className="stat-card">
              <span className="stat-icon lavender">
                <IconActivity />
              </span>
              <div>
                <span>저장한 시뮬레이션</span>
                <strong>
                  {number(s.simulations)}
                  <small>개</small>
                </strong>
              </div>
            </div>
            <div className="stat-card">
              <span className="stat-icon peach">
                <IconDatabase />
              </span>
              <div>
                <span>데이터 연결</span>
                <strong>
                  {number(s.connectors)}
                  <small>개</small>
                </strong>
              </div>
            </div>
          </div>
        </>
      )}
      <div className="section-heading">
        <h2>서비스 운영 설정</h2>
      </div>
      <div className="admin-grid">
        {[
          {
            to: "general",
            icon: IconSettings,
            title: "서비스 기본 설정",
            desc: "서비스 주소, 회원가입, 합성 데이터와 선택적 검토 프로세스",
          },
          {
            to: "ai",
            icon: IconSparkles,
            title: "AI 모델 연결",
            desc: "내부망 모델 또는 호환 API, 스트리밍과 최대 토큰",
          },
          {
            to: "providers",
            icon: IconKey,
            title: "SSO 로그인",
            desc: "Google · LinkedIn · Kakao · Naver · Keycloak · OIDC",
          },
          {
            to: "connectors",
            icon: IconDatabase,
            title: "데이터 연동",
            desc: "JSON · XML · CSV · PostgreSQL · MySQL 커넥터",
          },
          {
            to: "security",
            icon: IconShieldCheck,
            title: "보안과 개인 키 정책",
            desc: "세션 유지 시간, 키 유효기간과 허용 권한",
          },
          {
            to: "users",
            icon: IconUsers,
            title: "사용자와 역할",
            desc: "서비스 관리자 · 사용자 · 검토자 권한 관리",
          },
        ].map((item) => (
          <Link to={`/admin/${item.to}`} className="admin-card" key={item.to}>
            <span className="admin-card-icon">
              <item.icon size={25} />
            </span>
            <h3>{item.title}</h3>
            <p>{item.desc}</p>
            <IconArrowUpRight className="admin-card-arrow" size={21} />
          </Link>
        ))}
      </div>
      <Panel title="현재 서비스 정책">
        <div className="policy-grid">
          <div>
            <span>회원가입</span>
            <Badge
              color={
                settings.data?.general?.registrationEnabled ? "teal" : "gray"
              }
            >
              {settings.data?.general?.registrationEnabled
                ? "허용"
                : "관리자 초대"}
            </Badge>
          </div>
          <div>
            <span>팀장 검토 · 승인</span>
            <Badge
              color={settings.data?.general?.approvalEnabled ? "teal" : "gray"}
            >
              {settings.data?.general?.approvalEnabled ? "사용" : "사용 안 함"}
            </Badge>
          </div>
          <div>
            <span>AI 모델</span>
            <Badge color={settings.data?.ai?.enabled ? "teal" : "gray"}>
              {settings.data?.ai?.enabled ? "AI 연결 사용" : "오프라인 엔진"}
            </Badge>
          </div>
          <div>
            <span>합성 데이터</span>
            <Badge
              color={settings.data?.general?.demoEnabled ? "orange" : "gray"}
            >
              {settings.data?.general?.demoEnabled ? "데모 사용" : "사용 안 함"}
            </Badge>
          </div>
        </div>
      </Panel>
    </>
  );
}
const settingTitles: any = {
  general: ["서비스 설정", "조직의 운영 방식에 맞게 서비스를 구성하세요."],
  ai: ["AI 모델 설정", "오프라인 내부망 모델과 OpenAI 호환 API를 연결하세요."],
  security: [
    "보안 · 키 정책",
    "세션과 개인 키의 유효기간, 허용 권한을 관리하세요.",
  ],
  scoring: [
    "설명 가능한 점수 모델",
    "여섯 가지 요소의 가중치를 조직의 기준에 맞게 설정하세요.",
  ],
};
export function SettingsPage({ section }: { section: string }) {
  const resource = useResource<any>("/admin/settings"),
    { notify, setPublicInfo } = useApp(),
    [settings, setSettings] = useState<any>(null),
    [busy, setBusy] = useState("");
  useEffect(() => {
    if (resource.data) setSettings(resource.data);
  }, [resource.data, section]);
  const set = (k: string, v: any) =>
    setSettings((s: any) => ({ ...s, [section]: { ...s[section], [k]: v } }));
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy("save");
    try {
      const result = await api("/admin/settings", "PUT", settings);
      setSettings(result);
      const p = await api("/public");
      setPublicInfo(p);
      notify("서비스 설정을 저장했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function test() {
    setBusy("test");
    try {
      const r = await api("/admin/ai/test", "POST");
      notify(r.message || "AI 연결이 정상입니다.", r.ok ? "success" : "error");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  const current = settings?.[section];
  const sum = Object.values(settings?.scoring || {}).reduce<number>(
    (a, v) => a + Number(v),
    0,
  );
  const scopeData = [
    "profile:read",
    "profile:write",
    "jobs:read",
    "simulate:write",
    "ai:use",
    "mcp:use",
    ...(current?.allowedKeyScopes || []),
  ].filter((v, i, a) => a.indexOf(v) === i);
  return (
    <>
      <PageTitle
        eyebrow="SERVICE SETTINGS"
        title={settingTitles[section][0]}
        description={settingTitles[section][1]}
      />
      <Loading loading={resource.loading} error={resource.error} />
      {current && (
        <form onSubmit={save}>
          <div className="settings-layout">
            <Panel title="운영 설정">
              {section === "general" && (
                <div className="form-stack">
                  <TextInput
                    required
                    label="서비스 이름"
                    value={current.name || ""}
                    onChange={(e) => set("name", e.target.value)}
                  />
                  <TextInput
                    label="서비스 외부 주소"
                    description="SSO 리디렉션 URI와 외부 접근에 사용하는 주소입니다."
                    placeholder="https://nextrole.example.com"
                    value={current.baseUrl || ""}
                    onChange={(e) => set("baseUrl", e.target.value)}
                  />
                  <div className="setting-toggle">
                    <div>
                      <strong>비밀번호 회원가입 허용</strong>
                      <p>
                        로그인 화면에서 이메일과 비밀번호로 계정을 만들 수
                        있습니다. SSO 계정 생성은 별도로 동작합니다.
                      </p>
                    </div>
                    <Switch
                      aria-label="회원가입 허용"
                      checked={!!current.registrationEnabled}
                      onChange={(e) =>
                        set("registrationEnabled", e.target.checked)
                      }
                    />
                  </div>
                  <div className="setting-toggle">
                    <div>
                      <strong>합성 데이터 모드</strong>
                      <p>
                        데모 직무·훈련·채용 예시를 제공합니다. 합성 데이터는
                        화면에 표시됩니다.
                      </p>
                    </div>
                    <Switch
                      aria-label="합성 데이터 모드"
                      checked={!!current.demoEnabled}
                      onChange={(e) => set("demoEnabled", e.target.checked)}
                    />
                  </div>
                  <div className="setting-toggle">
                    <div>
                      <strong>팀장 검토 · 승인 프로세스</strong>
                      <p>활성화하면 검토 요청과 승인·반려 메뉴를 제공합니다.</p>
                    </div>
                    <Switch
                      aria-label="팀장 검토 승인 프로세스"
                      checked={!!current.approvalEnabled}
                      onChange={(e) => set("approvalEnabled", e.target.checked)}
                    />
                  </div>
                </div>
              )}
              {section === "ai" && (
                <div className="form-stack">
                  <Switch
                    label="AI 모델 연결 사용"
                    checked={!!current.enabled}
                    onChange={(e) => set("enabled", e.target.checked)}
                  />
                  <TextInput
                    label="API 기본 주소"
                    placeholder="http://model.internal:8000/v1"
                    description="OpenAI 호환 서버의 /v1 주소를 입력하세요."
                    value={current.baseUrl || ""}
                    onChange={(e) => set("baseUrl", e.target.value)}
                  />
                  <TextInput
                    label="모델 이름"
                    placeholder="예: 사내 모델 이름"
                    value={current.model || ""}
                    onChange={(e) => set("model", e.target.value)}
                  />
                  <PasswordInput
                    label="API 키"
                    placeholder={
                      current.hasSecret
                        ? "저장된 키 사용 · 변경할 때만 입력"
                        : "API 키를 입력하세요"
                    }
                    description="비워 두면 저장된 키를 유지합니다."
                    value={current.apiKey || ""}
                    onChange={(e) => set("apiKey", e.target.value)}
                    autoComplete="new-password"
                  />
                  <div className="form-grid">
                    <NumberInput
                      label="최대 출력 토큰"
                      description="상한 262,144 (256K). 모델의 한도를 확인하세요."
                      min={1}
                      max={262144}
                      value={current.maxTokens || 4096}
                      onChange={(v) => set("maxTokens", Number(v))}
                    />
                    <NumberInput
                      label="AI 요청 제한 시간 (초)"
                      description="긴 스트리밍 응답을 위해 최대 14,400초까지 설정합니다."
                      min={30}
                      max={14400}
                      value={current.timeoutSeconds || 3600}
                      onChange={(v) => set("timeoutSeconds", Number(v))}
                    />
                    <NumberInput
                      label="컨텍스트 윈도우"
                      description="입력과 출력을 합한 모델의 전체 토큰 한도"
                      min={1024}
                      max={2097152}
                      value={current.contextWindow || 262144}
                      onChange={(v) => set("contextWindow", Number(v))}
                    />
                    <Select
                      label="출력 토큰 매개변수"
                      value={current.tokenParameter || "max_tokens"}
                      data={[
                        { value: "max_tokens", label: "max_tokens (기본)" },
                        {
                          value: "max_completion_tokens",
                          label: "max_completion_tokens",
                        },
                      ]}
                      onChange={(v) => set("tokenParameter", v)}
                    />
                    <Switch
                      label="Temperature 매개변수 전달"
                      checked={current.sendTemperature !== false}
                      onChange={(e) => set("sendTemperature", e.target.checked)}
                    />
                    <NumberInput
                      label="Temperature"
                      disabled={current.sendTemperature === false}
                      min={0}
                      max={2}
                      step={0.1}
                      decimalScale={2}
                      value={current.temperature ?? 0.3}
                      onChange={(v) => set("temperature", Number(v))}
                    />
                  </div>
                  <Alert color="teal" icon={<IconSparkles />}>
                    AI 호출은 기본으로 스트리밍 처리합니다. 연결을 사용하지
                    않으면 오프라인 규칙 엔진으로 분석과 시뮬레이션을
                    제공합니다.
                  </Alert>
                  <Button
                    variant="light"
                    loading={busy === "test"}
                    onClick={test}
                    leftSection={<IconLink size={18} />}
                  >
                    저장된 설정으로 연결 테스트
                  </Button>
                </div>
              )}
              {section === "security" && (
                <div className="form-stack">
                  <NumberInput
                    required
                    label="세션 유지 시간"
                    suffix="시간"
                    min={1}
                    max={168}
                    value={current.sessionHours}
                    onChange={(v) => set("sessionHours", Number(v))}
                  />
                  <NumberInput
                    required
                    label="개인 API 키 최대 유효기간"
                    suffix="일"
                    min={1}
                    max={365}
                    value={current.keyMaxDays}
                    onChange={(v) => set("keyMaxDays", Number(v))}
                  />
                  <MultiSelect
                    label="개인 키에 허용할 권한"
                    description="사용자는 이 목록 안에서 자신의 API 키 권한을 선택합니다."
                    data={scopeData}
                    value={current.allowedKeyScopes || []}
                    onChange={(v) => set("allowedKeyScopes", v)}
                  />
                  <Alert color="yellow">
                    허용 권한을 줄이면 기존 키를 사용하는 외부 연결에 영향을 줄
                    수 있습니다.
                  </Alert>
                </div>
              )}
              {section === "scoring" && (
                <div className="form-stack">
                  <Alert
                    color={Math.abs(sum - 100) < 0.01 ? "teal" : "yellow"}
                    title={`가중치 합계 ${sum}%`}
                  >
                    가중치 합계를 100%로 설정하세요. 점수는 채용 확률이 아닌
                    역량 기반의 의사결정 참고값입니다.
                  </Alert>
                  <div className="form-grid">
                    {[
                      { k: "skill", n: "기술 역량" },
                      { k: "transfer", n: "전이 가능 역량" },
                      { k: "experience", n: "경력 경험" },
                      { k: "domain", n: "산업 · 도메인" },
                      { k: "education", n: "학력 · 자격" },
                      { k: "preference", n: "개인 선호" },
                    ].map((f) => (
                      <NumberInput
                        key={f.k}
                        label={f.n}
                        min={0}
                        max={100}
                        suffix="%"
                        value={current[f.k] || 0}
                        onChange={(v) => set(f.k, Number(v))}
                      />
                    ))}
                  </div>
                </div>
              )}
              <div className="form-actions">
                <Button
                  type="submit"
                  loading={busy === "save"}
                  disabled={section === "scoring" && Math.abs(sum - 100) > 0.01}
                  leftSection={<IconCheck size={18} />}
                >
                  설정 저장
                </Button>
              </div>
            </Panel>
            <aside>
              <Panel className="settings-help" title="설정 가이드">
                <IconSettings size={31} />
                {section === "general" ? (
                  <>
                    <p>
                      모든 관리 설정은 이 화면에서 저장하며, 재시작 후에도
                      유지됩니다.
                    </p>
                    <p>
                      검토 프로세스를 사용하려면 검토자 또는 관리자 역할을
                      지정하세요. 자신의 요청은 직접 승인할 수 없습니다.
                    </p>
                  </>
                ) : section === "ai" ? (
                  <>
                    <p>
                      오프라인 환경에서는 내부망에서 접근 가능한 모델 서버
                      주소를 사용하세요.
                    </p>
                    <p>
                      최대 출력 토큰과 컨텍스트 윈도우는 서로 다릅니다. 입력
                      프롬프트가 들어갈 여유를 남겨야 합니다. 모델이 지원하는
                      한도에 맞춰 설정하세요.
                    </p>
                    <p>
                      변경 내용을 먼저 저장한 다음 연결 테스트를 실행하세요.
                    </p>
                  </>
                ) : section === "security" ? (
                  <>
                    <p>
                      개인 API 키는 사용자별로 분리됩니다. 필요한 범위의 권한만
                      허용하고 정기적으로 회전하세요.
                    </p>
                    <p>
                      키 원문은 한 번만 표시되며 서버에 원문 그대로 저장되지
                      않습니다.
                    </p>
                  </>
                ) : (
                  <>
                    <p>각 점수의 근거는 시뮬레이터에 함께 표시됩니다.</p>
                    <p>
                      설정을 변경하면 이후 계산되는 직무 추천과 시뮬레이션에
                      적용됩니다.
                    </p>
                  </>
                )}
              </Panel>
            </aside>
          </div>
        </form>
      )}
    </>
  );
}
export function UsersPage() {
  const resource = useResource<any[]>("/admin/users"),
    { notify, user } = useApp(),
    [editing, setEditing] = useState<any>(null),
    [busy, setBusy] = useState(false),
    [search, setSearch] = useState("");
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api(
        editing.id ? `/admin/users/${editing.id}` : "/admin/users",
        editing.id ? "PUT" : "POST",
        editing,
      );
      setEditing(null);
      notify("사용자 설정을 저장했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        title="사용자와 역할 관리"
        description="함께 사용하는 계정과 서비스 접근 권한을 관리하세요."
        action={
          <Button
            leftSection={<IconPlus size={18} />}
            onClick={() =>
              setEditing({
                email: "",
                name: "",
                password: "",
                role: "user",
                disabled: false,
              })
            }
          >
            사용자 추가
          </Button>
        }
      />
      <Panel>
        <TextInput
          placeholder="이름 또는 이메일 검색"
          aria-label="사용자 검색"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Loading loading={resource.loading} error={resource.error} />
        <div className="table-scroll">
          <Table verticalSpacing="lg">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>사용자</Table.Th>
                <Table.Th>이메일</Table.Th>
                <Table.Th>역할</Table.Th>
                <Table.Th>상태</Table.Th>
                <Table.Th>관리</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {resource.data
                ?.filter((u) =>
                  `${u.name} ${u.email}`
                    .toLowerCase()
                    .includes(search.toLowerCase()),
                )
                .map((u) => (
                  <Table.Tr key={u.id}>
                    <Table.Td>
                      <strong>{u.name}</strong>
                      {u.id === user.id && (
                        <Badge ml="sm" variant="light" size="xs">
                          나
                        </Badge>
                      )}
                    </Table.Td>
                    <Table.Td>{u.email}</Table.Td>
                    <Table.Td>
                      {
                        (
                          {
                            admin: "서비스 관리자",
                            user: "사용자",
                            reviewer: "검토자",
                          } as any
                        )[u.role]
                      }
                    </Table.Td>
                    <Table.Td>
                      <Badge color={u.disabled ? "gray" : "teal"}>
                        {u.disabled ? "비활성" : "활성"}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <Button
                        size="xs"
                        variant="light"
                        onClick={() => setEditing({ ...u, password: "" })}
                        leftSection={<IconEdit size={15} />}
                      >
                        수정
                      </Button>
                    </Table.Td>
                  </Table.Tr>
                ))}
            </Table.Tbody>
          </Table>
        </div>
      </Panel>
      <Modal
        opened={!!editing}
        onClose={() => setEditing(null)}
        title={editing?.id ? "사용자 수정" : "사용자 추가"}
        centered
      >
        {editing && (
          <form onSubmit={save} className="form-stack">
            <TextInput
              required
              label="이메일"
              type="email"
              readOnly={!!editing.id}
              value={editing.email}
              onChange={(e) =>
                setEditing({ ...editing, email: e.target.value })
              }
            />
            <TextInput
              required
              label="이름"
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
            />
            <PasswordInput
              required={!editing.id}
              minLength={12}
              label={editing.id ? "비밀번호 재설정 (선택)" : "초기 비밀번호"}
              description="12자 이상. 비워 두면 기존 비밀번호를 유지합니다."
              value={editing.password}
              onChange={(e) =>
                setEditing({ ...editing, password: e.target.value })
              }
            />
            <Select
              label="역할"
              data={[
                { value: "user", label: "사용자" },
                { value: "reviewer", label: "검토자" },
                { value: "admin", label: "서비스 관리자" },
              ]}
              value={editing.role}
              onChange={(v) => setEditing({ ...editing, role: v })}
            />
            <Switch
              label="계정 비활성화"
              checked={!!editing.disabled}
              onChange={(e) =>
                setEditing({ ...editing, disabled: e.target.checked })
              }
            />
            <Button type="submit" loading={busy}>
              사용자 저장
            </Button>
          </form>
        )}
      </Modal>
    </>
  );
}
const presets: any = {
  google: {
    name: "Google",
    issuer: "https://accounts.google.com",
    scopes: "openid email profile",
    subjectField: "sub",
    emailField: "email",
    nameField: "name",
  },
  linkedin: {
    name: "LinkedIn",
    issuer: "https://www.linkedin.com",
    scopes: "openid profile email",
    subjectField: "sub",
    emailField: "email",
    nameField: "name",
  },
  kakao: {
    name: "카카오",
    issuer: "https://kauth.kakao.com",
    scopes: "openid profile_nickname account_email",
    subjectField: "sub",
    emailField: "email",
    nameField: "nickname",
  },
  naver: {
    name: "네이버",
    authorizationUrl: "https://nid.naver.com/oauth2.0/authorize",
    tokenUrl: "https://nid.naver.com/oauth2.0/token",
    userInfoUrl: "https://openapi.naver.com/v1/nid/me",
    scopes: "",
    subjectField: "response.id",
    emailField: "response.email",
    nameField: "response.name",
  },
  keycloak: {
    name: "Keycloak",
    issuer: "",
    scopes: "openid profile email",
    subjectField: "sub",
    emailField: "email",
    nameField: "name",
  },
  oidc: {
    name: "OIDC",
    issuer: "",
    scopes: "openid profile email",
    subjectField: "sub",
    emailField: "email",
    nameField: "name",
  },
  oauth2: {
    name: "OAuth 2.0",
    scopes: "",
    subjectField: "id",
    emailField: "email",
    nameField: "name",
  },
};
export function ProvidersPage() {
  const resource = useResource<any[]>("/admin/providers"),
    settings = useResource<any>("/admin/settings"),
    { notify } = useApp(),
    [editing, setEditing] = useState<any>(null),
    [remove, setRemove] = useState<any>(null),
    [busy, setBusy] = useState(false);
  const base = settings.data?.general?.baseUrl || window.location.origin;
  function preset(type: string) {
    setEditing((e: any) => ({
      ...e,
      type,
      ...presets[type],
      clientId: e?.clientId || "",
      clientSecret: "",
      enabled: e?.enabled ?? false,
      authorizationUrl: presets[type].authorizationUrl || "",
      tokenUrl: presets[type].tokenUrl || "",
      userInfoUrl: presets[type].userInfoUrl || "",
      issuer: presets[type].issuer || "",
    }));
  }
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    try {
      await api(
        editing.id ? `/admin/providers/${editing.id}` : "/admin/providers",
        editing.id ? "PUT" : "POST",
        editing,
      );
      setEditing(null);
      notify("SSO 연결 설정을 저장했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  async function del() {
    try {
      await api(`/admin/providers/${remove.id}`, "DELETE");
      setRemove(null);
      void resource.reload();
      notify("SSO 연결을 삭제했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageTitle
        title="간편하게, 안전하게 로그인"
        description="제공자를 선택하고 Client ID와 Secret을 입력하면 표준 인증 설정을 자동으로 적용합니다."
        action={
          <Button
            leftSection={<IconPlus size={18} />}
            onClick={() => {
              setEditing({
                type: "google",
                ...presets.google,
                clientId: "",
                clientSecret: "",
                enabled: false,
              });
            }}
          >
            SSO 연결 추가
          </Button>
        }
      />
      <div className="provider-brands">
        {[
          "Google",
          "LinkedIn",
          "Kakao",
          "Naver",
          "Keycloak",
          "OIDC / OAuth 2.0",
        ].map((x) => (
          <span key={x}>{x}</span>
        ))}
      </div>
      <Alert color="teal" title="제공자 관리 콘솔의 설정도 확인하세요">
        로그인 제공자에 NextRole의 리디렉션 URI를 등록하세요. 제공자별 동의
        항목과 앱 검수가 필요할 수 있으며, OIDC는 발급자 주소에서 메타데이터를
        자동 탐색합니다.
      </Alert>
      <Loading loading={resource.loading} error={resource.error} />
      {resource.data?.length === 0 ? (
        <Empty
          title="아직 연결된 로그인 제공자가 없습니다"
          description="원하는 제공자를 선택해 첫 번째 SSO 연결을 추가하세요."
        />
      ) : (
        <div className="admin-grid">
          {resource.data?.map((p) => (
            <Panel key={p.id}>
              <div className="panel-title">
                <span className="admin-card-icon">
                  <IconKey size={25} />
                </span>
                <Badge color={p.enabled ? "teal" : "gray"}>
                  {p.enabled ? "활성" : "비활성"}
                </Badge>
              </div>
              <h2>{p.name}</h2>
              <p className="muted">
                {p.type.toUpperCase()} ·{" "}
                {p.hasSecret ? "Secret 저장됨" : "Secret 미설정"}
              </p>
              <label className="field-label">리디렉션 URI</label>
              <Code block>
                {base.replace(/\/$/, "")}/api/v1/auth/sso/{p.id}/callback
              </Code>
              <div className="button-row">
                <Button
                  variant="light"
                  onClick={() => setEditing({ ...p, clientSecret: "" })}
                >
                  연결 수정
                </Button>
                <Button
                  color="red"
                  variant="subtle"
                  onClick={() => setRemove(p)}
                >
                  삭제
                </Button>
              </div>
            </Panel>
          ))}
        </div>
      )}
      <Modal
        opened={!!editing}
        onClose={() => setEditing(null)}
        title={editing?.id ? "SSO 연결 수정" : "SSO 연결 추가"}
        size="lg"
        centered
      >
        {editing && (
          <form className="form-stack" onSubmit={save}>
            <Select
              label="제공자 유형"
              value={editing.type}
              onChange={(v) => v && preset(v)}
              data={Object.keys(presets).map((value) => ({
                value,
                label: presets[value].name,
              }))}
            />
            <div className="form-grid">
              <TextInput
                required
                label="로그인 버튼 이름"
                value={editing.name}
                onChange={(e) =>
                  setEditing({ ...editing, name: e.target.value })
                }
              />
              <Switch
                mt="xl"
                label="연결 활성화"
                checked={!!editing.enabled}
                onChange={(e) =>
                  setEditing({ ...editing, enabled: e.target.checked })
                }
              />
            </div>
            <TextInput
              required
              label="Client ID"
              value={editing.clientId || ""}
              onChange={(e) =>
                setEditing({ ...editing, clientId: e.target.value })
              }
            />
            <PasswordInput
              label="Client Secret"
              placeholder={editing.hasSecret ? "저장됨 · 변경할 때만 입력" : ""}
              description="비워 두면 기존 Secret을 유지합니다."
              value={editing.clientSecret || ""}
              onChange={(e) =>
                setEditing({ ...editing, clientSecret: e.target.value })
              }
              autoComplete="new-password"
            />
            {!["naver", "oauth2"].includes(editing.type) && (
              <TextInput
                label="Issuer · 발급자 주소"
                description={
                  editing.type === "keycloak"
                    ? "예: https://sso.example.com/realms/my-realm"
                    : "표준 OIDC 메타데이터를 자동으로 탐색합니다."
                }
                value={editing.issuer || ""}
                onChange={(e) =>
                  setEditing({ ...editing, issuer: e.target.value })
                }
              />
            )}
            <TextInput
              label="요청 Scope (공백으로 구분)"
              value={editing.scopes || ""}
              onChange={(e) =>
                setEditing({ ...editing, scopes: e.target.value })
              }
            />
            <details>
              <summary>고급 설정 · 사용자 필드 및 엔드포인트</summary>
              <div className="form-stack">
                <TextInput
                  label="Authorization URL"
                  value={editing.authorizationUrl || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, authorizationUrl: e.target.value })
                  }
                />
                <TextInput
                  label="Token URL"
                  value={editing.tokenUrl || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, tokenUrl: e.target.value })
                  }
                />
                <TextInput
                  label="UserInfo URL"
                  value={editing.userInfoUrl || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, userInfoUrl: e.target.value })
                  }
                />
                <TextInput
                  label="사용자 식별자 필드"
                  value={editing.subjectField || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, subjectField: e.target.value })
                  }
                />
                <TextInput
                  label="이메일 필드"
                  value={editing.emailField || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, emailField: e.target.value })
                  }
                />
                <TextInput
                  label="이름 필드"
                  value={editing.nameField || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, nameField: e.target.value })
                  }
                />
              </div>
            </details>
            {editing.id && (
              <Alert color="gray">
                리디렉션 URI: {base.replace(/\/$/, "")}/api/v1/auth/sso/
                {editing.id}/callback
              </Alert>
            )}
            <Button type="submit" loading={busy}>
              SSO 연결 저장
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        opened={!!remove}
        onClose={() => setRemove(null)}
        title="SSO 연결 삭제"
        centered
      >
        <p>{remove?.name} 로그인 연결을 삭제하시겠어요?</p>
        <Button color="red" onClick={del}>
          연결 삭제
        </Button>
      </Modal>
    </>
  );
}
const initialConnector = {
  name: "",
  type: "json",
  dataset: "jobs",
  enabled: true,
  endpoint: "",
  apiKey: "",
  authHeader: "Authorization",
  apiKeyParam: "",
  dsn: "",
  query: "",
  rootPath: "",
  mapping: {
    id: "id",
    title: "title",
    organization: "organization",
    region: "region",
    url: "url",
    description: "description",
    skills: "skills",
  },
  params: {},
  method: "GET",
  body: "",
};
export function ConnectorsPage() {
  const resource = useResource<any[]>("/admin/connectors"),
    { notify } = useApp(),
    [editing, setEditing] = useState<any>(null),
    [mapping, setMapping] = useState(""),
    [params, setParams] = useState(""),
    [remove, setRemove] = useState<any>(null),
    [preview, setPreview] = useState<any>(null),
    [busy, setBusy] = useState(""),
    [importOpen, setImportOpen] = useState(false),
    [importData, setImportData] = useState(""),
    [dataset, setDataset] = useState("jobs");
  function edit(c: any) {
    setEditing({ ...c, apiKey: "", dsn: "" });
    setMapping(JSON.stringify(c.mapping || {}, null, 2));
    setParams(JSON.stringify(c.params || {}, null, 2));
  }
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy("save");
    try {
      const body = {
        ...editing,
        mapping: JSON.parse(mapping || "{}"),
        params: JSON.parse(params || "{}"),
      };
      await api(
        editing.id ? `/admin/connectors/${editing.id}` : "/admin/connectors",
        editing.id ? "PUT" : "POST",
        body,
      );
      setEditing(null);
      notify("데이터 연결을 저장했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function run(c: any, action: string) {
    setBusy(`${c.id}:${action}`);
    try {
      const result = await api(`/admin/connectors/${c.id}/${action}`, "POST");
      if (action === "test") setPreview(result);
      notify(
        `${action === "test" ? "연결 테스트" : "데이터 동기화"} 완료 · ${result.count || 0}개 항목`,
      );
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function del() {
    try {
      await api(`/admin/connectors/${remove.id}`, "DELETE");
      setRemove(null);
      void resource.reload();
      notify("데이터 연결을 삭제했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  async function importRecords() {
    setBusy("import");
    try {
      const parsed = JSON.parse(importData);
      if (!Array.isArray(parsed))
        throw new Error("JSON 배열 형식으로 입력해 주세요.");
      const result = await api("/admin/import", "POST", {
        dataset,
        records: parsed,
      });
      notify(`${result.count}개 데이터를 가져왔습니다.`);
      setImportOpen(false);
      setImportData("");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <PageTitle
        title="데이터를 연결하고, 가능성을 확장"
        description="외부 API와 내부 데이터베이스를 같은 형식으로 매핑하여 활용하세요."
        action={
          <div className="button-row compact">
            <Button
              variant="default"
              leftSection={<IconUpload size={18} />}
              onClick={() => setImportOpen(true)}
            >
              JSON 가져오기
            </Button>
            <Button
              leftSection={<IconPlus size={18} />}
              onClick={() => edit({ ...initialConnector })}
            >
              연결 추가
            </Button>
          </div>
        }
      />
      <div className="integration-strip">
        <span>
          <IconDatabase size={22} />
          PostgreSQL · MySQL
        </span>
        <span>
          <IconLink size={22} />
          JSON · XML API
        </span>
        <span>
          <IconDownload size={22} />
          CSV · 파일 가져오기
        </span>
      </div>
      <Alert color="teal" title="고용24와 사내 데이터도 유연하게">
        인증키 전달 방식, 요청 매개변수, 결과 목록 경로와 필드 매핑을
        설정하세요. 연결 테스트로 데이터를 확인한 뒤 동기화하면 서비스에
        반영됩니다.
      </Alert>
      <Loading loading={resource.loading} error={resource.error} />
      {resource.data?.length === 0 && (
        <Empty
          title="아직 연결된 데이터 소스가 없습니다"
          description="훈련·채용·직무 정보를 제공하는 API 또는 데이터베이스를 연결하세요."
        />
      )}
      <div className="connector-list">
        {resource.data?.map((c) => (
          <Panel key={c.id}>
            <div className="panel-title">
              <div className="connector-title">
                <span className="admin-card-icon">
                  <IconDatabase size={23} />
                </span>
                <div>
                  <h2>{c.name}</h2>
                  <span className="muted">
                    {c.type.toUpperCase()} ·{" "}
                    {
                      (
                        {
                          jobs: "채용 공고",
                          training: "교육 · 훈련",
                          occupations: "직무 정보",
                        } as any
                      )[c.dataset]
                    }
                  </span>
                </div>
              </div>
              <Badge color={c.enabled ? "teal" : "gray"}>
                {c.enabled ? "활성" : "비활성"}
              </Badge>
            </div>
            <p className="connector-endpoint">
              {c.endpoint ||
                (c.hasDsn ? "암호화된 데이터베이스 연결" : "DB 연결 설정")}
            </p>
            {c.lastError && (
              <Alert color="red" mb="md">
                {c.lastError}
              </Alert>
            )}
            <div className="connector-footer">
              <small className="muted">
                마지막 동기화:{" "}
                {c.lastSync
                  ? new Date(c.lastSync).toLocaleString("ko-KR")
                  : "아직 없음"}
              </small>
              <div className="button-row compact">
                <Button
                  variant="default"
                  size="sm"
                  loading={busy === `${c.id}:test`}
                  onClick={() => run(c, "test")}
                >
                  연결 테스트
                </Button>
                <Button
                  variant="light"
                  size="sm"
                  loading={busy === `${c.id}:sync`}
                  onClick={() => run(c, "sync")}
                  leftSection={<IconRefresh size={16} />}
                >
                  동기화
                </Button>
                <Button variant="subtle" size="sm" onClick={() => edit(c)}>
                  수정
                </Button>
                <Button
                  variant="subtle"
                  color="red"
                  size="sm"
                  onClick={() => setRemove(c)}
                >
                  삭제
                </Button>
              </div>
            </div>
          </Panel>
        ))}
      </div>
      <Modal
        opened={!!editing}
        onClose={() => setEditing(null)}
        title={editing?.id ? "데이터 연결 수정" : "데이터 연결 추가"}
        size="xl"
        centered
      >
        {editing && (
          <form className="form-stack" onSubmit={save}>
            <div className="form-grid">
              <TextInput
                required
                label="연결 이름"
                placeholder="예: 고용24 채용정보"
                value={editing.name}
                onChange={(e) =>
                  setEditing({ ...editing, name: e.target.value })
                }
              />
              <Select
                label="데이터 형식"
                value={editing.type}
                data={[
                  { value: "json", label: "JSON API" },
                  { value: "xml", label: "XML API" },
                  { value: "csv", label: "CSV" },
                  { value: "postgres", label: "PostgreSQL" },
                  { value: "mysql", label: "MySQL" },
                ]}
                onChange={(v) => setEditing({ ...editing, type: v })}
              />
              <Select
                label="데이터 용도"
                value={editing.dataset}
                data={[
                  { value: "jobs", label: "채용 공고" },
                  { value: "training", label: "교육 · 훈련" },
                  { value: "occupations", label: "직무 정보" },
                ]}
                onChange={(v) => setEditing({ ...editing, dataset: v })}
              />
              <Switch
                mt="xl"
                label="연결 활성화"
                checked={editing.enabled}
                onChange={(e) =>
                  setEditing({ ...editing, enabled: e.target.checked })
                }
              />
            </div>
            {["postgres", "mysql"].includes(editing.type) ? (
              <>
                <PasswordInput
                  label="데이터베이스 DSN"
                  placeholder={
                    editing.hasDsn
                      ? "저장된 연결 유지 · 변경 시 입력"
                      : editing.type === "postgres"
                        ? "postgres://user:password@db:5432/database"
                        : "user:password@tcp(db:3306)/database"
                  }
                  value={editing.dsn || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, dsn: e.target.value })
                  }
                />
                <Textarea
                  label="읽기 전용 SELECT 쿼리"
                  minRows={3}
                  placeholder="SELECT id, title, organization, region, url FROM opportunities"
                  value={editing.query || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, query: e.target.value })
                  }
                />
              </>
            ) : (
              <>
                <TextInput
                  required
                  label="Endpoint URL"
                  placeholder="https://api.example.com/jobs"
                  value={editing.endpoint || ""}
                  onChange={(e) =>
                    setEditing({ ...editing, endpoint: e.target.value })
                  }
                />
                <div className="form-grid">
                  <Select
                    label="HTTP 메서드"
                    value={editing.method || "GET"}
                    data={["GET", "POST"]}
                    onChange={(v) => setEditing({ ...editing, method: v })}
                  />
                  <TextInput
                    label="결과 목록 경로"
                    placeholder="예: data.items 또는 wantedRoot.wanted"
                    value={editing.rootPath || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, rootPath: e.target.value })
                    }
                  />
                  <PasswordInput
                    label="API 인증키"
                    description="비워 두면 기존 키를 유지합니다."
                    value={editing.apiKey || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, apiKey: e.target.value })
                    }
                  />
                  <TextInput
                    label="인증 헤더 이름"
                    placeholder="Authorization"
                    value={editing.authHeader || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, authHeader: e.target.value })
                    }
                  />
                  <TextInput
                    label="인증키 쿼리 매개변수"
                    placeholder="예: authKey (헤더 대신 쿼리로 전달)"
                    value={editing.apiKeyParam || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, apiKeyParam: e.target.value })
                    }
                  />
                </div>
                <Textarea
                  label="요청 매개변수 (JSON 객체)"
                  placeholder={'{"returnType":"XML","display":"100"}'}
                  minRows={3}
                  value={params}
                  onChange={(e) => setParams(e.target.value)}
                  styles={{ input: { fontFamily: "monospace" } }}
                />
                {editing.method === "POST" && (
                  <Textarea
                    label="요청 본문"
                    minRows={3}
                    value={editing.body || ""}
                    onChange={(e) =>
                      setEditing({ ...editing, body: e.target.value })
                    }
                  />
                )}
              </>
            )}
            <Textarea
              label="필드 매핑 (JSON 객체)"
              description="NextRole 필드: 원본 데이터 필드 경로. 예: title: wantedTitle, organization: company"
              minRows={7}
              value={mapping}
              onChange={(e) => setMapping(e.target.value)}
              styles={{ input: { fontFamily: "monospace" } }}
            />
            <Alert color="gray">
              채용·훈련: id, title, organization, region, url, description,
              skills. 직무: id, title, category, description, skills, domain.
              출처는 동기화 시 유지됩니다.
            </Alert>
            <Button type="submit" loading={busy === "save"}>
              연결 저장
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        opened={!!remove}
        onClose={() => setRemove(null)}
        title="데이터 연결 삭제"
        centered
      >
        <p>{remove?.name} 연결 설정을 삭제하시겠어요?</p>
        <Button color="red" onClick={del}>
          연결 삭제
        </Button>
      </Modal>
      <Modal
        opened={!!preview}
        onClose={() => setPreview(null)}
        title="연결 테스트 결과"
        size="xl"
        centered
      >
        <Badge color="teal">{preview?.count || 0}개 항목 확인</Badge>
        <pre className="code-block">
          {JSON.stringify(preview?.preview, null, 2)}
        </pre>
      </Modal>
      <Modal
        opened={importOpen}
        onClose={() => setImportOpen(false)}
        title="JSON 데이터 가져오기"
        size="xl"
        centered
      >
        <div className="form-stack">
          <Select
            label="데이터 용도"
            value={dataset}
            onChange={(v) => setDataset(v || "jobs")}
            data={[
              { value: "jobs", label: "채용 공고" },
              { value: "training", label: "교육 · 훈련" },
              { value: "occupations", label: "직무 정보" },
            ]}
          />
          <Textarea
            label="레코드 JSON 배열"
            minRows={12}
            placeholder={
              '[{"id":"job-1","title":"백엔드 개발자","organization":"기업명","region":"서울","url":"https://example.com/job/1","skills":["Java"],"source":{"name":"사내 데이터","url":"","synthetic":false}}]'
            }
            value={importData}
            onChange={(e) => setImportData(e.target.value)}
            styles={{ input: { fontFamily: "monospace" } }}
          />
          <label className="file-import">
            JSON 파일 선택
            <input
              type="file"
              accept=".json,application/json"
              onChange={async (e) => {
                const f = e.target.files?.[0];
                if (f) setImportData(await f.text());
              }}
            />
          </label>
          <Button loading={busy === "import"} onClick={importRecords}>
            데이터 가져오기
          </Button>
        </div>
      </Modal>
    </>
  );
}
export function AuditPage() {
  const resource = useResource<any[]>("/admin/audit"),
    [search, setSearch] = useState("");
  return (
    <>
      <PageTitle
        title="변경과 접근의 기록"
        description="관리 설정과 주요 작업 이력을 확인해 서비스 운영을 추적하세요."
        action={
          <Button
            variant="default"
            onClick={() => resource.reload()}
            leftSection={<IconRefresh size={18} />}
          >
            새로고침
          </Button>
        }
      />
      <Panel>
        <TextInput
          aria-label="감사 로그 검색"
          placeholder="사용자 · 작업 · 대상 검색"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
        <Loading loading={resource.loading} error={resource.error} />
        <div className="table-scroll">
          <Table verticalSpacing="md">
            <Table.Thead>
              <Table.Tr>
                <Table.Th>일시</Table.Th>
                <Table.Th>작업자</Table.Th>
                <Table.Th>작업</Table.Th>
                <Table.Th>대상</Table.Th>
                <Table.Th>상세</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {resource.data
                ?.filter((a) =>
                  JSON.stringify(a)
                    .toLowerCase()
                    .includes(search.toLowerCase()),
                )
                .map((a) => (
                  <Table.Tr key={a.id}>
                    <Table.Td className="nowrap">
                      {new Date(a.createdAt).toLocaleString("ko-KR")}
                    </Table.Td>
                    <Table.Td>{a.actor || "시스템"}</Table.Td>
                    <Table.Td>
                      <Badge variant="light" color="gray">
                        {a.action}
                      </Badge>
                    </Table.Td>
                    <Table.Td>{a.target || "—"}</Table.Td>
                    <Table.Td className="audit-details">
                      {typeof a.details === "string"
                        ? a.details
                        : JSON.stringify(a.details || {})}
                    </Table.Td>
                  </Table.Tr>
                ))}
            </Table.Tbody>
          </Table>
        </div>
        {resource.data?.length === 0 && (
          <Empty title="기록된 작업이 없습니다" />
        )}
      </Panel>
    </>
  );
}
