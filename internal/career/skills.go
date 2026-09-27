package career

import (
	"math"
	"sort"
	"strings"
	"unicode"
)

type skillDefinition struct {
	name              string
	aliases, features []string
	hours             float64
}

// Features form a local, versioned semantic vector space. Cosine similarity
// between these explicit ontology features contributes ONLY to transferability;
// it never changes the exact skill gap. No external embedding service is needed.
var skillDefinitions = []skillDefinition{
	{"Java", []string{"java", "자바"}, []string{"software", "backend", "programming", "jvm"}, 40},
	{"Spring", []string{"spring", "spring boot", "스프링", "스프링부트"}, []string{"software", "backend", "framework", "jvm"}, 35},
	{"Python", []string{"python", "파이썬"}, []string{"software", "backend", "programming", "data"}, 35},
	{"Go", []string{"golang", "go 언어", "고랭", "go"}, []string{"software", "backend", "programming", "infrastructure"}, 40},
	{"JavaScript", []string{"javascript", "자바스크립트", "js"}, []string{"software", "frontend", "programming", "web"}, 35},
	{"TypeScript", []string{"typescript", "타입스크립트", "ts"}, []string{"software", "frontend", "programming", "web"}, 30},
	{"React", []string{"react", "reactjs", "react.js", "리액트"}, []string{"software", "frontend", "framework", "web"}, 35},
	{"HTML/CSS", []string{"html/css", "html", "css"}, []string{"software", "frontend", "design", "web"}, 25},
	{"SQL", []string{"sql", "postgresql", "postgres", "mysql", "mariadb", "oracle database", "데이터베이스 쿼리"}, []string{"software", "data", "query", "database"}, 35},
	{"Linux", []string{"linux", "리눅스", "unix"}, []string{"infrastructure", "systems", "operations", "os"}, 35},
	{"Docker", []string{"docker", "도커", "컨테이너"}, []string{"infrastructure", "systems", "containers", "deployment"}, 25},
	{"Kubernetes", []string{"kubernetes", "k8s", "쿠버네티스"}, []string{"infrastructure", "systems", "containers", "orchestration"}, 50},
	{"CI/CD", []string{"ci/cd", "cicd", "jenkins", "github actions", "gitlab ci"}, []string{"software", "infrastructure", "automation", "deployment"}, 30},
	{"Git", []string{"git", "깃", "형상관리", "버전 관리"}, []string{"software", "collaboration", "versioning"}, 15},
	{"Backend API", []string{"backend api", "rest api", "restful api", "백엔드 api", "api 개발"}, []string{"software", "backend", "web", "integration"}, 40},
	{"FastAPI", []string{"fastapi", "fast api"}, []string{"software", "backend", "framework", "python"}, 25},
	{"GPU 운영", []string{"gpu 운영", "gpu infrastructure", "gpu 인프라", "cuda"}, []string{"infrastructure", "systems", "ai", "accelerator"}, 50},
	{"LLM Serving", []string{"llm serving", "llm 서빙", "모델 서빙", "llmserving"}, []string{"infrastructure", "ai", "deployment", "inference"}, 45},
	{"vLLM", []string{"vllm"}, []string{"ai", "inference", "framework", "accelerator"}, 30},
	{"Observability", []string{"observability", "옵저버빌리티", "관측성", "prometheus", "grafana", "모니터링"}, []string{"infrastructure", "operations", "monitoring", "reliability"}, 30},
	{"Terraform", []string{"terraform", "테라폼", "infrastructure as code", "iac"}, []string{"infrastructure", "automation", "provisioning", "cloud"}, 35},
	{"AWS", []string{"aws", "아마존 웹 서비스", "amazon web services"}, []string{"infrastructure", "cloud", "systems", "provisioning"}, 45},
	{"네트워크", []string{"네트워크", "networking", "network", "tcp/ip"}, []string{"infrastructure", "systems", "connectivity", "security"}, 40},
	{"보안", []string{"보안", "security", "정보보호"}, []string{"security", "systems", "risk", "compliance"}, 45},
	{"Data Pipeline", []string{"data pipeline", "데이터 파이프라인", "etl", "elt"}, []string{"data", "automation", "integration", "pipeline"}, 40},
	{"Spark", []string{"spark", "스파크", "pyspark"}, []string{"data", "distributed", "framework", "pipeline"}, 45},
	{"Airflow", []string{"airflow", "에어플로우"}, []string{"data", "automation", "orchestration", "pipeline"}, 30},
	{"데이터 모델링", []string{"데이터 모델링", "data modeling", "data modelling", "스키마 설계"}, []string{"data", "database", "design", "analysis"}, 40},
	{"머신러닝", []string{"머신러닝", "machine learning", "기계학습"}, []string{"ai", "data", "statistics", "modeling"}, 60},
	{"통계", []string{"통계", "statistics", "statistical"}, []string{"data", "statistics", "analysis", "research"}, 45},
	{"PyTorch", []string{"pytorch", "파이토치"}, []string{"ai", "data", "framework", "modeling"}, 45},
	{"MLOps", []string{"mlops", "ml ops"}, []string{"ai", "deployment", "automation", "monitoring"}, 45},
	{"LLM API", []string{"llm api", "llm integration", "생성형 ai api"}, []string{"ai", "backend", "integration", "inference"}, 30},
	{"RAG", []string{"rag", "검색 증강 생성", "retrieval augmented generation"}, []string{"ai", "data", "retrieval", "inference"}, 40},
	{"데이터 분석", []string{"데이터 분석", "data analysis", "data analytics", "데이터분석"}, []string{"data", "analysis", "business", "research"}, 35},
	{"데이터 시각화", []string{"데이터 시각화", "data visualization", "tableau", "power bi", "powerbi"}, []string{"data", "analysis", "presentation", "design"}, 30},
	{"Excel", []string{"excel", "엑셀", "스프레드시트"}, []string{"data", "business", "spreadsheet", "analysis"}, 20},
	{"테스트 자동화", []string{"테스트 자동화", "test automation", "selenium", "playwright", "cypress", "자동화 테스트"}, []string{"software", "automation", "testing", "quality"}, 35},
	{"품질 관리", []string{"품질 관리", "품질관리", "quality assurance", "quality control", "qa"}, []string{"quality", "operations", "testing", "process"}, 35},
	{"문제 해결", []string{"문제 해결", "문제해결", "problem solving", "트러블슈팅", "troubleshooting"}, []string{"operations", "analysis", "reasoning", "support"}, 25},
	{"문서 작성", []string{"문서 작성", "문서작성", "documentation", "보고서 작성", "기술 문서"}, []string{"business", "communication", "writing", "process"}, 20},
	{"커뮤니케이션", []string{"커뮤니케이션", "communication", "의사소통"}, []string{"business", "communication", "collaboration", "people"}, 25},
	{"프로젝트 관리", []string{"프로젝트 관리", "project management", "프로젝트관리", "pmp"}, []string{"business", "operations", "planning", "collaboration"}, 35},
	{"요구사항 분석", []string{"요구사항 분석", "요구사항분석", "requirements analysis", "요구 분석"}, []string{"business", "analysis", "research", "planning"}, 30},
	{"제품 기획", []string{"제품 기획", "제품기획", "product management", "서비스 기획", "프로덕트 기획"}, []string{"business", "product", "planning", "research"}, 40},
	{"고객 조사", []string{"고객 조사", "고객조사", "user research", "사용자 조사", "고객 인터뷰", "시장 조사"}, []string{"business", "research", "customer", "analysis"}, 30},
	{"Figma", []string{"figma", "피그마"}, []string{"design", "frontend", "prototype", "tool"}, 25},
	{"UX 디자인", []string{"ux 디자인", "ux design", "ui/ux", "ux/ui", "사용자 경험 설계", "ui 디자인"}, []string{"design", "research", "customer", "frontend"}, 40},
	{"프로토타이핑", []string{"프로토타이핑", "prototyping", "프로토타입"}, []string{"design", "prototype", "product", "testing"}, 25},
	{"웹 접근성", []string{"웹 접근성", "accessibility", "wcag"}, []string{"design", "frontend", "compliance", "web"}, 30},
	{"디지털 마케팅", []string{"디지털 마케팅", "digital marketing", "퍼포먼스 마케팅"}, []string{"marketing", "business", "data", "campaign"}, 35},
	{"광고 운영", []string{"광고 운영", "광고운영", "paid ads", "광고 캠페인", "google ads", "퍼포먼스 광고"}, []string{"marketing", "campaign", "operations", "analysis"}, 30},
	{"SEO", []string{"seo", "검색엔진 최적화", "검색 최적화"}, []string{"marketing", "web", "content", "analysis"}, 30},
	{"콘텐츠 제작", []string{"콘텐츠 제작", "콘텐츠제작", "content creation", "카피라이팅", "콘텐츠 기획"}, []string{"marketing", "content", "writing", "design"}, 25},
	{"영업", []string{"영업", "sales", "세일즈"}, []string{"business", "customer", "negotiation", "revenue"}, 35},
	{"협상", []string{"협상", "negotiation"}, []string{"business", "communication", "negotiation", "people"}, 30},
	{"CRM", []string{"crm", "고객관계관리", "salesforce", "hubspot"}, []string{"business", "customer", "data", "operations"}, 25},
	{"고객 응대", []string{"고객 응대", "고객응대", "customer support", "고객 상담", "고객 지원", "cs 업무"}, []string{"customer", "communication", "support", "people"}, 25},
	{"운영 관리", []string{"운영 관리", "운영관리", "operations management", "서비스 운영"}, []string{"operations", "business", "process", "planning"}, 35},
	{"프로세스 개선", []string{"프로세스 개선", "process improvement", "업무 개선", "업무 효율화"}, []string{"operations", "process", "analysis", "quality"}, 35},
	{"인사 관리", []string{"인사 관리", "인사관리", "human resources", "hr 업무", "채용 운영"}, []string{"business", "people", "operations", "compliance"}, 35},
	{"교육 설계", []string{"교육 설계", "교육설계", "instructional design", "교육 기획"}, []string{"people", "planning", "design", "learning"}, 30},
	{"노무 기초", []string{"노무", "노동법", "labor relations"}, []string{"people", "compliance", "risk", "business"}, 40},
	{"회계", []string{"회계", "accounting", "부기", "결산"}, []string{"finance", "business", "recordkeeping", "compliance"}, 50},
	{"ERP", []string{"erp", "sap", "전사적 자원관리"}, []string{"business", "data", "operations", "integration"}, 35},
	{"재무 분석", []string{"재무 분석", "재무분석", "financial analysis"}, []string{"finance", "analysis", "data", "business"}, 40},
	{"세무 기초", []string{"세무", "taxation", "tax accounting"}, []string{"finance", "compliance", "recordkeeping", "business"}, 45},
	{"물류 관리", []string{"물류 관리", "물류관리", "logistics", "물류 운영"}, []string{"operations", "supplychain", "planning", "process"}, 35},
	{"재고 관리", []string{"재고 관리", "재고관리", "inventory management", "창고 관리"}, []string{"operations", "supplychain", "recordkeeping", "data"}, 30},
	{"위험 관리", []string{"위험 관리", "리스크 관리", "risk management"}, []string{"risk", "business", "analysis", "compliance"}, 35},
}

func key(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || strings.ContainsRune("_-/.", r) {
			return -1
		}
		return unicode.ToLower(r)
	}, strings.TrimSpace(s))
}

var skillAliasIndex = func() map[string]string {
	index := map[string]string{}
	for _, d := range skillDefinitions {
		index[key(d.name)] = d.name
		for _, alias := range d.aliases {
			index[key(alias)] = d.name
		}
	}
	return index
}()

var skillDefinitionIndex = func() map[string]skillDefinition {
	index := map[string]skillDefinition{}
	for _, d := range skillDefinitions {
		index[d.name] = d
	}
	return index
}()

// NormalizeSkill maps spelling variants to a known canonical name, retaining
// unknown terms verbatim so administrator imports are not silently discarded.
func NormalizeSkill(name string) string {
	if canonical, ok := skillAliasIndex[key(name)]; ok {
		return canonical
	}
	return strings.TrimSpace(name)
}

func learningHours(name string) float64 {
	if d, ok := skillDefinitionIndex[NormalizeSkill(name)]; ok {
		return d.hours
	}
	return 40
}

func features(name string) []string { return skillDefinitionIndex[name].features }

// SkillSimilarity is cosine similarity in the documented local feature space.
// Exact aliases score 1; unrelated or unknown terms never gain fuzzy credit.
func SkillSimilarity(a, b string) float64 {
	a, b = NormalizeSkill(a), NormalizeSkill(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	af, bf := features(a), features(b)
	if len(af) == 0 || len(bf) == 0 {
		return 0
	}
	overlap := 0
	for _, x := range af {
		for _, y := range bf {
			if x == y {
				overlap++
			}
		}
	}
	return float64(overlap) / math.Sqrt(float64(len(af)*len(bf)))
}

func verifiedSkills(profile Profile) map[string]float64 {
	result := map[string]float64{}
	for _, s := range profile.Skills {
		if s.Confidence != "" && s.Confidence != "explicit" {
			continue
		}
		name := NormalizeSkill(s.Name)
		if name != "" {
			result[name] = math.Max(result[name], bounded(s.Level, 0, 5))
		}
	}
	return result
}

func mergeSkills(profile Profile, additions []Skill) Profile {
	merged := make([]Skill, len(profile.Skills))
	copy(merged, profile.Skills)
	for _, s := range additions {
		s.Name = NormalizeSkill(s.Name)
		if s.Name == "" {
			continue
		}
		s.Level = bounded(s.Level, 0, 5)
		s.Confidence = "explicit"
		merged = append(merged, s)
	}
	profile.Skills = merged
	return profile
}

func sortedSkillNames(skills map[string]float64) []string {
	names := make([]string, 0, len(skills))
	for n := range skills {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
