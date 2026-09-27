import { useEffect, useState } from "react";
import { Link } from "react-router-dom";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Modal,
  MultiSelect,
  NumberInput,
  Select,
  Table,
  TagsInput,
  Textarea,
  TextInput,
} from "@mantine/core";
import { useDebouncedValue } from "@mantine/hooks";
import {
  IconArrowUpRight,
  IconCheck,
  IconDatabase,
  IconHistory,
  IconPlus,
  IconSearch,
  IconTrash,
} from "@tabler/icons-react";
import { api, Source } from "./api";
import {
  Empty,
  Loading,
  PageTitle,
  Panel,
  SourceTag,
  number,
  sourceKinds,
  useApp,
  useResource,
} from "./App";
export const datasetOptions = [
  { value: "occupation_details", label: "직업정보 상세 원천" },
  { value: "ncs_units", label: "NCS 능력단위 원천" },
  { value: "jobs", label: "채용 공고" },
  { value: "training", label: "훈련 과정" },
  { value: "occupations", label: "직무 모델" },
];
type SourceRecord = {
  id: string;
  dataset: string;
  title: string;
  source: Source;
  raw?: Record<string, any>;
  occupationCode?: string;
  [key: string]: any;
};
type Records = {
  records: SourceRecord[];
  total: number;
  offset: number;
  limit: number;
};
type MappingSkill = {
  internalSkillId: string;
  name: string;
  aliases: string[];
  level: number;
  weight: number;
  sourceRecordIds: string[];
  basis: string;
};
type Mapping = {
  id?: string;
  title: string;
  occupationRecordId: string;
  occupationCode: string;
  jobCodes?: string[];
  valid?: boolean;
  validationWarning?: string;
  ncsRecordIds: string[];
  skills: MappingSkill[];
  basis: string;
  version: number;
  status: "draft" | "published";
  reviewedBy?: string;
  reviewedAt?: string;
  updatedAt?: string;
};
const newMapping = (): Mapping => ({
  title: "",
  occupationRecordId: "",
  occupationCode: "",
  jobCodes: [],
  ncsRecordIds: [],
  skills: [],
  basis: "",
  version: 1,
  status: "draft",
});
export function DataSourcesPage() {
  const overview = useResource<any>("/admin/data"),
    [dataset, setDataset] = useState("occupation_details"),
    [query, setQuery] = useState(""),
    [search, setSearch] = useState(""),
    [offset, setOffset] = useState(0),
    [record, setRecord] = useState<SourceRecord | null>(null),
    [remove, setRemove] = useState<SourceRecord | null>(null),
    [busy, setBusy] = useState(false),
    { notify } = useApp();
  const rows = useResource<Records>(
    `/admin/source-records?dataset=${dataset}&q=${encodeURIComponent(search)}&offset=${offset}&limit=25`,
  );
  async function del() {
    if (!remove) return;
    setBusy(true);
    try {
      await api(
        `/admin/source-records/${encodeURIComponent(remove.dataset || dataset)}/${encodeURIComponent(remove.id)}`,
        "DELETE",
      );
      setRemove(null);
      setRecord(null);
      notify("선택한 원천 데이터를 삭제했습니다.");
      void rows.reload();
      void overview.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <PageTitle
        eyebrow="DATA PROVENANCE"
        title="원천과 가공을 분리해 확인"
        description="공공 원천, 이용자 입력, 합성 데이터와 검토한 가공 데이터를 출처별로 관리하세요."
        action={
          <Button
            component={Link}
            to="/admin/connectors"
            variant="light"
            leftSection={<IconDatabase size={18} />}
          >
            데이터 연동 설정
          </Button>
        }
      />
      <Alert color="blue" title="원천 자료와 내부 분석값은 서로 다릅니다">
        공공 API의 문자열·코드·업무 설명을 내부 0~5 역량 수준으로 간주하지
        않습니다. 출처가 있는 원천 자료는 별도로 보관하고, 관리자 검토를 거친
        매핑만 시뮬레이션 계산에 사용합니다.
      </Alert>
      <Loading loading={overview.loading} error={overview.error} />
      {overview.data && (
        <>
          <div className="stat-grid">
            <div className="stat-card">
              <div>
                <span>보관한 원천·데이터 항목</span>
                <strong>
                  {number(
                    (overview.data.datasets || []).reduce(
                      (sum: number, x: any) => sum + x.count,
                      0,
                    ),
                  )}
                  <small>개</small>
                </strong>
              </div>
            </div>
            <div className="stat-card">
              <div>
                <span>전체 역량 매핑</span>
                <strong>
                  {number(overview.data.mappingCount)}
                  <small>개</small>
                </strong>
              </div>
            </div>
            <div className="stat-card">
              <div>
                <span>검토·게시된 매핑</span>
                <strong>
                  {number(overview.data.publishedMappingCount)}
                  <small>개</small>
                </strong>
              </div>
            </div>
          </div>
          <Panel title="출처별 데이터 현황">
            <div className="source-summary-grid">
              {(overview.data.datasets || []).map((s: any, i: number) => (
                <button
                  key={`${s.dataset}-${s.kind}-${i}`}
                  className="source-summary"
                  onClick={() => {
                    setDataset(s.dataset);
                    setOffset(0);
                  }}
                >
                  <Badge
                    color={sourceKinds[s.kind]?.color || "gray"}
                    variant="light"
                  >
                    {sourceKinds[s.kind]?.label || s.kind || "외부 데이터"}
                  </Badge>
                  <strong>
                    {datasetOptions.find((d) => d.value === s.dataset)?.label ||
                      s.dataset}
                  </strong>
                  <span>
                    {s.provider || "제공자 정보 없음"} · {number(s.count)}개
                  </span>
                  {s.updatedAt && (
                    <small>
                      최근 갱신 {new Date(s.updatedAt).toLocaleString("ko-KR")}
                    </small>
                  )}
                </button>
              ))}
            </div>
            {!overview.data.datasets?.length && (
              <p>
                등록된 원천 데이터가 없습니다. API 또는 데이터베이스 연결을 먼저
                설정하세요.
              </p>
            )}
          </Panel>
        </>
      )}
      <Panel title="보관한 원천 기록">
        <div className="data-filter-row">
          <Select
            label="데이터 종류"
            value={dataset}
            data={datasetOptions}
            onChange={(v) => {
              setDataset(v || "occupation_details");
              setOffset(0);
            }}
          />
          <form
            onSubmit={(e) => {
              e.preventDefault();
              setSearch(query);
              setOffset(0);
            }}
          >
            <TextInput
              label="원천 검색"
              placeholder="명칭이나 식별번호 검색"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <Button type="submit" leftSection={<IconSearch size={17} />}>
              검색
            </Button>
          </form>
        </div>
        <Loading loading={rows.loading} error={rows.error} />
        {rows.data?.records?.length ? (
          <>
            <div className="table-scroll">
              <Table verticalSpacing="md">
                <Table.Thead>
                  <Table.Tr>
                    <Table.Th>원천 명칭 · 식별번호</Table.Th>
                    <Table.Th>구분 · 출처</Table.Th>
                    <Table.Th>수집 시각</Table.Th>
                    <Table.Th>관리</Table.Th>
                  </Table.Tr>
                </Table.Thead>
                <Table.Tbody>
                  {rows.data.records.map((r) => (
                    <Table.Tr key={r.id}>
                      <Table.Td>
                        <strong>{r.title || r.id}</strong>
                        <small className="block muted">{r.id}</small>
                      </Table.Td>
                      <Table.Td>
                        <SourceTag source={r.source} />
                      </Table.Td>
                      <Table.Td>
                        {r.source?.retrievedAt
                          ? new Date(r.source.retrievedAt).toLocaleString(
                              "ko-KR",
                            )
                          : "기록 없음"}
                      </Table.Td>
                      <Table.Td>
                        <div className="button-row compact">
                          <Button
                            size="sm"
                            variant="light"
                            onClick={() => setRecord(r)}
                          >
                            원천 보기
                          </Button>
                          <Button
                            size="sm"
                            variant="subtle"
                            color="red"
                            onClick={() => setRemove(r)}
                          >
                            삭제
                          </Button>
                        </div>
                      </Table.Td>
                    </Table.Tr>
                  ))}
                </Table.Tbody>
              </Table>
            </div>
            <div className="data-pagination">
              <span>
                전체 {number(rows.data.total)}개 중 {offset + 1}–
                {Math.min(offset + rows.data.records.length, rows.data.total)}
              </span>
              <Button
                variant="default"
                disabled={offset === 0}
                onClick={() => setOffset(Math.max(0, offset - 25))}
              >
                이전
              </Button>
              <Button
                variant="default"
                disabled={offset + 25 >= rows.data.total}
                onClick={() => setOffset(offset + 25)}
              >
                다음
              </Button>
            </div>
          </>
        ) : (
          !rows.loading && (
            <Empty
              title="아직 보관된 원천이 없습니다"
              description="연동을 테스트하고 동기화하면 선택한 공식 항목과 출처가 표시됩니다."
            />
          )
        )}
      </Panel>
      <Modal
        opened={!!record}
        onClose={() => setRecord(null)}
        title="원천 기록과 출처"
        size="xl"
        centered
      >
        {record && (
          <>
            <h3>{record.title || record.id}</h3>
            <SourceTag source={record.source} />
            <dl className="source-detail">
              <dt>내부 원천 ID</dt>
              <dd>{record.id}</dd>
              <dt>제공기관</dt>
              <dd>{record.source?.provider || "미기록"}</dd>
              <dt>원문 식별번호</dt>
              <dd>{record.source?.recordId || "미기록"}</dd>
              <dt>수집 시각</dt>
              <dd>{record.source?.retrievedAt || "미기록"}</dd>
              <dt>저장한 공식 항목</dt>
              <dd>{record.source?.fields?.join(", ") || "기록 없음"}</dd>
            </dl>
            <h3>선택하여 저장한 원문</h3>
            <p className="muted">
              이 데이터는 역량 수준이나 점수로 변환되기 전의 출처 자료입니다.
            </p>
            <pre className="code-block">
              {JSON.stringify(record.raw || {}, null, 2)}
            </pre>
            <details>
              <summary>출처·정규화 기록 전체 보기</summary>
              <pre className="code-block">
                {JSON.stringify(record, null, 2)}
              </pre>
            </details>
          </>
        )}
      </Modal>
      <Modal
        opened={!!remove}
        onClose={() => setRemove(null)}
        title="원천 데이터 삭제"
        centered
      >
        <p>
          {remove?.title || remove?.id} 기록을 삭제하시겠어요? 해당 원천을
          참조하는 매핑과 이력을 먼저 확인해 주세요.
        </p>
        <Button color="red" loading={busy} onClick={del}>
          원천 데이터 삭제
        </Button>
      </Modal>
    </>
  );
}
function SourceOptions({
  dataset,
  selected,
  onChange,
  label,
  multiple = false,
  required = false,
  onRecord,
}: {
  dataset: string;
  selected: string | string[];
  onChange: (v: any) => void;
  label: string;
  multiple?: boolean;
  required?: boolean;
  onRecord?: (r: SourceRecord) => void;
}) {
  const [search, setSearch] = useState(""),
    [debounced] = useDebouncedValue(search, 250),
    records = useResource<Records>(
      `/admin/source-records?dataset=${dataset}&q=${encodeURIComponent(debounced)}&offset=0&limit=100`,
    );
  const selectedIds = Array.isArray(selected)
    ? selected
    : selected
      ? [selected]
      : [];
  const options = new Map(
    selectedIds.map((id) => [id, { value: id, label: id }]),
  );
  for (const record of records.data?.records || [])
    options.set(record.id, {
      value: record.id,
      label: `${record.title || record.id} · ${record.id}`,
    });
  const shared = {
    label,
    required,
    searchable: true,
    searchValue: search,
    onSearchChange: setSearch,
    data: [...options.values()],
    nothingFoundMessage: records.loading
      ? "불러오는 중..."
      : "일치하는 원천이 없습니다. 먼저 동기화하세요.",
    description: "명칭이나 원천 식별번호로 검색합니다.",
    error: records.error || undefined,
  };
  return multiple ? (
    <MultiSelect {...shared} value={selectedIds} onChange={onChange} />
  ) : (
    <Select
      {...shared}
      value={typeof selected === "string" ? selected : null}
      onChange={(v) => {
        onChange(v || "");
        const record = records.data?.records.find((r) => r.id === v);
        if (record) onRecord?.(record);
      }}
    />
  );
}
export function MappingsPage() {
  const resource = useResource<Mapping[]>("/admin/mappings"),
    { notify } = useApp(),
    [editing, setEditing] = useState<Mapping | null>(null),
    [history, setHistory] = useState<Mapping | null>(null),
    [publish, setPublish] = useState<Mapping | null>(null),
    [remove, setRemove] = useState<Mapping | null>(null),
    [reviewed, setReviewed] = useState(false),
    [busy, setBusy] = useState("");
  const historyRows = useResource<any[]>(
    history?.id ? `/admin/mappings/${history.id}/history` : null,
  );
  function set<K extends keyof Mapping>(key: K, value: Mapping[K]) {
    setEditing((m) => (m ? { ...m, [key]: value } : m));
  }
  function updateSkill(i: number, key: keyof MappingSkill, value: any) {
    if (editing)
      set(
        "skills",
        editing.skills.map((s, j) => (j === i ? { ...s, [key]: value } : s)),
      );
  }
  async function save(e: React.FormEvent) {
    e.preventDefault();
    if (!editing) return;
    if (!editing.skills.length)
      return notify("분석할 내부 역량을 한 개 이상 추가해 주세요.", "error");
    if (editing.skills.some((s) => !s.sourceRecordIds.length))
      return notify("각 역량의 근거 원천을 한 개 이상 선택해 주세요.", "error");
    setBusy("save");
    try {
      await api(
        editing.id ? `/admin/mappings/${editing.id}` : "/admin/mappings",
        editing.id ? "PUT" : "POST",
        editing,
      );
      setEditing(null);
      notify("초안을 저장했습니다. 검토·게시 후 계산에 반영됩니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function activate() {
    if (!publish || !reviewed) return;
    setBusy("publish");
    try {
      await api(`/admin/mappings/${publish.id}/publish`, "POST", {
        version: publish.version,
      });
      setPublish(null);
      setReviewed(false);
      notify("검토한 매핑 버전을 게시했습니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  async function del() {
    if (!remove) return;
    setBusy("delete");
    try {
      await api(`/admin/mappings/${remove.id}`, "DELETE");
      setRemove(null);
      notify("매핑을 삭제했습니다. 변경 이력은 보존됩니다.");
      void resource.reload();
    } catch (e) {
      notify((e as Error).message, "error");
    } finally {
      setBusy("");
    }
  }
  return (
    <>
      <PageTitle
        eyebrow="REVIEWED SKILL MAPPING"
        title="근거를 검토하고, 내부 기준으로 연결"
        description="공공 원천을 그대로 보존하면서 역량 별칭·요구 수준·가중치와 매핑 근거를 별도로 관리하세요."
        action={
          <Button
            leftSection={<IconPlus size={18} />}
            onClick={() => setEditing(newMapping())}
          >
            매핑 초안 만들기
          </Button>
        }
      />
      <Alert
        color="violet"
        title="공식 NCS 수준이 아닌, 관리자 설정 분석 기준입니다"
      >
        요구 수준 0~5와 가중치는 NextRole의 내부 시뮬레이션 설정입니다. 관리자가
        원천 자료와 근거를 검토하여 게시한 버전만 분석에 사용합니다. 수정하면 새
        초안이 되어 다시 검토해야 합니다.
      </Alert>
      <Loading loading={resource.loading} error={resource.error} />
      {resource.data?.length === 0 && (
        <Empty
          title="아직 검토한 역량 매핑이 없습니다"
          description="직업정보와 NCS 원천을 수집한 뒤, 근거가 있는 내부 역량 사전을 작성하세요."
          action={
            <Button component={Link} to="/admin/data" variant="light">
              원천 데이터 확인
            </Button>
          }
        />
      )}
      <div className="mapping-list">
        {resource.data?.map((m) => (
          <Panel key={m.id}>
            <div className="panel-title">
              <h2>{m.title}</h2>
              <div className="tag-row">
                <Badge color={m.status === "published" ? "teal" : "gray"}>
                  {m.status === "published" ? "검토·게시됨" : "검토 전 초안"}
                </Badge>
                <Badge variant="light" color="violet">
                  버전 {m.version}
                </Badge>
              </div>
            </div>
            <SourceTag
              source={{
                kind: "derived",
                name: "NextRole 관리자 검토 매핑",
                synthetic: false,
              }}
            />
            <p>{m.basis}</p>
            {m.validationWarning && (
              <Alert color="orange" mb="md" title="현재 계산에서 제외됨">
                {m.validationWarning} 원천을 확인하고 초안을 수정한 뒤 다시
                검토·게시해 주세요.
              </Alert>
            )}
            <div className="tag-row">
              {m.skills.map((s) => (
                <span className="skill-tag" key={s.internalSkillId}>
                  {s.name} · 내부 수준 {s.level} · 가중치 {s.weight}
                </span>
              ))}
            </div>
            <p className="helper">
              원천 직무 코드 {m.occupationCode || "미기록"} · 역량{" "}
              {m.skills.length}개
              {m.reviewedAt
                ? ` · 검토 ${new Date(m.reviewedAt).toLocaleString("ko-KR")}`
                : ""}
            </p>
            <div className="button-row">
              <Button
                variant="light"
                onClick={() =>
                  setEditing({
                    ...m,
                    ncsRecordIds: m.ncsRecordIds || [],
                    skills: m.skills.map((s) => ({
                      ...s,
                      aliases: s.aliases || [],
                      sourceRecordIds: s.sourceRecordIds || [],
                    })),
                  })
                }
              >
                초안 수정
              </Button>
              <Button
                variant="default"
                leftSection={<IconHistory size={17} />}
                onClick={() => setHistory(m)}
              >
                버전 이력
              </Button>
              {m.status !== "published" && (
                <Button
                  leftSection={<IconCheck size={17} />}
                  onClick={() => {
                    setPublish(m);
                    setReviewed(false);
                  }}
                >
                  검토 후 게시
                </Button>
              )}
              <Button variant="subtle" color="red" onClick={() => setRemove(m)}>
                삭제
              </Button>
            </div>
          </Panel>
        ))}
      </div>
      <Modal
        opened={!!editing}
        onClose={() => setEditing(null)}
        title={editing?.id ? "역량 매핑 초안 수정" : "역량 매핑 초안 만들기"}
        size="xl"
        centered
      >
        {editing && (
          <form className="form-stack" onSubmit={save}>
            <TextInput
              required
              label="내부 직무 모델 이름"
              placeholder="예: AI 플랫폼 엔지니어"
              value={editing.title}
              onChange={(e) => set("title", e.target.value)}
            />
            <SourceOptions
              required
              dataset="occupation_details"
              label="기준 직업정보 원천"
              selected={editing.occupationRecordId}
              onChange={(v) => set("occupationRecordId", v)}
              onRecord={(r) =>
                setEditing((m) =>
                  m
                    ? {
                        ...m,
                        occupationRecordId: r.id,
                        occupationCode:
                          r.occupationCode || r.code || r.raw?.jobCd || "",
                      }
                    : m,
                )
              }
            />
            <TextInput
              label="원천 직업 코드"
              value={editing.occupationCode}
              onChange={(e) => set("occupationCode", e.target.value)}
              description="선택한 원천의 공식 직업 코드를 확인하세요."
            />
            <TagsInput
              label="연결할 채용 직종코드 (선택)"
              description="원천 직업코드와 채용 직종코드는 다를 수 있습니다. 확인한 공고의 occupationCode만 입력하세요."
              value={editing.jobCodes || []}
              onChange={(v) => set("jobCodes", v)}
            />
            <SourceOptions
              dataset="ncs_units"
              label="연결할 NCS 능력단위 원천 (선택)"
              multiple
              selected={editing.ncsRecordIds}
              onChange={(v) => set("ncsRecordIds", v)}
            />
            <Textarea
              required
              label="직무·역량 매핑 전체 근거"
              description="왜 이 원천과 내부 역량을 연결했는지 설명하세요."
              minRows={3}
              value={editing.basis}
              onChange={(e) => set("basis", e.target.value)}
            />
            <div className="panel-title">
              <h2>내부 역량 기준</h2>
              <Button
                variant="light"
                leftSection={<IconPlus size={16} />}
                onClick={() =>
                  set("skills", [
                    ...editing.skills,
                    {
                      internalSkillId: "",
                      name: "",
                      aliases: [],
                      level: 1,
                      weight: 1,
                      sourceRecordIds: [],
                      basis: "",
                    },
                  ])
                }
              >
                역량 추가
              </Button>
            </div>
            <Alert color="gray">
              아래 0~5는 관리자 설정 수준이며 공식 NCS 수준과 다릅니다. 별칭과
              가중치를 포함한 추론·가공값은 원천과 구분되어 저장됩니다.
            </Alert>
            {editing.skills.map((skill, i) => (
              <section className="mapping-skill-editor" key={i}>
                <div className="panel-title">
                  <h3>역량 {i + 1}</h3>
                  <Button
                    variant="subtle"
                    color="red"
                    size="sm"
                    leftSection={<IconTrash size={16} />}
                    onClick={() =>
                      set(
                        "skills",
                        editing.skills.filter((_, j) => j !== i),
                      )
                    }
                  >
                    역량 삭제
                  </Button>
                </div>
                <div className="form-grid">
                  <TextInput
                    required
                    label="내부 역량 식별번호"
                    placeholder="예: python"
                    value={skill.internalSkillId}
                    onChange={(e) =>
                      updateSkill(i, "internalSkillId", e.target.value)
                    }
                  />
                  <TextInput
                    required
                    label="역량 이름"
                    value={skill.name}
                    onChange={(e) => updateSkill(i, "name", e.target.value)}
                  />
                  <NumberInput
                    label="내부 요구 수준 (0 초과~5)"
                    min={0.1}
                    step={0.1}
                    decimalScale={2}
                    max={5}
                    value={skill.level}
                    onChange={(v) => updateSkill(i, "level", Number(v))}
                  />
                  <NumberInput
                    label="가중치"
                    min={0.1}
                    max={100}
                    step={0.1}
                    decimalScale={2}
                    value={skill.weight}
                    onChange={(v) => updateSkill(i, "weight", Number(v))}
                  />
                </div>
                <TagsInput
                  label="동일 역량으로 처리할 별칭"
                  value={skill.aliases}
                  onChange={(v) => updateSkill(i, "aliases", v)}
                  mt="md"
                />
                <MultiSelect
                  required
                  label="수준·역량 판단의 근거 원천"
                  description="위에서 선택한 직업정보와 NCS 원천 중 해당 근거를 선택하세요."
                  mt="md"
                  data={[
                    ...new Set(
                      [
                        editing.occupationRecordId,
                        ...editing.ncsRecordIds,
                        ...skill.sourceRecordIds,
                      ].filter(Boolean),
                    ),
                  ].map((id) => ({ value: id, label: id }))}
                  value={skill.sourceRecordIds}
                  onChange={(v) => updateSkill(i, "sourceRecordIds", v)}
                />
                <Textarea
                  required
                  label="이 역량의 수준·가중치 근거"
                  minRows={3}
                  mt="md"
                  value={skill.basis}
                  onChange={(e) => updateSkill(i, "basis", e.target.value)}
                />
              </section>
            ))}
            <Button type="submit" loading={busy === "save"}>
              검토 전 초안 저장
            </Button>
          </form>
        )}
      </Modal>
      <Modal
        opened={!!publish}
        onClose={() => setPublish(null)}
        title="검토한 매핑 버전 게시"
        size="lg"
        centered
      >
        {publish && (
          <>
            <h3>
              {publish.title} · 버전 {publish.version}
            </h3>
            <p>{publish.basis}</p>
            <div className="mapping-review-list">
              {publish.skills.map((s) => (
                <div key={s.internalSkillId}>
                  <strong>
                    {s.name} · 내부 수준 {s.level} · 가중치 {s.weight}
                  </strong>
                  <p>{s.basis}</p>
                  <small>근거 원천: {s.sourceRecordIds.join(", ")}</small>
                </div>
              ))}
            </div>
            <Checkbox
              label="원천 자료, 매핑 근거, 내부 요구 수준과 가중치를 검토했으며 이 버전을 계산에 사용하도록 게시합니다."
              checked={reviewed}
              onChange={(e) => setReviewed(e.target.checked)}
              mt="lg"
            />
            <Button
              mt="lg"
              disabled={!reviewed}
              loading={busy === "publish"}
              onClick={activate}
            >
              검토 완료 및 버전 게시
            </Button>
          </>
        )}
      </Modal>
      <Modal
        opened={!!history}
        onClose={() => setHistory(null)}
        title="매핑 버전 이력"
        size="xl"
        centered
      >
        <Loading loading={historyRows.loading} error={historyRows.error} />
        {historyRows.data?.map((entry, i) => {
          const m = entry.mapping || entry;
          return (
            <details className="mapping-history" key={`${m.version}-${i}`}>
              <summary>
                버전 {m.version || "—"} ·{" "}
                {m.status === "published" ? "게시" : "초안"} ·{" "}
                {m.updatedAt
                  ? new Date(m.updatedAt).toLocaleString("ko-KR")
                  : "시각 미기록"}
              </summary>
              <pre className="code-block">{JSON.stringify(entry, null, 2)}</pre>
            </details>
          );
        })}
        {historyRows.data?.length === 0 && <p>저장된 이전 버전이 없습니다.</p>}
      </Modal>
      <Modal
        opened={!!remove}
        onClose={() => setRemove(null)}
        title="역량 매핑 삭제"
        centered
      >
        <p>
          {remove?.title} 매핑을 삭제하시겠어요? 게시된 직무 모델은 이후
          계산에서 제외되며 변경 이력은 보존됩니다.
        </p>
        <Button color="red" loading={busy === "delete"} onClick={del}>
          매핑 삭제
        </Button>
      </Modal>
    </>
  );
}
