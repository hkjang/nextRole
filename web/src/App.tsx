import { createContext, useContext, useEffect, useRef, useState } from "react";
import {
  Link,
  NavLink,
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";
import {
  Alert,
  Avatar,
  Badge,
  Button,
  Loader,
  Menu,
  Modal,
  PasswordInput,
  TextInput,
} from "@mantine/core";
import {
  IconArrowUpRight,
  IconArrowsShuffle,
  IconBriefcase2,
  IconChartBar,
  IconCheck,
  IconChevronDown,
  IconCompass,
  IconDatabase,
  IconFileCheck,
  IconHistory,
  IconKey,
  IconLayoutDashboard,
  IconLogout,
  IconMenu2,
  IconRoute,
  IconSettings,
  IconShieldLock,
  IconSparkles,
  IconTerminal2,
  IconUser,
  IconUsers,
  IconX,
} from "@tabler/icons-react";
import { api, APIError, User } from "./api";
import {
  Dashboard,
  ProfilePage,
  DiscoverPage,
  ComparePage,
  SimulatorPage,
  RoadmapPage,
  OpportunitiesPage,
  SavedPage,
  KeysPage,
  PreferencesPage,
  ApprovalsPage,
  APIPage,
} from "./UserPages";
import {
  AdminOverview,
  UsersPage,
  SettingsPage,
  ProvidersPage,
  ConnectorsPage,
  AuditPage,
} from "./AdminPages";
import { PrivacyPage, DataPolicyPage, CareerConsentGate } from "./PrivacyPages";
import { DataSourcesPage, MappingsPage } from "./AdminDataPages";
export const AppContext = createContext<any>(null);
export function useApp() {
  return useContext(AppContext);
}
export function useResource<T = any>(path: string | null) {
  const [data, setData] = useState<T | null>(null),
    [error, setError] = useState(""),
    [loading, setLoading] = useState(true);
  const sequence = useRef(0);
  async function reload() {
    if (!path) {
      setData(null);
      setLoading(false);
      setError("");
      return;
    }
    const current = ++sequence.current;
    setLoading(true);
    setError("");
    try {
      const result = await api<T>(path);
      if (sequence.current === current) setData(result);
    } catch (e) {
      if (sequence.current === current) setError((e as Error).message);
    } finally {
      if (sequence.current === current) setLoading(false);
    }
  }
  useEffect(() => {
    void reload();
    return () => {
      sequence.current++;
    };
  }, [path]);
  return { data, setData, error, loading, reload };
}
export function PageTitle({
  eyebrow,
  title,
  description,
  action,
}: {
  eyebrow?: string;
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="page-heading">
      <div>
        <div className="eyebrow">{eyebrow || "MY NEXT CHAPTER"}</div>
        <h1>{title}</h1>
        {description && <p>{description}</p>}
      </div>
      {action}
    </div>
  );
}
export function Loading({
  error,
  loading,
}: {
  error?: string;
  loading?: boolean;
}) {
  return error ? (
    <Alert color="red" title="잠시 확인해 주세요">
      {error}
    </Alert>
  ) : loading ? (
    <div className="loading">
      <Loader color="teal" />
      <span>데이터를 불러오고 있어요</span>
    </div>
  ) : null;
}
export function Empty({
  title,
  description,
  action,
}: {
  title: string;
  description?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="empty">
      <IconCompass size={38} />
      <h3>{title}</h3>
      <p>{description}</p>
      {action}
    </div>
  );
}
export const sourceKinds: Record<string, { label: string; color: string }> = {
  public_api: { label: "공공 API 원천", color: "blue" },
  user_input: { label: "사용자 입력", color: "grape" },
  synthetic: { label: "합성 예시", color: "orange" },
  derived: { label: "가공·추론", color: "violet" },
  external: { label: "외부 데이터", color: "gray" },
};
export function SourceTag({ source }: { source?: any }) {
  if (!source) return null;
  const kind = source.synthetic ? "synthetic" : source.kind || "external";
  const info = sourceKinds[kind] || sourceKinds.external;
  return (
    <div className="source-tag">
      <Badge color={info.color} variant="light">
        {info.label}
      </Badge>
      {source.url ? (
        <a href={source.url} target="_blank" rel="noreferrer">
          {source.name || source.provider || "원문 출처"} ↗
        </a>
      ) : (
        <span>{source.name || source.provider || "출처 확인 필요"}</span>
      )}
      {kind === "derived" && (
        <span className="source-derived-note">
          공식 원문과 구분되는 내부 가공값
        </span>
      )}
    </div>
  );
}
export function Panel({
  children,
  title,
  action,
  className = "",
}: {
  children: React.ReactNode;
  title?: string;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <section className={`panel ${className}`}>
      {title && (
        <div className="panel-title">
          <h2>{title}</h2>
          {action}
        </div>
      )}
      {children}
    </section>
  );
}
export function Score({
  value,
  size = "normal",
}: {
  value: number;
  size?: string;
}) {
  return (
    <div
      className={`score-ring ${size}`}
      style={{ "--score": `${value}%` } as React.CSSProperties}
    >
      <div>
        <strong>{Math.round(value)}</strong>
        <span>적합도 / 100</span>
      </div>
    </div>
  );
}
export const number = (n: number) =>
  new Intl.NumberFormat("ko-KR").format(n || 0);
const userNav = [
  ["/", "대시보드", IconLayoutDashboard],
  ["/profile", "내 경력", IconUser],
  ["/discover", "직무 발견", IconCompass],
  ["/compare", "직무 비교", IconChartBar],
  ["/simulator", "경력 시뮬레이터", IconArrowsShuffle],
  ["/roadmap", "나의 로드맵", IconRoute],
  ["/opportunities", "교육 · 채용", IconBriefcase2],
  ["/saved", "저장한 시뮬레이션", IconHistory],
] as const;
const adminNav = [
  ["/admin", "관리 대시보드", IconLayoutDashboard],
  ["/admin/users", "사용자 관리", IconUsers],
  ["/admin/general", "서비스 설정", IconSettings],
  ["/admin/ai", "AI 모델 설정", IconSparkles],
  ["/admin/security", "보안 · 키 정책", IconShieldLock],
  ["/admin/scoring", "점수 모델", IconChartBar],
  ["/admin/providers", "SSO 로그인 연동", IconKey],
  ["/admin/connectors", "데이터 연동", IconDatabase],
  ["/admin/data", "원천·가공 데이터", IconDatabase],
  ["/admin/mappings", "역량 매핑 검토", IconFileCheck],
  ["/admin/data-policy", "개인정보 안내", IconShieldLock],
  ["/admin/audit", "감사 로그", IconHistory],
] as const;
function Auth() {
  const { publicInfo, setUser, notify } = useApp(),
    [register, setRegister] = useState(false),
    [email, setEmail] = useState(""),
    [password, setPassword] = useState(""),
    [name, setName] = useState(""),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  const nav = useNavigate();
  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const data = await api(
        `/auth/${register ? "register" : "login"}`,
        "POST",
        { email, password, name },
      );
      setUser(data.user);
      notify(
        register
          ? "가입을 완료했습니다. 나의 다음 경력을 만나 보세요."
          : "환영합니다. 오늘의 가능성을 탐색해 보세요.",
      );
      nav("/");
    } catch (e) {
      setError((e as Error).message);
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="auth-page">
      <div className="auth-story">
        <Link to="/" className="brand brand-white">
          <img src="/favicon.svg" alt="" />
          NextRole<span>CAREER LAB</span>
        </Link>
        <div className="auth-copy">
          <div className="pill light">
            내일의 나를 만나는 가장 구체적인 방법
          </div>
          <h1>
            당신의 경험은
            <br />
            다음 가능성으로
            <br />
            <em>이어집니다.</em>
          </h1>
          <p>
            지금까지 쌓아 온 역량을 발견하고,
            <br />
            내가 원하는 미래를 자유롭게 실험해 보세요.
          </p>
          <div className="auth-path">
            <div>
              지금의 나<small>경력 · 경험 · 강점</small>
            </div>
            <IconArrowUpRight />
            <div>
              다음의 나<small>새로운 역량 · 가능성</small>
            </div>
          </div>
        </div>
        <div className="auth-bottom">
          CAREER IS A JOURNEY. MAKE IT YOURS.
          <span>© {new Date().getFullYear()} NextRole</span>
        </div>
      </div>
      <div className="auth-form">
        <div className="auth-mobile-brand brand">
          <img src="/favicon.svg" alt="" />
          NextRole
        </div>
        <div className="auth-form-inner">
          <Badge variant="light" size="lg" color="teal">
            나의 다음 커리어, NextRole
          </Badge>
          <h2>{register ? "새로운 가능성의 시작" : "다시 만나 반가워요"}</h2>
          <p>
            {register
              ? "계정을 만들고 나만의 경력 실험실을 시작하세요."
              : "로그인하고 나만의 경력 여정을 이어가세요."}
          </p>
          <form onSubmit={submit}>
            {register && (
              <TextInput
                required
                label="이름"
                value={name}
                onChange={(e) => setName(e.target.value)}
                size="md"
              />
            )}
            <TextInput
              required
              label={register ? "이메일" : "이메일 또는 계정"}
              type={register ? "email" : "text"}
              placeholder="name@company.com 또는 계정"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              size="md"
              autoComplete="username"
            />
            <PasswordInput
              required
              label="비밀번호"
              placeholder="비밀번호를 입력하세요"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              size="md"
              autoComplete={register ? "new-password" : "current-password"}
            />
            {register && <small>비밀번호는 12자 이상으로 설정해 주세요.</small>}
            {error && <Alert color="red">{error}</Alert>}
            <Button
              fullWidth
              size="lg"
              type="submit"
              loading={busy}
              rightSection={<IconArrowUpRight size={20} />}
            >
              {register ? "계정 만들기" : "로그인"}
            </Button>
          </form>
          {(publicInfo.providers || []).length > 0 && (
            <>
              <div className="separator">간편 로그인</div>
              <div className="sso-buttons">
                {publicInfo.providers.map((p: any) => (
                  <Button
                    key={p.id}
                    component="a"
                    href={`/api/v1/auth/sso/${p.id}/start`}
                    variant="default"
                    size="md"
                  >
                    {p.name}로 계속하기
                  </Button>
                ))}
              </div>
            </>
          )}
          {publicInfo.registrationEnabled && (
            <button
              className="text-button auth-switch"
              onClick={() => {
                setRegister(!register);
                setError("");
              }}
            >
              {register
                ? "이미 계정이 있나요? 로그인"
                : "처음이신가요? 계정 만들기"}
            </button>
          )}
          <div className="auth-note">
            <IconShieldLock size={17} />
            <span>나의 경력은 안전하게, 가능성은 자유롭게.</span>
          </div>
        </div>
        <div className="auth-version">
          NextRole <span>v{publicInfo.version || "1.0.0"}</span> · AI 경력전환
          시뮬레이터
        </div>
      </div>
    </div>
  );
}
function Shell() {
  const { user, setUser, publicInfo, notify } = useApp(),
    location = useLocation(),
    [mobile, setMobile] = useState(false),
    [about, setAbout] = useState(false);
  const admin = location.pathname.startsWith("/admin");
  useEffect(() => {
    setMobile(false);
    document.title = `${[...userNav, ...adminNav].find((n) => n[0] === location.pathname)?.[1] || "나의 다음 가능성"} · NextRole`;
  }, [location.pathname]);
  async function logout() {
    try {
      await api("/auth/logout", "POST");
      setUser(null);
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  const nav = admin ? adminNav : userNav;
  return (
    <div className="app-shell">
      {mobile && (
        <div className="sidebar-backdrop" onClick={() => setMobile(false)} />
      )}
      <aside className={`sidebar ${mobile ? "open" : ""}`}>
        <Link to="/" className="brand">
          <img src="/favicon.svg" alt="NextRole 로고" />
          <span>
            NextRole<small>YOUR NEXT POSSIBILITY</small>
          </span>
        </Link>
        <div className="workspace-label">
          {admin ? "서비스 관리 공간" : "나의 커리어 공간"}
          <Badge size="xs" variant="light" color={admin ? "orange" : "teal"}>
            {admin ? "ADMIN" : "PERSONAL"}
          </Badge>
        </div>
        <nav>
          {nav.map(([path, label, Icon]) => (
            <NavLink
              key={path}
              to={path}
              end={path === "/" || path === "/admin"}
              className={({ isActive }) =>
                `nav-item ${isActive ? "active" : ""}`
              }
            >
              <Icon size={21} stroke={1.7} />
              <span>{label}</span>
              {path === "/simulator" && <span className="nav-dot" />}
            </NavLink>
          ))}
          {!admin && publicInfo.approvalEnabled && (
            <NavLink to="/approvals" className="nav-item">
              <IconFileCheck size={21} />
              검토 · 승인
            </NavLink>
          )}
          <div className="nav-divider" />
          {!admin && (
            <>
              <NavLink to="/keys" className="nav-item">
                <IconKey size={21} />
                개인 API 키
              </NavLink>
              <NavLink to="/api" className="nav-item">
                <IconTerminal2 size={21} />
                API · MCP
              </NavLink>
              <NavLink to="/privacy" className="nav-item">
                <IconShieldLock size={21} />
                개인정보·동의
              </NavLink>
              <NavLink to="/preferences" className="nav-item">
                <IconSettings size={21} />
                개인 설정
              </NavLink>
            </>
          )}
          {user.role === "admin" && (
            <Link to={admin ? "/" : "/admin"} className="nav-item switch-space">
              {admin ? <IconUser size={21} /> : <IconShieldLock size={21} />}
              <span>{admin ? "개인 공간으로" : "서비스 관리"}</span>
              <IconArrowUpRight size={17} />
            </Link>
          )}
        </nav>
        <div className="sidebar-bottom">
          <div className="sidebar-tip">
            <IconSparkles size={23} />
            <strong>변화는 작은 실험에서</strong>
            <p>
              역량 하나를 더하면,
              <br />
              어떤 미래가 열릴까요?
            </p>
            <Link to="/simulator">
              시뮬레이션 시작하기 <IconArrowUpRight size={16} />
            </Link>
          </div>
          <span className="version-small">
            NextRole v{publicInfo.version || user.version || "1.0.0"}
          </span>
        </div>
      </aside>
      <div className="main-shell">
        <header className="topbar">
          <div className="topbar-left">
            <button
              className="icon-button mobile-toggle"
              aria-label="메뉴 열기"
              onClick={() => setMobile(true)}
            >
              <IconMenu2 />
            </button>
            <span className="breadcrumb">
              {admin ? "서비스 관리" : "나의 커리어"}
              <span>/</span>
              <strong>
                {[...userNav, ...adminNav].find(
                  (n) => n[0] === location.pathname,
                )?.[1] ||
                  (
                    {
                      "/keys": "개인 API 키",
                      "/api": "API · MCP",
                      "/preferences": "개인 설정",
                      "/privacy": "개인정보·동의",
                      "/approvals": "검토 · 승인",
                    } as any
                  )[location.pathname] ||
                  "NextRole"}
              </strong>
            </span>
          </div>
          <div className="topbar-right">
            <span className="private-label">
              <span />
              나만의 안전한 경력 공간
            </span>
            <Menu width={245} position="bottom-end" shadow="md">
              <Menu.Target>
                <button className="profile-button">
                  <Avatar color="teal" radius="xl" size={35}>
                    {(user.name || user.email)?.slice(0, 1)}
                  </Avatar>
                  <span>
                    {user.name || "사용자"}
                    <small>
                      {user.role === "admin"
                        ? "관리자"
                        : user.role === "reviewer"
                          ? "검토자"
                          : "개인 계정"}
                    </small>
                  </span>
                  <IconChevronDown size={17} />
                </button>
              </Menu.Target>
              <Menu.Dropdown>
                <Menu.Label>{user.email}</Menu.Label>
                <Menu.Item
                  component={Link}
                  to="/preferences"
                  leftSection={<IconUser size={17} />}
                >
                  개인 설정
                </Menu.Item>
                <Menu.Item
                  component={Link}
                  to="/keys"
                  leftSection={<IconKey size={17} />}
                >
                  API 키 관리
                </Menu.Item>
                <Menu.Item
                  onClick={() => setAbout(true)}
                  leftSection={<IconSparkles size={17} />}
                >
                  NextRole v{publicInfo.version || "1.0.0"}
                </Menu.Item>
                <Menu.Divider />
                <Menu.Item
                  color="red"
                  onClick={logout}
                  leftSection={<IconLogout size={17} />}
                >
                  로그아웃
                </Menu.Item>
              </Menu.Dropdown>
            </Menu>
          </div>
        </header>
        <main className="main-content">
          <Routes>
            <Route
              path="/"
              element={
                <CareerConsentGate>
                  <Dashboard />
                </CareerConsentGate>
              }
            />
            <Route path="/profile" element={<ProfilePage />} />
            <Route
              path="/discover"
              element={
                <CareerConsentGate>
                  <DiscoverPage />
                </CareerConsentGate>
              }
            />
            <Route
              path="/compare"
              element={
                <CareerConsentGate>
                  <ComparePage />
                </CareerConsentGate>
              }
            />
            <Route
              path="/simulator"
              element={
                <CareerConsentGate>
                  <SimulatorPage />
                </CareerConsentGate>
              }
            />
            <Route
              path="/roadmap"
              element={
                <CareerConsentGate>
                  <RoadmapPage />
                </CareerConsentGate>
              }
            />
            <Route
              path="/opportunities"
              element={
                <CareerConsentGate>
                  <OpportunitiesPage />
                </CareerConsentGate>
              }
            />
            <Route path="/saved" element={<SavedPage />} />
            <Route path="/keys" element={<KeysPage />} />
            <Route path="/api" element={<APIPage />} />
            <Route path="/preferences" element={<PreferencesPage />} />
            <Route path="/privacy" element={<PrivacyPage />} />
            <Route
              path="/approvals"
              element={
                publicInfo.approvalEnabled ? (
                  <ApprovalsPage />
                ) : (
                  <Navigate to="/" replace />
                )
              }
            />
            {user.role === "admin" && (
              <>
                <Route path="/admin" element={<AdminOverview />} />
                <Route path="/admin/users" element={<UsersPage />} />
                <Route
                  path="/admin/general"
                  element={<SettingsPage section="general" />}
                />
                <Route
                  path="/admin/ai"
                  element={<SettingsPage section="ai" />}
                />
                <Route
                  path="/admin/security"
                  element={<SettingsPage section="security" />}
                />
                <Route
                  path="/admin/scoring"
                  element={<SettingsPage section="scoring" />}
                />
                <Route path="/admin/providers" element={<ProvidersPage />} />
                <Route path="/admin/connectors" element={<ConnectorsPage />} />
                <Route path="/admin/data" element={<DataSourcesPage />} />
                <Route path="/admin/mappings" element={<MappingsPage />} />
                <Route path="/admin/data-policy" element={<DataPolicyPage />} />
                <Route path="/admin/audit" element={<AuditPage />} />
              </>
            )}
            <Route
              path="*"
              element={
                <Empty
                  title="페이지를 찾을 수 없습니다"
                  action={
                    <Button component={Link} to="/">
                      대시보드로 이동
                    </Button>
                  }
                />
              }
            />
          </Routes>
          <footer className="page-footer">
            <span>NextRole · 나의 다음 가능성</span>
            <span>점수와 경로는 의사결정 참고 자료입니다.</span>
          </footer>
        </main>
      </div>
      <Modal
        opened={about}
        onClose={() => setAbout(false)}
        title="NextRole 버전 정보"
        centered
      >
        <div className="about-logo">
          <img src="/favicon.svg" width="60" alt="NextRole" />
          <h2>NextRole</h2>
          <Badge size="lg">v{publicInfo.version || "1.0.0"}</Badge>
          <p>
            현재의 경력을 이해하고,
            <br />
            미래의 경력을 실험하는 AI 경력전환 시뮬레이터
          </p>
        </div>
      </Modal>
    </div>
  );
}
export default function App() {
  const [user, setUser] = useState<User | null>(null),
    [publicInfo, setPublicInfo] = useState<any>({
      version: "1.0.0",
      providers: [],
    }),
    [ready, setReady] = useState(false),
    [initError, setInitError] = useState(""),
    [toast, setToast] = useState<{ text: string; kind: string } | null>(null);
  function notify(text: string, kind = "success") {
    setToast({ text, kind });
  }
  useEffect(() => {
    if (toast) {
      const t = setTimeout(() => setToast(null), 6500);
      return () => clearTimeout(t);
    }
  }, [toast]);
  useEffect(() => {
    async function init() {
      try {
        const [p, u] = await Promise.allSettled([
          api("/public"),
          api<User>("/me"),
        ]);
        if (p.status === "fulfilled") setPublicInfo(p.value);
        else setInitError(p.reason.message);
        if (u.status === "fulfilled") setUser(u.value);
        else if (!(u.reason instanceof APIError && u.reason.status === 401))
          setInitError(u.reason.message);
      } finally {
        setReady(true);
      }
    }
    void init();
  }, []);
  return (
    <AppContext.Provider
      value={{ user, setUser, publicInfo, setPublicInfo, notify }}
    >
      {!ready ? (
        <div className="initial-loading">
          <img src="/favicon.svg" width="60" alt="" />
          <h2>NextRole</h2>
          <Loader />
        </div>
      ) : initError ? (
        <div className="initial-loading">
          <Alert color="red" title="서비스에 연결할 수 없습니다">
            {initError}
          </Alert>
          <Button onClick={() => location.reload()}>다시 연결</Button>
        </div>
      ) : user ? (
        <Shell />
      ) : (
        <Auth />
      )}
      {toast && (
        <div
          className={`toast ${toast.kind === "error" ? "error" : ""}`}
          role="status"
        >
          {toast.kind === "error" ? (
            <IconX size={20} />
          ) : (
            <IconCheck size={20} />
          )}
          <span>{toast.text}</span>
          <button aria-label="알림 닫기" onClick={() => setToast(null)}>
            <IconX size={17} />
          </button>
        </div>
      )}
    </AppContext.Provider>
  );
}
