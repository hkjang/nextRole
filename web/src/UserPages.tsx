import { useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  FileButton,
  Modal,
  MultiSelect,
  NumberInput,
  PasswordInput,
  Progress,
  SegmentedControl,
  Select,
  Slider,
  Switch,
  Table,
  TagsInput,
  Textarea,
  TextInput,
} from "@mantine/core";
import {
  IconArrowDown,
  IconArrowRight,
  IconArrowUpRight,
  IconBookmark,
  IconChartBar,
  IconCheck,
  IconChevronRight,
  IconCode,
  IconCopy,
  IconDownload,
  IconFileText,
  IconFlask,
  IconKey,
  IconPlus,
  IconRefresh,
  IconRoute,
  IconSearch,
  IconSparkles,
  IconTargetArrow,
  IconTrash,
  IconUpload,
  IconWand,
} from "@tabler/icons-react";
import MarketInsights from "./MarketInsights";
import {
  api,
  emptyProfile,
  Job,
  Profile,
  Simulation,
  Skill,
  streamAI,
} from "./api";
import {
  Empty,
  Loading,
  number,
  PageTitle,
  Panel,
  Score,
  SourceTag,
  useApp,
  useResource,
} from "./App";
const sample =
  "Java 백엔드 개발자로 10년간 금융 도메인에서 일했습니다. Java, Spring, SQL을 활용한 API 개발과 Linux 운영을 담당했고, Docker 4년, Kubernetes 1년, CI/CD 4년 경험이 있습니다. Python은 기초 수준입니다. AI 플랫폼 엔지니어로 전환하고 싶습니다.";
const sampleProfile: Profile = {
  name: "",
  currentRole: "Java 백엔드 개발자",
  yearsExperience: 10,
  education: "대학교 졸업",
  region: "서울",
  domain: "금융",
  narrative: sample,
  skills: [
    { name: "Java", level: 5, years: 10, confidence: "explicit" },
    { name: "Spring", level: 5, years: 8, confidence: "explicit" },
    { name: "SQL", level: 4, years: 8, confidence: "explicit" },
    { name: "Linux", level: 4, years: 6, confidence: "explicit" },
    { name: "Docker", level: 3, years: 4, confidence: "explicit" },
    { name: "Kubernetes", level: 2, years: 1, confidence: "explicit" },
    { name: "CI/CD", level: 4, years: 4, confidence: "explicit" },
    { name: "Python", level: 1, years: 1, confidence: "explicit" },
  ],
  certifications: [],
  preferences: ["AI 플랫폼 엔지니어"],
  weeklyHours: 10,
};
function JobCard({
  sim,
  index,
  showRanking = false,
}: {
  sim: Simulation;
  index?: number;
  showRanking?: boolean;
}) {
  return (
    <div className="job-card">
      <div className="job-card-top">
        <span className="job-symbol">
          <IconCode size={24} />
        </span>
        <Badge variant="light" color="teal">
          {sim.job.category}
        </Badge>
        {index !== undefined && <span className="rank">0{index + 1}</span>}
      </div>
      <h3>{sim.job.title}</h3>
      <p>{sim.job.description}</p>
      {(sim.marketDemand || 0) > 0 && (
        <Badge variant="light" color="blue" mb="md">
          실제 공고 {number(sim.marketDemand!)}건
        </Badge>
      )}
      <div className="job-score">
        <span>현재 경력 적합도</span>
        <strong>
          {Math.round(sim.score)}
          <small> / 100</small>
        </strong>
      </div>
      <Progress value={sim.score} color="teal" size={6} />
      <div className="tag-row">
        {(sim.strengths || []).slice(0, 3).map((s) => (
          <span key={s} className="skill-tag">
            {s}
          </span>
        ))}
      </div>
      {showRanking && sim.rankingReason && (
        <details>
          <summary>추천 순위의 근거</summary>
          <p>{sim.rankingReason}</p>
        </details>
      )}
      <Link to={`/simulator?job=${sim.job.id}`} className="card-link">
        이 직무로 실험하기 <IconArrowUpRight size={19} />
      </Link>
    </div>
  );
}
export function Dashboard() {
  const { user } = useApp(),
    profile = useResource<Profile>("/profile"),
    recommendations = useResource<Simulation[]>("/recommendations"),
    saved = useResource<Simulation[]>("/simulations");
  const [date] = useState(() =>
    new Date().toLocaleDateString("ko-KR", {
      month: "long",
      day: "numeric",
      weekday: "long",
    }),
  );
  return (
    <>
      <PageTitle
        eyebrow={date}
        title={`${user.name || "회원"}님, 다음 가능성을 만나 보세요.`}
        description="지금까지의 경험을 바탕으로, 앞으로의 커리어를 설계해요."
        action={
          <Button
            variant="default"
            component={Link}
            to="/profile"
            leftSection={<IconFileText size={18} />}
          >
            내 경력 업데이트
          </Button>
        }
      />
      <section className="hero-panel">
        <div className="hero-text">
          <span className="pill">
            <span className="live-dot" />
            나의 미래를 실험하는 커리어 랩
          </span>
          <h2>
            경험은 그대로,
            <br />
            가능성은 <span>더 넓게.</span>
          </h2>
          <p>
            역량 하나를 더하면 어떤 변화가 생길까요?
            <br />
            목표 직무를 바꾸고, 나에게 맞는 전환 경로를 발견하세요.
          </p>
          <Button
            component={Link}
            to="/simulator"
            size="lg"
            color="dark"
            rightSection={<IconArrowUpRight size={21} />}
          >
            경력 시뮬레이션 실행
          </Button>
          <span className="hero-footnote">
            작은 가정 하나로 시작하는 새로운 커리어
          </span>
        </div>
        <div className="hero-graphic" aria-hidden="true">
          <div className="orbit orbit-one" />
          <div className="orbit orbit-two" />
          <div className="floating-label label-top">
            <IconSparkles size={18} /> 가능성을 연결하다
          </div>
          <div className="path-node current">
            <span>01 · CURRENT</span>
            <IconCode size={28} />
            <strong>{profile.data?.currentRole || "지금까지의 경험"}</strong>
            <small>나의 강점과 보유 역량</small>
          </div>
          <div className="path-dots">•••••••••</div>
          <div className="path-node future">
            <span>02 · NEXT ROLE</span>
            <IconTargetArrow size={30} />
            <strong>내가 원하는 미래</strong>
            <small>새로운 역량, 더 넓은 기회</small>
            <div className="future-tags">
              <span>WHAT IF</span>
              <IconArrowUpRight size={24} />
            </div>
          </div>
          <div className="floating-label label-bottom">
            <span className="green-dot" />
            가능성은 계속 확장 중
          </div>
        </div>
      </section>
      <div className="stat-grid">
        <div className="stat-card">
          <span className="stat-icon mint">
            <IconCode />
          </span>
          <div>
            <span>나의 보유 역량</span>
            <strong>
              {profile.data?.skills?.length || 0}
              <small>개</small>
            </strong>
          </div>
          <Link to="/profile">
            <IconArrowUpRight size={19} />
          </Link>
        </div>
        <div className="stat-card">
          <span className="stat-icon peach">
            <IconTargetArrow />
          </span>
          <div>
            <span>발견한 전환 가능 직무</span>
            <strong>
              {recommendations.data?.length || 0}
              <small>개</small>
            </strong>
          </div>
          <Link to="/discover">
            <IconArrowUpRight size={19} />
          </Link>
        </div>
        <div className="stat-card">
          <span className="stat-icon lavender">
            <IconBookmark />
          </span>
          <div>
            <span>저장한 시뮬레이션</span>
            <strong>
              {saved.data?.length || 0}
              <small>개</small>
            </strong>
          </div>
          <Link to="/saved">
            <IconArrowUpRight size={19} />
          </Link>
        </div>
      </div>
      <div className="section-heading">
        <div>
          <span className="eyebrow">DISCOVER YOUR NEXT ROLE</span>
          <h2>내 경험과 연결되는 직무</h2>
        </div>
        <Link to="/discover">
          모든 직무 살펴보기 <IconArrowRight size={18} />
        </Link>
      </div>
      <Loading
        loading={recommendations.loading}
        error={recommendations.error}
      />
      {recommendations.data && (
        <>
          <div className="job-grid">
            {recommendations.data.slice(0, 3).map((sim, i) => (
              <JobCard key={sim.job.id} sim={sim} index={i} />
            ))}
          </div>
          {!profile.data?.skills?.length && (
            <Alert
              mt="lg"
              color="teal"
              icon={<IconSparkles />}
              title="나만의 결과를 준비해 보세요"
            >
              아직 등록된 경력이 없습니다. <Link to="/profile">내 경력</Link>
              에서 경험을 입력하면 더욱 의미 있는 시뮬레이션을 시작할 수 있어요.
            </Alert>
          )}
        </>
      )}
      {user.preferences?.showInsights !== false && (
        <div className="insight-banner">
          <span className="insight-icon">
            <IconFlask size={27} />
          </span>
          <div>
            <h3>정답을 찾기보다, 가능성을 실험하세요.</h3>
            <p>
              적합도는 채용 확률이 아닙니다. 어떤 역량이 다음 단계에 도움이 될지
              살펴보는 출발점이에요.
            </p>
          </div>
          <Link to="/compare">
            경로 비교하기 <IconArrowUpRight size={18} />
          </Link>
        </div>
      )}
    </>
  );
}
function AIExplanation({
  task,
  text,
  jobId,
  onProfile,
  scenario,
}: {
  task: string;
  text?: string;
  jobId?: string;
  onProfile?: (profile: Profile) => void;
  scenario?: any;
}) {
  const { notify } = useApp(),
    [output, setOutput] = useState(""),
    [busy, setBusy] = useState(false),
    [mode, setMode] = useState(""),
    [parsed, setParsed] = useState<Profile | null>(null);
  const abort = useRef<AbortController | null>(null);
  useEffect(() => {
    abort.current?.abort();
    setOutput("");
    setMode("");
    setParsed(null);
  }, [task, text, jobId, JSON.stringify(scenario)]);
  useEffect(() => () => abort.current?.abort(), []);
  async function run() {
    setOutput("");
    setParsed(null);
    setBusy(true);
    setMode("");
    abort.current = new AbortController();
    try {
      setMode(
        await streamAI(
          { task, text, jobId, ...scenario },
          (delta) => setOutput((t) => t + delta),
          abort.current.signal,
          (p) => {
            if (p && Array.isArray(p.skills)) {
              setParsed(p);
              onProfile?.(p);
            }
          },
        ),
      );
    } catch (e) {
      if ((e as Error).name !== "AbortError")
        notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <Panel className="ai-panel">
      <div className="panel-title">
        <h2>
          <IconSparkles size={21} /> AI 커리어 인사이트
        </h2>
        {busy ? (
          <Button variant="subtle" onClick={() => abort.current?.abort()}>
            생성 중지
          </Button>
        ) : (
          <Button
            variant="light"
            onClick={run}
            leftSection={<IconSparkles size={17} />}
          >
            {task === "parse"
              ? "AI 스트리밍 분석"
              : output
                ? "다시 생성"
                : "스트리밍 설명 보기"}
          </Button>
        )}
      </div>
      <p className="muted">
        {task === "parse"
          ? "경력을 스트리밍으로 분석하고, 추출한 내용을 프로필 초안에 반영합니다. 역량과 경험을 검토한 후 경력을 저장해 주세요."
          : "내 경력과 구조화된 직무 데이터를 바탕으로 전환 방향을 설명합니다. AI의 추론은 직접 확인해 주세요."}
      </p>
      {output && (
        <div className="ai-output">
          {task === "parse"
            ? parsed
              ? `경력 구조화 완료 · ${parsed.currentRole || "직무 확인 필요"}\n발견한 역량 ${parsed.skills.length}개: ${parsed.skills.map((s) => s.name).join(", ")}\n추출한 내용을 위 프로필에서 검토한 후 저장하세요.`
              : busy
                ? "경력과 프로젝트에서 역량을 찾고 있어요..."
                : "분석이 중단되었습니다. 현재 프로필을 확인하고 다시 시도해 주세요."
            : output}
          {busy && <span className="typing-cursor" />}
        </div>
      )}
      {mode && (
        <Badge color={mode === "offline" ? "gray" : "teal"}>
          {mode === "offline" ? "오프라인 규칙 기반 설명" : "AI 스트리밍 완료"}
        </Badge>
      )}
    </Panel>
  );
}
export function ProfilePage() {
  const { notify } = useApp(),
    resource = useResource<Profile>("/profile"),
    [profile, setProfile] = useState<Profile>(emptyProfile),
    [busy, setBusy] = useState(""),
    [newSkill, setNewSkill] = useState("");
  useEffect(() => {
    if (resource.data)
      setProfile({
        ...emptyProfile,
        ...resource.data,
        skills: resource.data.skills || [],
        certifications: resource.data.certifications || [],
        preferences: resource.data.preferences || [],
      });
  }, [resource.data]);
  const set = (key: keyof Profile, value: any) =>
    setProfile((p) => ({ ...p, [key]: value }));
  async function save() {
    setBusy("save");
    try {
      await api("/profile", "PUT", profile);
      notify("경력 프로필을 저장했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function parse() {
    if (!profile.narrative.trim())
      return notify("분석할 경력을 입력해 주세요.", "error");
    setBusy("parse");
    try {
      const parsed = await api<Profile>("/profile/parse", "POST", {
        text: profile.narrative,
      });
      setProfile({
        ...profile,
        ...parsed,
        name: profile.name || parsed.name,
        narrative: profile.narrative,
      });
      notify("경력을 분석했습니다. 추출한 내용을 확인한 후 저장해 주세요.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function upload(file: File | null) {
    if (!file) return;
    setBusy("upload");
    try {
      const form = new FormData();
      form.append("file", file);
      const data = await api<Profile>("/profile/upload", "POST", form);
      setProfile({ ...profile, ...data });
      notify("이력서 내용을 추출했습니다. 확인 후 저장해 주세요.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <PageTitle
        title="내 경험이 다음 가능성의 시작"
        description="경력과 역량을 정리하면, 나에게 맞는 경로가 더 선명해져요."
        action={
          <Button
            onClick={save}
            loading={busy === "save"}
            leftSection={<IconCheck size={18} />}
          >
            경력 저장
          </Button>
        }
      />
      <Loading loading={resource.loading} error={resource.error} />
      <div className="profile-layout">
        <div>
          <Panel
            title="말하듯 편하게, 경력을 알려 주세요"
            action={<Badge variant="light">01 · 경력 분석</Badge>}
          >
            <Textarea
              label="나의 경력 이야기"
              placeholder="어떤 일을 해 오셨나요? 담당 업무, 프로젝트, 사용한 기술과 경험 기간을 자유롭게 적어 주세요."
              minRows={7}
              value={profile.narrative}
              onChange={(e) => set("narrative", e.target.value)}
            />
            <div className="button-row">
              <Button
                onClick={parse}
                loading={busy === "parse"}
                leftSection={<IconWand size={18} />}
              >
                경력에서 역량 추출
              </Button>
              <FileButton onChange={upload} accept=".pdf,.docx,.txt">
                {(props) => (
                  <Button
                    {...props}
                    variant="default"
                    loading={busy === "upload"}
                    leftSection={<IconUpload size={18} />}
                  >
                    이력서 업로드
                  </Button>
                )}
              </FileButton>
              <button
                className="text-button"
                onClick={() => {
                  setProfile({ ...sampleProfile, name: profile.name });
                  notify(
                    "예시 경력을 채웠습니다. 저장하면 내 프로필에 적용됩니다.",
                  );
                }}
              >
                예시 경력 채우기
              </button>
            </div>
            <p className="helper">
              PDF · DOCX · TXT 지원. 이미지 스캔 문서는 텍스트 추출이 어려울 수
              있습니다.
            </p>
          </Panel>
          <Panel title="내 경력 기본 정보">
            <div className="form-grid">
              <TextInput
                label="이름"
                value={profile.name}
                onChange={(e) => set("name", e.target.value)}
              />
              <TextInput
                label="현재 직무"
                placeholder="예: 백엔드 개발자"
                value={profile.currentRole}
                onChange={(e) => set("currentRole", e.target.value)}
              />
              <NumberInput
                label="경력 기간 (년)"
                min={0}
                max={70}
                value={profile.yearsExperience}
                onChange={(v) => set("yearsExperience", Number(v))}
              />
              <NumberInput
                label="경력 공백 (개월)"
                description="공백 자체는 감점하지 않고 재진입 준비에 활용합니다."
                min={0}
                max={600}
                value={profile.careerBreakMonths || 0}
                onChange={(v) => set("careerBreakMonths", Number(v))}
              />
              <TextInput
                label="학력"
                value={profile.education}
                onChange={(e) => set("education", e.target.value)}
              />
              <TextInput
                label="희망 지역"
                placeholder="예: 서울, 원격"
                value={profile.region}
                onChange={(e) => set("region", e.target.value)}
              />
              <TextInput
                label="산업 · 도메인"
                placeholder="예: 금융, 커머스"
                value={profile.domain}
                onChange={(e) => set("domain", e.target.value)}
              />
              <TagsInput
                label="자격증"
                description="입력 후 Enter 또는 쉼표로 추가하세요."
                value={profile.certifications}
                onChange={(v) => set("certifications", v)}
              />
              <NumberInput
                label="주당 학습 가능 시간"
                min={1}
                max={80}
                value={profile.weeklyHours}
                onChange={(v) => set("weeklyHours", Number(v))}
              />
              <TagsInput
                label="선호 직무"
                description="입력 후 Enter 또는 쉼표로 추가하세요."
                value={profile.preferences}
                onChange={(v) => set("preferences", v)}
              />
            </div>
          </Panel>
          <AIExplanation
            task="parse"
            text={profile.narrative}
            onProfile={(p) =>
              setProfile((current) => ({
                ...emptyProfile,
                ...current,
                ...p,
                name: current.name || p.name,
                narrative: current.narrative,
                skills: p.skills || [],
                certifications: p.certifications || [],
                preferences: p.preferences || [],
              }))
            }
          />
        </div>
        <Panel
          title="나의 역량"
          action={<Badge variant="light">{profile.skills.length}개</Badge>}
        >
          <p className="muted">
            0 = 경험 없음 · 5 = 전문가 수준
            <br />
            분석된 역량과 신뢰도를 직접 확인해 주세요.
          </p>
          <div className="skills-editor">
            {profile.skills.map((skill, i) => (
              <div className="skill-editor" key={`${skill.name}-${i}`}>
                <div className="skill-editor-head">
                  <strong>{skill.name}</strong>
                  <button
                    className="icon-button"
                    aria-label={`${skill.name} 삭제`}
                    onClick={() =>
                      set(
                        "skills",
                        profile.skills.filter((_, j) => i !== j),
                      )
                    }
                  >
                    <IconTrash size={17} />
                  </button>
                </div>
                <Slider
                  aria-label={`${skill.name} 수준`}
                  min={0}
                  max={5}
                  step={1}
                  value={skill.level}
                  onChange={(v) =>
                    set(
                      "skills",
                      profile.skills.map((s, j) =>
                        j === i ? { ...s, level: v } : s,
                      ),
                    )
                  }
                  marks={[
                    { value: 0, label: "0" },
                    { value: 5, label: "5" },
                  ]}
                />
                <div className="skill-editor-bottom">
                  <NumberInput
                    aria-label={`${skill.name} 경험 년수`}
                    size="xs"
                    min={0}
                    max={60}
                    suffix="년"
                    value={skill.years || 0}
                    onChange={(v) =>
                      set(
                        "skills",
                        profile.skills.map((s, j) =>
                          j === i ? { ...s, years: Number(v) } : s,
                        ),
                      )
                    }
                  />
                  <Select
                    aria-label={`${skill.name} 신뢰도`}
                    size="xs"
                    value={skill.confidence || "explicit"}
                    data={[
                      { value: "explicit", label: "명시한 역량" },
                      { value: "inferred", label: "추론한 역량" },
                      { value: "review", label: "검증 필요" },
                    ]}
                    onChange={(v) =>
                      set(
                        "skills",
                        profile.skills.map((s, j) =>
                          j === i ? { ...s, confidence: v || "explicit" } : s,
                        ),
                      )
                    }
                  />
                </div>
              </div>
            ))}
          </div>
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (
                newSkill.trim() &&
                !profile.skills.some(
                  (s) => s.name.toLowerCase() === newSkill.trim().toLowerCase(),
                )
              ) {
                set("skills", [
                  ...profile.skills,
                  {
                    name: newSkill.trim(),
                    level: 1,
                    years: 0,
                    confidence: "explicit",
                  },
                ]);
                setNewSkill("");
              }
            }}
            className="inline-form"
          >
            <TextInput
              aria-label="새 역량 이름"
              placeholder="역량 이름"
              value={newSkill}
              onChange={(e) => setNewSkill(e.target.value)}
            />
            <Button type="submit" variant="light" aria-label="역량 추가">
              <IconPlus size={19} />
            </Button>
          </form>
        </Panel>
      </div>
    </>
  );
}
export function DiscoverPage() {
  const { notify } = useApp(),
    recs = useResource<Simulation[]>("/recommendations"),
    [query, setQuery] = useState(""),
    [search, setSearch] = useState(""),
    jobs = useResource<Job[]>(`/jobs?q=${encodeURIComponent(search)}`);
  async function feedback(jobId: string, preference: string) {
    try {
      await api("/feedback", "POST", { jobId, preference });
      notify("직무 선호를 반영했습니다.");
      void recs.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageTitle
        title="경험과 가능성이 만나는 곳"
        description="내 경력에서 출발할 수 있는 직무를 발견하고, 새로운 목표를 탐색하세요."
      />
      <Panel className="search-panel">
        <form
          className="search-form"
          onSubmit={(e) => {
            e.preventDefault();
            setSearch(query);
          }}
        >
          <TextInput
            size="lg"
            aria-label="목표 직무 검색"
            placeholder="관심 있는 직무나 기술을 검색해 보세요"
            leftSection={<IconSearch size={21} />}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <Button type="submit" size="lg">
            직무 검색
          </Button>
        </form>
      </Panel>
      {!search && (
        <>
          <div className="section-heading">
            <h2>내 경력에 가까운 TOP 5</h2>
            <Badge color="teal" variant="light">
              설명 가능한 역량 매칭
            </Badge>
          </div>
          <Loading error={recs.error} loading={recs.loading} />
          <div className="job-grid">
            {recs.data?.map((sim, i) => (
              <div key={sim.job.id}>
                <JobCard sim={sim} index={i} showRanking />
                <div className="feedback-row">
                  <button onClick={() => feedback(sim.job.id, "like")}>
                    관심 있어요
                  </button>
                  <button onClick={() => feedback(sim.job.id, "dislike")}>
                    관심 없어요
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}
      <div className="section-heading">
        <h2>{search ? `“${search}” 검색 결과` : "직무 라이브러리"}</h2>
        <span className="muted">{jobs.data?.length || 0}개 직무</span>
      </div>
      <Loading error={jobs.error} loading={jobs.loading} />
      {jobs.data?.length === 0 && (
        <Empty
          title="일치하는 직무가 없습니다"
          description="다른 이름이나 관련 역량으로 검색해 보세요."
        />
      )}
      <div className="job-list">
        {jobs.data?.map((job) => (
          <Panel key={job.id}>
            <div className="job-list-row">
              <span className="job-symbol">
                <IconCode size={24} />
              </span>
              <div className="job-list-main">
                <Badge variant="light">{job.category}</Badge>
                <h3>{job.title}</h3>
                <p>{job.description}</p>
                <div className="tag-row">
                  {job.skills.slice(0, 5).map((s) => (
                    <span className="skill-tag" key={s.name}>
                      {s.name}
                    </span>
                  ))}
                </div>
                <SourceTag source={job.source} />
              </div>
              <Button
                component={Link}
                to={`/simulator?job=${job.id}`}
                variant="light"
                rightSection={<IconArrowUpRight size={18} />}
              >
                시뮬레이션
              </Button>
            </div>
          </Panel>
        ))}
      </div>
    </>
  );
}
function useSimulation() {
  const { user } = useApp();
  const jobs = useResource<Job[]>("/jobs"),
    [params, setParams] = useSearchParams(),
    [jobId, setJobId] = useState(params.get("job") || ""),
    [months, setMonths] = useState(
      String(user.preferences?.defaultMonths || 6),
    ),
    [added, setAdded] = useState<Skill[]>([]),
    [project, setProject] = useState(""),
    [certifications, setCertifications] = useState(""),
    [simulation, setSimulation] = useState<Simulation | null>(null),
    [loading, setLoading] = useState(false),
    [error, setError] = useState("");
  const serial = useRef(0);
  const restored = useRef("");
  useEffect(() => {
    const id = params.get("saved");
    if (!id || restored.current === id) return;
    restored.current = id;
    api<Simulation[]>("/simulations")
      .then((rows) => {
        const saved = rows.find((row) => row.id === id);
        if (saved?.scenario) {
          setJobId(saved.scenario.jobId);
          setMonths(String(saved.scenario.months));
          setAdded(saved.scenario.addedSkills || []);
          setProject(saved.scenario.project || "");
          setCertifications((saved.scenario.certifications || []).join(", "));
        }
      })
      .catch((e) => setError(e.message));
  }, [params]);
  useEffect(() => {
    if (!jobId && jobs.data?.length) {
      const id = jobs.data[0].id;
      setJobId(id);
    }
  }, [jobs.data, jobId]);
  useEffect(() => {
    if (!jobId) return;
    const current = ++serial.current;
    setLoading(true);
    setError("");
    const timer = setTimeout(() => {
      api<Simulation>("/simulate", "POST", {
        jobId,
        months: Number(months),
        addedSkills: added,
        project,
        certifications: certifications
          .split(",")
          .map((v) => v.trim())
          .filter(Boolean),
      })
        .then((v) => {
          if (serial.current === current) setSimulation(v);
        })
        .catch((e) => {
          if (serial.current === current) setError(e.message);
        })
        .finally(() => {
          if (serial.current === current) setLoading(false);
        });
    }, 180);
    return () => clearTimeout(timer);
  }, [jobId, months, added, project, certifications]);
  function selectJob(id: string | null) {
    if (!id) return;
    setJobId(id);
    setAdded([]);
    setProject("");
    setCertifications("");
    setParams({ job: id }, { replace: true });
  }
  return {
    jobs,
    jobId,
    selectJob,
    months,
    setMonths,
    added,
    setAdded,
    project,
    setProject,
    certifications,
    setCertifications,
    simulation,
    setSimulation,
    loading,
    error,
  };
}
function GapChart({ sim }: { sim: Simulation }) {
  return (
    <div className="gap-chart">
      <div className="chart-legend">
        <span>
          <i className="legend-current" />
          현재 역량
        </span>
        <span>
          <i className="legend-required" />
          목표 수준
        </span>
      </div>
      {sim.gaps.map((g) => (
        <div className="gap-row" key={g.name}>
          <div className="gap-label">
            <strong>{g.name}</strong>
            <span>
              {g.current} <small>/ {g.required}</small>
            </span>
          </div>
          <div className="gap-track">
            <div
              className="gap-required"
              style={{ width: `${(g.required / 5) * 100}%` }}
            />
            <div
              className="gap-current"
              style={{ width: `${(Math.min(g.current, 5) / 5) * 100}%` }}
            />
          </div>
          <span className={`gap-status ${g.gap === 0 ? "complete" : ""}`}>
            {g.gap === 0 ? "충족" : `+${g.gap} 필요`}
          </span>
        </div>
      ))}
    </div>
  );
}
export function SimulatorPage() {
  const s = useSimulation(),
    { notify } = useApp(),
    [saving, setSaving] = useState(false),
    [pathSort, setPathSort] = useState("recommended");
  const { project, setProject, certifications, setCertifications } = s;
  const sim = s.simulation;
  async function save() {
    setSaving(true);
    try {
      await api("/simulate", "POST", {
        jobId: s.jobId,
        months: Number(s.months),
        addedSkills: s.added,
        project,
        certifications: certifications
          .split(",")
          .map((v) => v.trim())
          .filter(Boolean),
        save: true,
      });
      notify("시뮬레이션을 저장했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setSaving(false);
    }
  }
  async function apply() {
    try {
      const result = await api("/simulate", "POST", {
        jobId: s.jobId,
        months: Number(s.months),
        addedSkills: s.added,
        project,
        certifications: certifications
          .split(",")
          .map((v) => v.trim())
          .filter(Boolean),
      });
      s.setSimulation(result);
      notify(
        `프로젝트·자격 조건을 적용했습니다. 적합도 ${Math.round(result.score)}점`,
      );
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageTitle
        eyebrow="CAREER SIMULATION LAB"
        title="만약, 이 역량을 더한다면?"
        description="목표와 조건을 바꾸며 나에게 가장 잘 맞는 미래를 실험해 보세요."
        action={
          <Button
            onClick={save}
            disabled={!sim}
            loading={saving}
            variant="default"
            leftSection={<IconBookmark size={18} />}
          >
            시뮬레이션 저장
          </Button>
        }
      />
      <div className="simulation-toolbar">
        <Select
          label="목표 직무"
          placeholder="목표 직무 선택"
          searchable
          value={s.jobId}
          onChange={s.selectJob}
          data={(s.jobs.data || []).map((j) => ({
            value: j.id,
            label: j.title,
          }))}
        />
        <div>
          <label className="field-label">목표 준비 기간</label>
          <SegmentedControl
            value={s.months}
            onChange={s.setMonths}
            data={[
              { value: "3", label: "3개월" },
              { value: "6", label: "6개월" },
              { value: "12", label: "12개월" },
            ]}
            size="md"
          />
        </div>
        <Button
          variant="subtle"
          onClick={() => {
            s.setAdded([]);
            setProject("");
            setCertifications("");
          }}
          leftSection={<IconRefresh size={18} />}
        >
          조건 초기화
        </Button>
      </div>
      <Loading
        loading={!sim && (s.loading || s.jobs.loading)}
        error={s.error || s.jobs.error}
      />
      {sim && (
        <>
          <div className="simulation-grid">
            <div className="simulation-main">
              <Panel className="score-panel">
                <div>
                  <Badge color="teal" variant="light">
                    {sim.job.category}
                  </Badge>
                  <h2>{sim.job.title}</h2>
                  <p>{sim.explanation}</p>
                  <div className="score-metrics">
                    <div>
                      <span>전환 난이도</span>
                      <strong>
                        {Math.round(sim.difficulty)}
                        <small> / 100</small>
                      </strong>
                    </div>
                    <div>
                      <span>예상 준비 기간</span>
                      <strong>
                        {sim.estimatedMonths}
                        <small>개월</small>
                      </strong>
                    </div>
                    <div>
                      <span>시뮬레이션 변화</span>
                      <strong className="positive">
                        +{Math.round(sim.score - sim.baselineScore)}
                        <small>점</small>
                      </strong>
                    </div>
                  </div>
                  <SourceTag source={sim.source} />
                </div>
                <div className="score-change">
                  <Score value={sim.score} />
                  <span>
                    현재 {Math.round(sim.baselineScore)}점{" "}
                    <IconArrowRight size={16} />{" "}
                    <strong>{Math.round(sim.score)}점</strong>
                  </span>
                  {s.loading && <small>다시 계산 중...</small>}
                </div>
              </Panel>
              <Panel
                title="목표까지의 역량 거리"
                action={
                  <Badge color="gray" variant="light">
                    수준 0–5
                  </Badge>
                }
              >
                <GapChart sim={sim} />
              </Panel>
              <Panel title="적합도를 만드는 여섯 가지 근거">
                <div className="factor-list">
                  {sim.factors.map((f) => (
                    <div className="factor-row" key={f.name}>
                      <div>
                        <strong>{f.name}</strong>
                        <p>{f.reason}</p>
                      </div>
                      <span>
                        <b>{Math.round(f.score * 10) / 10}</b> / {f.max}
                      </span>
                    </div>
                  ))}
                </div>
              </Panel>
            </div>
            <div className="simulation-side">
              <Panel className="whatif-panel">
                <div className="panel-title">
                  <h2>
                    <IconFlask size={23} /> What-if 실험
                  </h2>
                  <Badge color="teal">실시간</Badge>
                </div>
                <p>
                  새로운 역량을 갖췄다고 가정해 보세요. 선택하는 순간 결과가
                  달라져요.
                </p>
                {sim.gaps
                  .filter(
                    (g) =>
                      g.current < g.required ||
                      s.added.some((a) => a.name === g.name),
                  )
                  .map((g) => {
                    const selected = s.added.find((a) => a.name === g.name);
                    return (
                      <div
                        className={`whatif-item ${selected ? "selected" : ""}`}
                        key={g.name}
                      >
                        <Checkbox
                          label={g.name}
                          checked={!!selected}
                          onChange={(e) =>
                            s.setAdded(
                              e.target.checked
                                ? [
                                    ...s.added,
                                    { name: g.name, level: g.required },
                                  ]
                                : s.added.filter((a) => a.name !== g.name),
                            )
                          }
                        />
                        {selected ? (
                          <Slider
                            aria-label={`${g.name} 가정 수준`}
                            min={1}
                            max={5}
                            value={selected.level}
                            onChange={(v) =>
                              s.setAdded(
                                s.added.map((a) =>
                                  a.name === g.name ? { ...a, level: v } : a,
                                ),
                              )
                            }
                            marks={[
                              { value: 1, label: "1" },
                              { value: 5, label: "5" },
                            ]}
                          />
                        ) : (
                          <span className="muted">
                            목표 수준 {g.required} 습득
                          </span>
                        )}
                      </div>
                    );
                  })}
                {!sim.gaps.some((g) => g.gap > 0) && s.added.length === 0 && (
                  <p>목표 직무의 모든 요구 역량을 충족했어요.</p>
                )}
                <div className="whatif-summary">
                  <span>선택한 성장 역량</span>
                  <strong>{s.added.length}개</strong>
                </div>
              </Panel>
              <Panel title="학습 효과가 높은 역량">
                <p className="muted">
                  각 역량을 목표 수준까지 익혔을 때의 적합도 변화입니다.
                </p>
                {sim.roi.slice(0, 5).map((r, i) => (
                  <div className="roi-row" key={r.name}>
                    <span className="roi-rank">{i + 1}</span>
                    <strong>{r.name}</strong>
                    <Badge color="teal">
                      +{Math.round(r.gain * 10) / 10}점
                    </Badge>
                  </div>
                ))}
                <Link
                  className="card-link"
                  to={`/opportunities?job=${s.jobId}`}
                >
                  관련 교육 찾아보기 <IconArrowUpRight size={18} />
                </Link>
              </Panel>
              <Panel title="프로젝트 · 자격 실험">
                <Textarea
                  label="프로젝트 경험 가정"
                  placeholder="예: vLLM 기반 모델 서빙 프로젝트"
                  value={project}
                  onChange={(e) => setProject(e.target.value)}
                />
                <TextInput
                  mt="md"
                  label="자격증 가정 (쉼표 구분)"
                  value={certifications}
                  onChange={(e) => setCertifications(e.target.value)}
                />
                <Button variant="light" fullWidth mt="md" onClick={apply}>
                  추가 조건 확인
                </Button>
              </Panel>
            </div>
          </div>
          {sim.warnings?.map((w) => (
            <Alert key={w} color="yellow" mt="md">
              {w}
            </Alert>
          ))}
          <div className="section-heading">
            <div>
              <span className="eyebrow">MORE THAN ONE WAY</span>
              <h2>목표까지, 다양한 경로</h2>
            </div>
            <Select
              aria-label="경로 정렬"
              value={pathSort}
              onChange={(v) => setPathSort(v || "recommended")}
              data={[
                { value: "recommended", label: "추천 순" },
                { value: "fastest", label: "기간 짧은 순" },
                { value: "fewest", label: "추가 역량 적은 순" },
                { value: "reuse", label: "경력 활용 높은 순" },
              ]}
              size="sm"
            />
            <Link to={`/roadmap?job=${s.jobId}`}>
              상세 로드맵 보기 <IconArrowRight size={18} />
            </Link>
          </div>
          <div className="path-grid">
            {[...sim.paths]
              .sort((a, b) =>
                pathSort === "fastest"
                  ? a.months - b.months
                  : pathSort === "fewest"
                    ? a.skills.length - b.skills.length
                    : pathSort === "reuse"
                      ? b.reuse - a.reuse
                      : 0,
              )
              .map((p, i) => (
                <Panel key={p.id} className="path-card">
                  <Badge variant="light" color={i === 0 ? "teal" : "gray"}>
                    {i === 0
                      ? (
                          {
                            recommended: "추천 경로",
                            fastest: "가장 빠른 경로",
                            fewest: "추가 역량 최소",
                            reuse: "경력 활용 최대",
                          } as any
                        )[pathSort]
                      : `대안 경로 ${i}`}
                  </Badge>
                  <h3>{p.title}</h3>
                  <div className="path-steps">
                    {p.roles.map((r, j) => (
                      <div key={`${r}-${j}`}>
                        <span />
                        {r}
                      </div>
                    ))}
                  </div>
                  <div className="path-data">
                    <span>
                      <b>{p.months}개월</b> 예상 기간
                    </span>
                    <span>
                      <b>{p.reuse}%</b> 경력 활용
                    </span>
                  </div>
                  <p>{p.reason}</p>
                  <small>예상 학습 비용 {number(p.cost)}원 · 추정치</small>
                </Panel>
              ))}
          </div>
          <AIExplanation
            task="explain"
            jobId={s.jobId}
            scenario={{
              months: Number(s.months),
              addedSkills: s.added,
              project,
              certifications: certifications
                .split(",")
                .map((v) => v.trim())
                .filter(Boolean),
            }}
          />
        </>
      )}
    </>
  );
}
export function ComparePage() {
  const jobs = useResource<Job[]>("/jobs"),
    [selected, setSelected] = useState<string[]>([]),
    [results, setResults] = useState<Simulation[]>([]),
    [busy, setBusy] = useState(false),
    [error, setError] = useState("");
  useEffect(() => {
    if (!selected.length) {
      setResults([]);
      return;
    }
    let active = true;
    setBusy(true);
    Promise.all(
      selected.map((jobId) =>
        api<Simulation>("/simulate", "POST", {
          jobId,
          months: 6,
          addedSkills: [],
        }),
      ),
    )
      .then((r) => {
        if (active) setResults(r);
      })
      .catch((e) => {
        if (active) setError(e.message);
      })
      .finally(() => {
        if (active) setBusy(false);
      });
    return () => {
      active = false;
    };
  }, [selected]);
  return (
    <>
      <PageTitle
        title="어떤 미래가 더 나다울까요?"
        description="최대 세 가지 목표 직무를 나란히 놓고, 전환의 거리와 경로를 비교하세요."
      />
      <Panel>
        <MultiSelect
          label="비교할 직무 선택"
          placeholder="관심 있는 직무를 최대 3개 선택하세요"
          data={(jobs.data || []).map((j) => ({ value: j.id, label: j.title }))}
          maxValues={3}
          value={selected}
          onChange={setSelected}
          searchable
          size="md"
        />
      </Panel>
      <Loading error={error || jobs.error} loading={busy} />
      {!selected.length && (
        <Empty
          title="두세 가지 가능성을 나란히 비교해 보세요"
          description="위에서 관심 있는 목표 직무를 선택하면 적합도, 역량 차이와 예상 경로가 나타납니다."
        />
      )}
      <div className="compare-grid">
        {results.map((sim) => (
          <Panel key={sim.job.id} className="compare-card">
            <Badge variant="light">{sim.job.category}</Badge>
            <h2>{sim.job.title}</h2>
            <Score value={sim.score} />
            <div className="comparison-metric">
              <span>전환 난이도</span>
              <strong>{Math.round(sim.difficulty)} / 100</strong>
            </div>
            <div className="comparison-metric">
              <span>예상 준비 기간</span>
              <strong>{sim.estimatedMonths}개월</strong>
            </div>
            <div className="comparison-metric">
              <span>보완할 역량</span>
              <strong>{sim.gaps.filter((g) => g.gap > 0).length}개</strong>
            </div>
            <h3>이어지는 강점</h3>
            <div className="tag-row">
              {sim.strengths.map((s) => (
                <span key={s} className="skill-tag">
                  {s}
                </span>
              ))}
            </div>
            <h3>우선 학습 역량</h3>
            {sim.roi.slice(0, 3).map((r) => (
              <div className="roi-row" key={r.name}>
                <span>{r.name}</span>
                <Badge color="teal">+{Math.round(r.gain)}점</Badge>
              </div>
            ))}
            <SourceTag source={sim.source} />
            <Button
              fullWidth
              mt="xl"
              component={Link}
              to={`/simulator?job=${sim.job.id}`}
              rightSection={<IconArrowUpRight size={18} />}
            >
              이 경로 실험하기
            </Button>
          </Panel>
        ))}
      </div>
    </>
  );
}
export function RoadmapPage() {
  const s = useSimulation(),
    progress = useResource<{ completed: string[] }>("/roadmap"),
    { notify } = useApp();
  const completed = progress.data?.completed || [];
  const tasks =
    s.simulation?.plan.flatMap((p) =>
      p.tasks.map((task, i) => ({ id: `${s.jobId}:${p.month}:${i}`, task })),
    ) || [];
  async function toggle(id: string, checked: boolean) {
    const next = checked
      ? [...completed, id]
      : completed.filter((x) => x !== id);
    try {
      await api("/roadmap", "PUT", { completed: next });
      progress.setData({ completed: next });
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  const done = tasks.filter((t) => completed.includes(t.id)).length;
  return (
    <>
      <PageTitle
        title="한 걸음씩, 다음 커리어로"
        description="큰 목표를 작은 실행으로 나누고, 성장의 발자국을 기록하세요."
      />
      <div className="simulation-toolbar">
        <Select
          label="목표 직무"
          value={s.jobId}
          data={(s.jobs.data || []).map((j) => ({
            value: j.id,
            label: j.title,
          }))}
          onChange={s.selectJob}
          searchable
        />
        <div>
          <label className="field-label">계획 기간</label>
          <SegmentedControl
            value={s.months}
            onChange={s.setMonths}
            data={[
              { value: "3", label: "3개월" },
              { value: "6", label: "6개월" },
              { value: "12", label: "12개월" },
            ]}
            size="md"
          />
        </div>
      </div>
      <Loading loading={s.loading} error={s.error || progress.error} />
      {s.simulation && (
        <>
          <div className="roadmap-progress">
            <div>
              <span className="eyebrow">MY PROGRESS</span>
              <h2>
                {done === 0
                  ? "첫 번째 작은 성취를 시작해요"
                  : `${done}개의 작은 성취가 쌓였어요`}
              </h2>
              <p>{s.simulation.job.title}를 향한 나의 실행 계획</p>
            </div>
            <div>
              <strong>
                {tasks.length ? Math.round((done / tasks.length) * 100) : 0}
                <small>%</small>
              </strong>
              <Progress
                value={tasks.length ? (done / tasks.length) * 100 : 0}
                size="lg"
                color="teal"
              />
              <span>
                {done} / {tasks.length}개 실행 항목 완료
              </span>
            </div>
          </div>
          <div className="roadmap-timeline">
            {s.simulation.plan.map((plan, idx) => (
              <div className="timeline-row" key={plan.month}>
                <div className="timeline-marker">
                  <span>{String(idx + 1).padStart(2, "0")}</span>
                  <small>{plan.month}개월차</small>
                </div>
                <Panel>
                  <div className="panel-title">
                    <h2>{plan.title}</h2>
                    <Badge variant="light">{plan.month}개월차</Badge>
                  </div>
                  <div className="tag-row">
                    {plan.skills.map((skill) => (
                      <span className="skill-tag" key={skill}>
                        {skill}
                      </span>
                    ))}
                  </div>
                  <div className="task-list">
                    {plan.tasks.map((task, i) => (
                      <Checkbox
                        key={i}
                        checked={completed.includes(
                          `${s.jobId}:${plan.month}:${i}`,
                        )}
                        onChange={(e) =>
                          toggle(
                            `${s.jobId}:${plan.month}:${i}`,
                            e.target.checked,
                          )
                        }
                        label={task}
                        size="md"
                      />
                    ))}
                  </div>
                </Panel>
              </div>
            ))}
          </div>
          <AIExplanation
            task="plan"
            jobId={s.jobId}
            scenario={{
              months: Number(s.months),
              addedSkills: s.added,
              project: s.project,
              certifications: s.certifications
                .split(",")
                .map((v) => v.trim())
                .filter(Boolean),
            }}
          />
        </>
      )}
    </>
  );
}
export function OpportunitiesPage() {
  const jobs = useResource<Job[]>("/jobs"),
    [params, setParams] = useSearchParams(),
    [jobId, setJobId] = useState(params.get("job") || ""),
    [tab, setTab] = useState("training");
  useEffect(() => {
    if (!jobId && jobs.data?.length) setJobId(jobs.data[0].id);
  }, [jobs.data, jobId]);
  const opportunities = useResource<any>(
    jobId ? `/opportunities?jobId=${encodeURIComponent(jobId)}` : null,
  );
  const rows = opportunities.data?.[tab] || [];
  return (
    <>
      <PageTitle
        title="배움에서, 다음 기회까지"
        description="목표 직무와 부족 역량에 맞는 교육과 채용 정보를 함께 살펴보세요."
      />
      <div className="simulation-toolbar">
        <Select
          label="관심 직무"
          value={jobId}
          data={(jobs.data || []).map((j) => ({ value: j.id, label: j.title }))}
          onChange={(v) => {
            setJobId(v || "");
            setParams({ job: v || "" });
          }}
          searchable
        />
        <SegmentedControl
          value={tab}
          onChange={setTab}
          data={[
            { value: "training", label: "교육 · 훈련" },
            { value: "jobs", label: "채용 기회" },
          ]}
          size="md"
        />
      </div>
      {opportunities.data?.synthetic && (
        <Alert color="orange" title="합성 데이터로 살펴보는 데모">
          현재 표시되는 예시는 실제 모집 중인 과정이나 채용 공고가 아닙니다.
          관리자가 데이터 연동을 설정하면 실제 출처가 있는 정보를 확인할 수
          있습니다.
        </Alert>
      )}
      <Loading loading={opportunities.loading} error={opportunities.error} />
      <div className="section-heading">
        <h2>
          {tab === "training"
            ? "다음 역량을 만드는 배움"
            : "새로운 커리어 기회"}
        </h2>
        <span>{rows.length}개</span>
      </div>
      {rows.length === 0 && !opportunities.loading && (
        <Empty
          title="아직 연결된 정보가 없습니다"
          description="관리자가 고용24, API 또는 데이터베이스 연동을 설정하면 교육·채용 정보를 확인할 수 있습니다."
        />
      )}
      <div className="opportunity-grid">
        {rows.map((item: any) => (
          <Panel key={item.id}>
            <div className="opportunity-top">
              <Badge
                color={tab === "training" ? "teal" : "blue"}
                variant="light"
              >
                {tab === "training" ? "교육 · 훈련" : "채용"}
              </Badge>
              <span>{item.region}</span>
            </div>
            <h3>{item.title}</h3>
            <span className="organization">{item.organization}</span>
            <p>{item.description}</p>
            <div className="tag-row">
              {(item.skills || []).map((sk: string) => (
                <span className="skill-tag" key={sk}>
                  {sk}
                </span>
              ))}
            </div>
            {item.cost !== undefined && (
              <p className="opportunity-meta">교육비 {number(item.cost)}원</p>
            )}
            {item.salary && (
              <p className="opportunity-meta">급여 {item.salary}</p>
            )}
            {item.deadline && (
              <p className="opportunity-meta">마감 {item.deadline}</p>
            )}
            <SourceTag source={item.source} />
            {item.url && (
              <Button
                mt="lg"
                fullWidth
                variant="light"
                component="a"
                href={item.url}
                target="_blank"
                rel="noreferrer"
                rightSection={<IconArrowUpRight size={18} />}
              >
                {item.source?.synthetic
                  ? "참고 출처 확인"
                  : "원문에서 자세히 보기"}
              </Button>
            )}
          </Panel>
        ))}
      </div>
      <MarketInsights jobId={jobId} />
      {opportunities.data?.skillFrequency?.length > 0 && (
        <Panel title="연결된 채용 정보에서 자주 찾는 역량">
          <div className="frequency-list">
            {opportunities.data.skillFrequency.map((s: any) => (
              <span key={s.name} className="frequency-chip">
                {s.name}
                <b>{s.count}건</b>
              </span>
            ))}
          </div>
        </Panel>
      )}
    </>
  );
}
export function SavedPage() {
  const resource = useResource<Simulation[]>("/simulations"),
    { notify, publicInfo } = useApp(),
    [remove, setRemove] = useState<string | null>(null),
    [note, setNote] = useState(""),
    [approval, setApproval] = useState<string | null>(null);
  async function deleteSim() {
    try {
      await api(`/simulations/${remove}`, "DELETE");
      setRemove(null);
      notify("저장한 시뮬레이션을 삭제했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  async function requestApproval() {
    try {
      await api("/approvals", "POST", { simulationId: approval, note });
      setApproval(null);
      setNote("");
      notify("검토를 요청했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageTitle
        title="기록해 둔 나의 가능성"
        description="저장한 시뮬레이션을 다시 살펴보고, 다음 선택을 구체화하세요."
        action={
          <Button
            component={Link}
            to="/simulator"
            leftSection={<IconPlus size={18} />}
          >
            새 시뮬레이션
          </Button>
        }
      />
      <Loading error={resource.error} loading={resource.loading} />
      {resource.data?.length === 0 && (
        <Empty
          title="첫 번째 가능성을 저장해 보세요"
          description="시뮬레이터에서 역량과 목표를 바꾸고, 마음에 드는 경로를 저장할 수 있어요."
          action={
            <Button component={Link} to="/simulator">
              시뮬레이션 시작하기
            </Button>
          }
        />
      )}
      <div className="saved-grid">
        {resource.data?.map((sim) => (
          <Panel key={sim.id}>
            <div className="panel-title">
              <Badge variant="light">{sim.job.category}</Badge>
              <button
                className="icon-button"
                aria-label={`${sim.job.title} 시뮬레이션 삭제`}
                onClick={() => setRemove(sim.id!)}
              >
                <IconTrash size={18} />
              </button>
            </div>
            <h2>{sim.job.title}</h2>
            <div className="saved-score">
              {Math.round(sim.score)}
              <small>점</small>
              <span>
                현재 대비 +{Math.round(sim.score - sim.baselineScore)}점
              </span>
            </div>
            <p>{sim.explanation}</p>
            <small className="muted">
              {sim.createdAt
                ? new Date(sim.createdAt).toLocaleString("ko-KR")
                : ""}
            </small>
            <div className="button-row">
              <Button
                component={Link}
                to={`/simulator?job=${sim.job.id}&saved=${sim.id}`}
                variant="light"
              >
                저장한 조건으로 다시 계산
              </Button>
              {publicInfo.approvalEnabled && (
                <Button variant="default" onClick={() => setApproval(sim.id!)}>
                  검토 요청
                </Button>
              )}
            </div>
          </Panel>
        ))}
      </div>
      <Modal
        opened={!!remove}
        onClose={() => setRemove(null)}
        title="시뮬레이션 삭제"
        centered
      >
        <p>선택한 시뮬레이션을 삭제하시겠어요?</p>
        <Button color="red" onClick={deleteSim}>
          삭제
        </Button>
      </Modal>
      <Modal
        opened={!!approval}
        onClose={() => setApproval(null)}
        title="팀장 검토 요청"
        centered
      >
        <Textarea
          label="검토 요청 메모"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        <Button mt="md" onClick={requestApproval}>
          검토 요청 보내기
        </Button>
      </Modal>
    </>
  );
}
export function KeysPage() {
  const keys = useResource<any[]>("/keys"),
    scopes = useResource<{
      available: string[];
      allowed: string[];
      maxDays?: number;
    }>("/key-scopes"),
    { notify } = useApp(),
    [editing, setEditing] = useState<any | null>(null),
    [secret, setSecret] = useState(""),
    [action, setAction] = useState<{ id: string; type: string } | null>(null),
    [busy, setBusy] = useState(false);
  async function save() {
    setBusy(true);
    try {
      const data = await api(
        editing.id ? `/keys/${editing.id}` : "/keys",
        editing.id ? "PUT" : "POST",
        editing,
      );
      if (data.key) setSecret(data.key);
      setEditing(null);
      notify("API 키 설정을 저장했습니다.");
      void keys.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  async function execute() {
    if (!action) return;
    setBusy(true);
    try {
      const data = await api(
        `/keys/${action.id}${action.type === "rotate" ? "/rotate" : ""}`,
        action.type === "rotate" ? "POST" : "DELETE",
      );
      if (data.key) setSecret(data.key);
      notify(
        action.type === "rotate"
          ? "새 키를 발급하고 기존 키를 폐기했습니다."
          : "API 키를 폐기했습니다.",
      );
      setAction(null);
      void keys.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        title="나의 연결을 안전하게 관리"
        description="개인별 API 키의 권한과 유효기간을 설정하고, 필요할 때 키를 회전하세요."
        action={
          <Button
            leftSection={<IconPlus size={18} />}
            onClick={() =>
              setEditing({
                name: "",
                scopes: scopes.data?.allowed?.slice(0, 1) || [],
                expiresInDays: Math.min(90, scopes.data?.maxDays || 90),
              })
            }
          >
            API 키 만들기
          </Button>
        }
      />
      <Alert
        color="teal"
        icon={<IconKey />}
        title="필요한 만큼의 권한만 부여하세요"
      >
        키 원문은 발급 또는 회전 직후 한 번만 표시됩니다. 키를 회전하면 이전
        키는 즉시 사용할 수 없습니다.
      </Alert>
      <Loading loading={keys.loading} error={keys.error || scopes.error} />
      <Panel title="내 API 키">
        {keys.data?.length === 0 ? (
          <Empty
            title="발급한 API 키가 없습니다"
            description="API 또는 MCP 연결에 사용할 개인 키를 만들어 보세요."
          />
        ) : (
          <div className="table-scroll">
            <Table verticalSpacing="md">
              <Table.Thead>
                <Table.Tr>
                  <Table.Th>이름 · 식별자</Table.Th>
                  <Table.Th>권한</Table.Th>
                  <Table.Th>만료일</Table.Th>
                  <Table.Th>상태</Table.Th>
                  <Table.Th>관리</Table.Th>
                </Table.Tr>
              </Table.Thead>
              <Table.Tbody>
                {keys.data?.map((k) => (
                  <Table.Tr key={k.id}>
                    <Table.Td>
                      <strong>{k.name}</strong>
                      <small className="block muted">{k.prefix}…</small>
                    </Table.Td>
                    <Table.Td>
                      <div className="tag-row">
                        {k.scopes.map((s: string) => (
                          <Badge key={s} variant="light" color="gray">
                            {s}
                          </Badge>
                        ))}
                      </div>
                    </Table.Td>
                    <Table.Td>
                      {k.expiresAt
                        ? new Date(k.expiresAt).toLocaleDateString("ko-KR")
                        : "없음"}
                    </Table.Td>
                    <Table.Td>
                      <Badge
                        color={
                          k.revokedAt ||
                          new Date(k.expiresAt).getTime() < Date.now()
                            ? "gray"
                            : "teal"
                        }
                      >
                        {k.revokedAt
                          ? "폐기됨"
                          : new Date(k.expiresAt).getTime() < Date.now()
                            ? "만료됨"
                            : "활성"}
                      </Badge>
                    </Table.Td>
                    <Table.Td>
                      <div className="button-row compact">
                        <Button
                          size="xs"
                          variant="subtle"
                          disabled={
                            !!k.revokedAt ||
                            new Date(k.expiresAt).getTime() < Date.now()
                          }
                          onClick={() => setEditing({ ...k })}
                        >
                          권한 수정
                        </Button>
                        <Button
                          size="xs"
                          variant="subtle"
                          disabled={
                            !!k.revokedAt ||
                            new Date(k.expiresAt).getTime() < Date.now()
                          }
                          onClick={() =>
                            setAction({ id: k.id, type: "rotate" })
                          }
                        >
                          회전
                        </Button>
                        <Button
                          size="xs"
                          variant="subtle"
                          color="red"
                          disabled={
                            !!k.revokedAt ||
                            new Date(k.expiresAt).getTime() < Date.now()
                          }
                          onClick={() =>
                            setAction({ id: k.id, type: "revoke" })
                          }
                        >
                          폐기
                        </Button>
                      </div>
                    </Table.Td>
                  </Table.Tr>
                ))}
              </Table.Tbody>
            </Table>
          </div>
        )}
      </Panel>
      <Modal
        opened={!!editing}
        onClose={() => setEditing(null)}
        title={editing?.id ? "API 키 권한 수정" : "개인 API 키 만들기"}
        centered
      >
        {editing && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              void save();
            }}
            className="form-stack"
          >
            <TextInput
              required
              label="키 이름"
              placeholder="예: 개인 MCP 클라이언트"
              value={editing.name}
              onChange={(e) => setEditing({ ...editing, name: e.target.value })}
            />
            <MultiSelect
              required
              label="허용 권한"
              data={scopes.data?.allowed || []}
              value={editing.scopes}
              onChange={(v) => setEditing({ ...editing, scopes: v })}
            />
            {!editing.id && (
              <NumberInput
                required
                label="유효기간 (일)"
                min={1}
                max={scopes.data?.maxDays || 365}
                value={editing.expiresInDays}
                onChange={(v) =>
                  setEditing({ ...editing, expiresInDays: Number(v) })
                }
              />
            )}
            <Button type="submit" loading={busy}>
              저장
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        opened={!!secret}
        onClose={() => setSecret("")}
        title="새 API 키 · 한 번만 표시됩니다"
        centered
      >
        <Alert color="yellow">
          안전한 곳에 복사해 보관하세요. 이 창을 닫으면 키 원문을 다시 볼 수
          없습니다.
        </Alert>
        <pre className="code-block secret">{secret}</pre>
        <Button
          leftSection={<IconCopy size={17} />}
          onClick={() =>
            navigator.clipboard
              .writeText(secret)
              .then(() => notify("키를 복사했습니다."))
              .catch(() =>
                notify(
                  "클립보드 접근이 안 됩니다. 키를 직접 선택해 복사해 주세요.",
                  "error",
                ),
              )
          }
        >
          키 복사
        </Button>
      </Modal>
      <Modal
        opened={!!action}
        onClose={() => setAction(null)}
        title={action?.type === "rotate" ? "키 회전 확인" : "키 폐기 확인"}
        centered
      >
        <p>
          기존 키로 연결된 서비스는 즉시 인증이 중단됩니다.
          {action?.type === "rotate"
            ? " 새 키로 연결 정보를 교체해 주세요."
            : ""}
        </p>
        <Button
          color={action?.type === "rotate" ? "teal" : "red"}
          onClick={execute}
          loading={busy}
        >
          {action?.type === "rotate" ? "새 키 발급 및 기존 키 폐기" : "키 폐기"}
        </Button>
      </Modal>
    </>
  );
}
export function PreferencesPage() {
  const { user, setUser, notify } = useApp(),
    [name, setName] = useState(user.name || ""),
    [preferences, setPreferences] = useState(user.preferences || {}),
    [current, setCurrent] = useState(""),
    [password, setPassword] = useState(""),
    [confirm, setConfirm] = useState(""),
    [busy, setBusy] = useState("");
  async function save(e: React.FormEvent) {
    e.preventDefault();
    setBusy("save");
    try {
      const data = await api("/me", "PUT", { name, preferences });
      setUser({ ...user, ...data });
      notify("개인 설정을 저장했습니다.");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function changePassword(e: React.FormEvent) {
    e.preventDefault();
    if (password !== confirm)
      return notify("새 비밀번호가 일치하지 않습니다.", "error");
    setBusy("password");
    try {
      await api("/me/password", "POST", {
        currentPassword: current,
        newPassword: password,
      });
      notify("비밀번호를 변경했습니다.");
      setCurrent("");
      setPassword("");
      setConfirm("");
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <PageTitle
        title="나에게 맞는 경력 공간"
        description="개인 프로필과 학습 선호, 계정 보안을 관리하세요."
      />
      <div className="two-col">
        <Panel title="프로필 · 개인화">
          <form className="form-stack" onSubmit={save}>
            <TextInput label="이메일" value={user.email} readOnly />
            <TextInput
              required
              label="이름"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
            <Select
              label="기본 시뮬레이션 기간"
              value={String(preferences.defaultMonths || 6)}
              data={[
                { value: "3", label: "3개월" },
                { value: "6", label: "6개월" },
                { value: "12", label: "12개월" },
              ]}
              onChange={(v) =>
                setPreferences({ ...preferences, defaultMonths: Number(v) })
              }
            />
            <TextInput
              label="선호 학습 방식"
              placeholder="예: 프로젝트 중심, 온라인 강의"
              value={preferences.learningStyle || ""}
              onChange={(e) =>
                setPreferences({
                  ...preferences,
                  learningStyle: e.target.value,
                })
              }
            />
            <Switch
              label="대시보드 경력 인사이트 안내 표시"
              checked={preferences.showInsights !== false}
              onChange={(e) =>
                setPreferences({
                  ...preferences,
                  showInsights: e.target.checked,
                })
              }
            />
            <Button type="submit" loading={busy === "save"}>
              개인 설정 저장
            </Button>
          </form>
        </Panel>
        <Panel title="비밀번호 변경">
          <form className="form-stack" onSubmit={changePassword}>
            <PasswordInput
              required
              label="현재 비밀번호"
              autoComplete="current-password"
              value={current}
              onChange={(e) => setCurrent(e.target.value)}
            />
            <PasswordInput
              required
              minLength={12}
              label="새 비밀번호"
              description="12자 이상으로 설정해 주세요."
              autoComplete="new-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
            <PasswordInput
              required
              label="새 비밀번호 확인"
              autoComplete="new-password"
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
            <Button variant="light" type="submit" loading={busy === "password"}>
              비밀번호 변경
            </Button>
          </form>
        </Panel>
      </div>
    </>
  );
}
function ApprovalSnapshot({ sim }: { sim: Simulation }) {
  return (
    <div className="approval-snapshot">
      <h3>
        {sim.job?.title} · {Math.round(sim.score)}점
      </h3>
      <p>{sim.explanation}</p>
      <GapChart sim={sim} />
      {sim.scenario && (
        <p className="muted">
          목표 준비 기간 {sim.scenario.months}개월 · 추가 역량{" "}
          {(sim.scenario.addedSkills || []).map((s) => s.name).join(", ") ||
            "없음"}
        </p>
      )}
      <h3>검토할 실행 계획</h3>
      {(sim.plan || []).map((plan) => (
        <div key={plan.month}>
          <strong>
            {plan.month}개월차 · {plan.title}
          </strong>
          <ul>
            {plan.tasks.map((task, i) => (
              <li key={i}>{task}</li>
            ))}
          </ul>
        </div>
      ))}
    </div>
  );
}
export function ApprovalsPage() {
  const resource = useResource<any[]>("/approvals"),
    { user, notify } = useApp(),
    [review, setReview] = useState<any>(null),
    [note, setNote] = useState("");
  async function submit(status: string) {
    try {
      await api(`/approvals/${review.id}`, "PUT", { status, note });
      setReview(null);
      setNote("");
      notify(status === "approved" ? "승인했습니다." : "반려했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    }
  }
  return (
    <>
      <PageTitle
        title="함께 검토하는 다음 단계"
        description="요청한 경력전환 계획과 검토 의견을 확인하세요."
      />
      <Loading loading={resource.loading} error={resource.error} />
      {resource.data?.length === 0 && (
        <Empty
          title="아직 검토 요청이 없습니다"
          description="저장한 시뮬레이션에서 검토 요청을 시작할 수 있습니다."
        />
      )}
      <div className="job-list">
        {resource.data?.map((a) => (
          <Panel key={a.id}>
            <div className="panel-title">
              <h2>{a.jobTitle || "경력전환 계획 검토"}</h2>
              <Badge
                color={
                  a.status === "approved"
                    ? "teal"
                    : a.status === "rejected"
                      ? "red"
                      : "orange"
                }
              >
                {(
                  {
                    pending: "검토 대기",
                    approved: "승인",
                    rejected: "반려",
                  } as any
                )[a.status] || a.status}
              </Badge>
            </div>
            <p>
              <strong>{a.userName || "요청자"}</strong> ·{" "}
              {a.note || "검토 요청 메모가 없습니다."}
            </p>
            {a.simulation && (
              <div className="approval-summary">
                <Badge variant="light">
                  적합도 {Math.round(a.simulation.score)}점
                </Badge>
                <Badge variant="light" color="gray">
                  예상 {a.simulation.estimatedMonths}개월
                </Badge>
                <span>
                  보완할 역량{" "}
                  {
                    (a.simulation.gaps || []).filter((g: any) => g.gap > 0)
                      .length
                  }
                  개
                </span>
                <details>
                  <summary>요청한 시뮬레이션 상세</summary>
                  <ApprovalSnapshot sim={a.simulation} />
                </details>
              </div>
            )}
            <small className="muted">
              요청일 {new Date(a.createdAt).toLocaleString("ko-KR")}
            </small>
            {a.reviewNote && <p>검토 의견: {a.reviewNote}</p>}
            {["admin", "reviewer"].includes(user.role) &&
              a.status === "pending" &&
              a.userId !== user.id && (
                <Button mt="md" onClick={() => setReview(a)}>
                  요청 검토
                </Button>
              )}
          </Panel>
        ))}
      </div>
      <Modal
        opened={!!review}
        onClose={() => setReview(null)}
        title="계획 검토"
        size="lg"
        centered
      >
        <p>
          <strong>{review?.userName}</strong> · {review?.note}
        </p>
        {review?.simulation && <ApprovalSnapshot sim={review.simulation} />}
        <Textarea
          label="검토 의견"
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        <div className="button-row">
          <Button onClick={() => submit("approved")}>승인</Button>
          <Button
            color="red"
            variant="light"
            onClick={() => submit("rejected")}
          >
            반려
          </Button>
        </div>
      </Modal>
    </>
  );
}
export function APIPage() {
  const { notify } = useApp();
  const origin = window.location.origin;
  const curl = `curl ${origin}/api/v1/simulate \\\n  -H "Authorization: Bearer YOUR_API_KEY" \\\n  -H "Content-Type: application/json" \\\n  -d '{"jobId":"ai-platform","months":6,"addedSkills":[{"name":"Python","level":4}]}'`;
  const mcp = JSON.stringify(
    {
      mcpServers: {
        nextrole: {
          url: `${origin}/mcp`,
          headers: { Authorization: "Bearer YOUR_API_KEY" },
        },
      },
    },
    null,
    2,
  );
  return (
    <>
      <PageTitle
        title="NextRole를 나의 도구와 연결"
        description="REST API와 MCP를 통해 경력 분석, 직무 검색, 시뮬레이션을 외부 도구에서 활용하세요."
      />
      <div className="two-col">
        <Panel title="REST API">
          <p>
            개인 API 키를 발급하고 필요한 권한을 부여한 다음, Bearer 인증으로
            호출하세요.
          </p>
          <div className="endpoint">
            <Badge>POST</Badge>
            <code>/api/v1/simulate</code>
          </div>
          <pre className="code-block">{curl}</pre>
          <Button
            variant="light"
            component="a"
            href="/api/v1/openapi.json"
            target="_blank"
            rightSection={<IconDownload size={18} />}
          >
            OpenAPI 명세 보기
          </Button>
        </Panel>
        <Panel title="MCP 연결">
          <p>
            HTTP MCP 클라이언트의 서버 설정에 NextRole 주소와 개인 API 키를
            추가하세요.
          </p>
          <pre className="code-block">{mcp}</pre>
          <Button
            variant="light"
            onClick={() =>
              navigator.clipboard
                .writeText(mcp)
                .then(() => notify("MCP 설정 예시를 복사했습니다."))
                .catch(() => notify("설정 내용을 직접 복사해 주세요.", "error"))
            }
            leftSection={<IconCopy size={18} />}
          >
            연결 설정 복사
          </Button>
        </Panel>
      </div>
      <Panel title="MCP에서 사용할 수 있는 도구">
        <div className="table-scroll">
          <Table>
            <Table.Thead>
              <Table.Tr>
                <Table.Th>도구</Table.Th>
                <Table.Th>기능</Table.Th>
                <Table.Th>필요 권한</Table.Th>
              </Table.Tr>
            </Table.Thead>
            <Table.Tbody>
              {[
                ["career_profile", "내 경력 프로필 조회", "profile:read"],
                ["job_search", "직무 검색 (query)", "jobs:read"],
                [
                  "career_simulate",
                  "역량·기간 조건 시뮬레이션",
                  "simulate:write",
                ],
                ["career_opportunities", "교육·채용 기회 조회", "jobs:read"],
              ].map((row) => (
                <Table.Tr key={row[0]}>
                  {row.map((cell) => (
                    <Table.Td key={cell}>{cell}</Table.Td>
                  ))}
                </Table.Tr>
              ))}
            </Table.Tbody>
          </Table>
        </div>
        <p className="muted">
          모든 MCP 도구 호출에 mcp:use 권한이 추가로 필요합니다. 개인 API
          키에서는 관리자 설정과 키 관리 작업을 허용하지 않습니다.
        </p>
      </Panel>
      <Panel title="활용 순서">
        <div className="api-steps">
          <div>
            <span>01</span>
            <h3>개인 키 발급</h3>
            <p>개인 API 키 메뉴에서 사용 목적과 권한, 유효기간을 설정합니다.</p>
            <Link to="/keys">키 관리로 이동 ↗</Link>
          </div>
          <div>
            <span>02</span>
            <h3>명세와 도구 확인</h3>
            <p>
              OpenAPI 명세 또는 MCP tools/list에서 사용 가능한 요청과 도구를
              확인합니다.
            </p>
          </div>
          <div>
            <span>03</span>
            <h3>안전하게 연결</h3>
            <p>
              Bearer 인증으로 요청합니다. 키를 정기적으로 회전하고 연결 정보를
              갱신하세요.
            </p>
          </div>
        </div>
      </Panel>
    </>
  );
}
