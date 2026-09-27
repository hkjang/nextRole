import {
  Alert,
  Badge,
  Button,
  Checkbox,
  NumberInput,
  Textarea,
  TextInput,
} from "@mantine/core";
import { IconArrowUpRight, IconDatabase } from "@tabler/icons-react";
import { Panel, SourceTag } from "./App";

export type PresetMetadata = {
  id: string;
  name: string;
  dataset: string;
  description: string;
  specUrl: string;
  endpoint: string;
  rootPath: string;
  format: string;
  fields: {
    name: string;
    label: string;
    description: string;
    required: boolean;
    defaultSelected: boolean;
  }[];
  defaultSelectedFields: string[];
  requiredParams: string[];
  defaultParams: Record<string, string>;
};
export type PresetResponse = { presets: any[]; metadata: PresetMetadata[] };
type Param = {
  name: string;
  label: string;
  description?: string;
  max?: number;
  multiline?: boolean;
};
const paramsByPreset: Record<string, Param[]> = {
  "work24-occupation-summary": [
    {
      name: "jobCd",
      label: "직업 코드",
      description: "공식 직업정보에서 확인한 jobCd를 입력하세요.",
    },
  ],
  "work24-ncs": [
    {
      name: "jobCont",
      label: "수행직무내용",
      multiline: true,
      description:
        "관리자가 작성한 일반 직무 설명만 입력하세요. 개인 경력·이력서 내용은 자동 전송하지 않습니다.",
    },
    {
      name: "limit",
      label: "조회 능력단위 수",
      description: "기본 5개 · 최대 5,000개는 NextRole의 처리 상한입니다.",
      max: 5000,
    },
  ],
  "work24-jobs": [
    { name: "occupation", label: "직종 코드 (선택)" },
    { name: "region", label: "지역 코드 (선택)" },
    { name: "keyword", label: "검색어 (선택)" },
    { name: "startPage", label: "조회 페이지", max: 1000 },
    { name: "display", label: "페이지당 공고 수", max: 100 },
  ],
  "work24-training": [
    {
      name: "srchTraStDt",
      label: "훈련 시작일 검색 범위 · 시작",
      description: "YYYYMMDD 형식으로 입력하세요.",
    },
    {
      name: "srchTraEndDt",
      label: "훈련 시작일 검색 범위 · 끝",
      description: "과정 종료일이 아닌 시작일의 검색 범위입니다.",
    },
    { name: "pageNum", label: "조회 페이지", max: 1000 },
    { name: "pageSize", label: "페이지당 과정 수", max: 100 },
    { name: "srchTraArea1", label: "시·도 코드 (선택)" },
    {
      name: "srchTraArea2",
      label: "시·군·구 코드 (선택)",
      description: "시·도 코드와 함께 입력하세요.",
    },
    { name: "srchNcs1", label: "NCS 대분류 코드 (선택)" },
    { name: "srchNcs2", label: "NCS 중분류 코드 (선택)" },
    { name: "srchNcs3", label: "NCS 소분류 코드 (선택)" },
    {
      name: "srchNcs4",
      label: "NCS 세분류 코드 (선택)",
      description: "조회 범위에 맞는 NCS 상위 분류 코드도 입력하세요.",
    },
  ],
};
export function PresetCards({
  metadata,
  onSelect,
}: {
  metadata: PresetMetadata[];
  onSelect: (id: string) => void;
}) {
  return (
    <div className="work24-presets">
      {metadata.map((m) => (
        <Panel key={m.id}>
          <div className="tag-row">
            <Badge color="blue">공공 API 원천</Badge>
            <Badge color="gray" variant="light">
              {m.format.toUpperCase()}
            </Badge>
          </div>
          <h2>{m.name}</h2>
          <p>{m.description}</p>
          <div className="preset-card-actions">
            <Button
              variant="light"
              leftSection={<IconDatabase size={17} />}
              onClick={() => onSelect(m.id)}
            >
              이 프리셋으로 연결
            </Button>
            <a href={m.specUrl} target="_blank" rel="noreferrer">
              공식 API 명세 <IconArrowUpRight size={16} />
            </a>
          </div>
        </Panel>
      ))}
    </div>
  );
}
export function PresetConfig({
  metadata,
  params,
  selectedFields,
  onParams,
  onFields,
}: {
  metadata: PresetMetadata;
  params: string;
  selectedFields?: string[];
  onParams: (v: string) => void;
  onFields: (v: string[]) => void;
}) {
  let values: Record<string, string> = {};
  let invalid = false;
  try {
    values = JSON.parse(params || "{}");
  } catch {
    invalid = true;
  }
  const chosen = selectedFields ?? metadata.defaultSelectedFields;
  const fields =
    paramsByPreset[metadata.id] ||
    metadata.requiredParams.map((name) => ({ name, label: name }));
  const change = (name: string, value: string) =>
    onParams(JSON.stringify({ ...values, [name]: value }, null, 2));
  return (
    <section className="preset-config">
      <Alert color="blue" title={metadata.name}>
        {metadata.description}{" "}
        <a href={metadata.specUrl} target="_blank" rel="noreferrer">
          공식 명세 확인
        </a>
      </Alert>
      <h3>조회 조건</h3>
      {invalid && (
        <Alert color="red">
          요청 매개변수 JSON을 올바르게 수정한 뒤 간편 입력을 사용하세요.
        </Alert>
      )}
      <div className="form-grid">
        {fields.map((f) =>
          f.multiline ? (
            <Textarea
              key={f.name}
              className="form-full"
              required={metadata.requiredParams.includes(f.name)}
              label={f.label}
              description={f.description}
              minRows={4}
              value={values[f.name] || ""}
              disabled={invalid}
              onChange={(e) => change(f.name, e.target.value)}
            />
          ) : f.max ? (
            <NumberInput
              key={f.name}
              required={metadata.requiredParams.includes(f.name)}
              label={f.label}
              description={f.description}
              min={1}
              max={f.max}
              allowDecimal={false}
              value={values[f.name] ? Number(values[f.name]) : ""}
              disabled={invalid}
              onChange={(v) => change(f.name, String(v))}
            />
          ) : (
            <TextInput
              key={f.name}
              required={metadata.requiredParams.includes(f.name)}
              label={f.label}
              description={f.description}
              value={values[f.name] || ""}
              disabled={invalid}
              onChange={(e) => change(f.name, e.target.value)}
            />
          ),
        )}
      </div>
      <div className="panel-title">
        <h3>선택하여 저장할 공식 항목</h3>
        <Badge variant="light">
          {chosen.length} / {metadata.fields.length}개 선택
        </Badge>
      </div>
      <p className="helper">
        필수 식별 항목은 유지됩니다. 선택하지 않은 응답 항목은 원천 저장에서
        제외하며, 공식 응답에 없는 역량 수준·가중치는 생성하지 않습니다.
      </p>
      <div className="button-row compact">
        <Button
          size="sm"
          variant="default"
          onClick={() => onFields(metadata.defaultSelectedFields)}
        >
          기본 항목 선택
        </Button>
        <Button
          size="sm"
          variant="subtle"
          onClick={() => onFields(metadata.fields.map((f) => f.name))}
        >
          모든 항목 선택
        </Button>
        <Button
          size="sm"
          variant="subtle"
          onClick={() =>
            onFields(
              metadata.fields.filter((f) => f.required).map((f) => f.name),
            )
          }
        >
          필수 항목만 선택
        </Button>
      </div>
      <div className="official-field-grid">
        {metadata.fields.map((f) => (
          <Checkbox
            key={f.name}
            label={`${f.label}${f.required ? " · 필수" : ""}`}
            description={f.name}
            checked={chosen.includes(f.name) || f.required}
            disabled={f.required}
            onChange={(e) =>
              onFields(
                e.target.checked
                  ? [...chosen, f.name]
                  : chosen.filter((n) => n !== f.name),
              )
            }
          />
        ))}
      </div>
      {metadata.id === "work24-training" && (
        <Alert color="orange">
          훈련비 원문은 개인별 확정 본인부담금이 아닙니다. 폐기된 eiEmplCnt3Gt10
          항목은 수집하지 않습니다. 과정 ID와 회차를 함께 보관합니다.
        </Alert>
      )}
    </section>
  );
}
export function ConnectorPreview({ preview }: { preview: any }) {
  const rows = Array.isArray(preview) ? preview : preview ? [preview] : [];
  return (
    <div className="connector-preview-list">
      {rows.map((r: any, i: number) => (
        <section key={r.id || i}>
          <h3>{r.title || r.id || `항목 ${i + 1}`}</h3>
          <SourceTag source={r.source} />
          {r.raw && (
            <>
              <h4>선택하여 보관할 원천 응답</h4>
              <pre className="code-block">{JSON.stringify(r.raw, null, 2)}</pre>
            </>
          )}
          <details>
            <summary>출처와 정규화 결과 확인</summary>
            <pre className="code-block">{JSON.stringify(r, null, 2)}</pre>
          </details>
        </section>
      ))}
    </div>
  );
}
