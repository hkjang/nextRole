export class APIError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
  }
}
export async function api<T = any>(
  path: string,
  method = "GET",
  body?: unknown,
): Promise<T> {
  const response = await fetch(`/api/v1${path}`, {
    method,
    credentials: "same-origin",
    headers:
      body instanceof FormData ? {} : { "Content-Type": "application/json" },
    body:
      body === undefined
        ? undefined
        : body instanceof FormData
          ? body
          : JSON.stringify(body),
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok)
    throw new APIError(
      response.status,
      data.error || `요청을 처리하지 못했습니다 (${response.status})`,
    );
  return data;
}
export async function streamAI(
  body: unknown,
  onDelta: (text: string) => void,
  signal?: AbortSignal,
  onProfile?: (profile: Profile) => void,
) {
  const response = await fetch("/api/v1/ai/stream", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    credentials: "same-origin",
    body: JSON.stringify(body),
    signal,
  });
  if (!response.ok) {
    const data = await response.json().catch(() => ({}));
    throw new Error(data.error || "AI 요청을 처리하지 못했습니다.");
  }
  if (!response.body) throw new Error("스트리밍 응답이 없습니다.");
  const reader = response.body.getReader(),
    decoder = new TextDecoder();
  let buffer = "",
    mode = "";
  for (;;) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value, { stream: !done }).replace(/\r\n/g, "\n");
    let boundary;
    while ((boundary = buffer.indexOf("\n\n")) >= 0) {
      const block = buffer.slice(0, boundary);
      buffer = buffer.slice(boundary + 2);
      let event = "message";
      const data: string[] = [];
      for (const line of block.split("\n")) {
        if (line.startsWith("event:")) event = line.slice(6).trim();
        if (line.startsWith("data:")) data.push(line.slice(5).trimStart());
      }
      if (data.length) {
        const raw = data.join("\n");
        if (raw === "[DONE]") continue;
        const parsed = JSON.parse(raw);
        if (event === "error") throw new Error(parsed.error);
        if (event === "delta") onDelta(parsed.text || "");
        if (event === "done") mode = parsed.mode;
        if (event === "profile") onProfile?.(parsed);
      }
    }
    if (done) break;
  }
  return mode;
}
export type Source = { name: string; url: string; synthetic: boolean };
export type Skill = {
  name: string;
  level: number;
  years?: number;
  confidence?: string;
};
export type Profile = {
  name: string;
  currentRole: string;
  yearsExperience: number;
  careerBreakMonths?: number;
  education: string;
  region: string;
  domain: string;
  narrative: string;
  skills: Skill[];
  certifications: string[];
  preferences: string[];
  weeklyHours: number;
};
export type Job = {
  id: string;
  title: string;
  category: string;
  description: string;
  skills: (Skill & { weight: number })[];
  minExperience: number;
  domain: string;
  education: string;
  regions: string[];
  source: Source;
  salaryRange: string;
  bridgeIds: string[];
};
export type Simulation = {
  marketDemand?: number;
  rankingReason?: string;
  id?: string;
  job: Job;
  score: number;
  baselineScore: number;
  difficulty: number;
  estimatedMonths: number;
  gaps: {
    name: string;
    required: number;
    current: number;
    gap: number;
    weight: number;
  }[];
  factors: { name: string; score: number; max: number; reason: string }[];
  roi: { name: string; before: number; after: number; gain: number }[];
  paths: {
    id: string;
    title: string;
    roles: string[];
    months: number;
    skills: string[];
    reuse: number;
    cost: number;
    reason: string;
  }[];
  plan: { month: number; title: string; tasks: string[]; skills: string[] }[];
  strengths: string[];
  warnings: string[];
  explanation: string;
  source: Source;
  createdAt?: string;
  scenario?: {
    jobId: string;
    months: number;
    addedSkills: Skill[];
    certifications: string[];
    project: string;
  };
};
export type User = {
  id: string;
  email: string;
  name: string;
  role: string;
  preferences: Record<string, any>;
  version: string;
};
export const emptyProfile: Profile = {
  name: "",
  currentRole: "",
  yearsExperience: 0,
  education: "",
  region: "",
  domain: "",
  narrative: "",
  skills: [],
  certifications: [],
  preferences: [],
  weeklyHours: 10,
};
