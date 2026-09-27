package connectors

import "time"

const (
	Work24Occupation = "work24-occupation-summary"
	Work24NCS        = "work24-ncs"
	Work24Jobs       = "work24-jobs"
	Work24Training   = "work24-training"
)

// FieldMetadata is an allowlist entry, not an instruction to retain a response
// verbatim. Nested official groups are projected onto their documented children.
type FieldMetadata struct {
	Name            string `json:"name"`
	Label           string `json:"label"`
	Description     string `json:"description"`
	Required        bool   `json:"required"`
	DefaultSelected bool   `json:"defaultSelected"`
}

type PresetMetadata struct {
	ID                    string            `json:"id"`
	Name                  string            `json:"name"`
	Dataset               string            `json:"dataset"`
	Description           string            `json:"description"`
	SpecURL               string            `json:"specUrl"`
	Endpoint              string            `json:"endpoint"`
	RootPath              string            `json:"rootPath"`
	Format                string            `json:"format"`
	Fields                []FieldMetadata   `json:"fields"`
	DefaultSelectedFields []string          `json:"defaultSelectedFields"`
	RequiredParams        []string          `json:"requiredParams"`
	DefaultParams         map[string]string `json:"defaultParams"`
}

func field(name, label string, required, selected bool) FieldMetadata {
	return FieldMetadata{Name: name, Label: label, Description: label, Required: required, DefaultSelected: selected || required}
}

// Work24Metadata describes the four official contracts checked on 2026-09-27.
// The authKey is always supplied through Config.APIKey, never in DefaultParams.
func Work24Metadata() []PresetMetadata { return work24MetadataAt(time.Now()) }

func work24MetadataAt(now time.Time) []PresetMetadata {
	base := "https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?fullApiSvcId="
	out := []PresetMetadata{
		{
			ID: Work24Occupation, Name: "고용24 직업정보 상세 · 요약", Dataset: "occupation_details", Format: "xml",
			Description: "직업코드별 공식 요약을 별도 원천자료로 보존합니다. 역량 수준·가중치는 자동 생성하지 않습니다.",
			SpecURL:     base + "000000000000000000000000000093%5E000000000000000000000000000090",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo212D01.do", RootPath: "jobSum",
			RequiredParams: []string{"returnType", "target", "jobGb", "dtlGb", "jobCd"},
			DefaultParams:  map[string]string{"returnType": "XML", "target": "JOBDTL", "jobGb": "1", "dtlGb": "1", "jobCd": ""},
			Fields: []FieldMetadata{
				field("jobCd", "직업코드", true, true), field("jobSmclNm", "직업 소분류명", true, true),
				field("jobLrclNm", "직업 대분류명", false, true), field("jobMdclNm", "직업 중분류명", false, true),
				field("jobSum", "하는 일", false, true), field("way", "되는 길", false, true),
				field("relMajorList", "관련 전공(코드·이름)", false, true), field("relCertList", "관련 자격증", false, true),
				field("sal", "임금 참고", false, true), field("jobSatis", "직업 만족도", false, false),
				field("jobProspect", "일자리 전망", false, true), field("jobStatus", "일자리 현황", false, false),
				field("jobAbil", "업무수행능력 원문", false, true), field("knowldg", "지식 원문", false, true),
				field("jobEnv", "업무 환경", false, false), field("jobChr", "성격", false, false),
				field("jobIntrst", "흥미", false, false), field("jobVals", "직업 가치관", false, false),
				field("jobActvImprtncs", "업무활동 중요도 원문", false, false), field("jobActvLvls", "업무활동 수준 원문", false, false),
				field("relJobList", "관련 직업(코드·이름)", false, true),
			},
		},
		{
			ID: Work24NCS, Name: "고용24 표준직무기술서 · NCS 능력단위", Dataset: "ncs_units", Format: "json",
			Description: "수행직무내용으로 조회한 NCS 능력단위를 원천자료로 보존합니다. 공식 분류와 서비스의 0~5 역량 수준은 구분합니다.",
			SpecURL:     "https://www.work24.go.kr/cm/e/a/0110/selectOpenApiSvcInfo.do?apiSvcId=000000000000000000000000000087&fullApiSvcId=000000000000000000000000000096%5E000000000000000000000000000084&upprApiSvcId=000000000000000000000000000086",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo215L01.do", RootPath: "result",
			RequiredParams: []string{"jobCont"}, DefaultParams: map[string]string{"jobCont": "", "limit": "5", "returnType": "JSON"},
			Fields: []FieldMetadata{
				field("ablt_unit", "NCS 능력단위 코드", true, true), field("job_sdvn", "NCS 능력단위명", true, true),
				field("ablt_def", "능력단위 정의", false, true), field("knwg_tchn_attd", "지식·기술·태도 원문", false, true),
				field("job_lrcl_cd", "대분류코드", false, true), field("job_lcfn", "대분류명", false, true),
				field("job_mlsf_cd", "중분류코드", false, true), field("job_mcn", "중분류명", false, true),
				field("job_scla_cd", "소분류코드", false, true), field("job_scfn", "소분류명", false, true),
				field("job_sdvn_cd", "세분류코드", false, true),
			},
		},
		{
			ID: Work24Jobs, Name: "고용24 채용정보", Dataset: "jobs", Format: "xml",
			Description: "공식 채용 목록과 원문 링크를 보존합니다. 목록에 없는 요구기술은 추론하지 않습니다.",
			SpecURL:     base + "000000000000000000000000000000",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/wk/callOpenApiSvcInfo210L01.do", RootPath: "wantedRoot.wanted",
			RequiredParams: []string{"callTp", "returnType", "startPage", "display"},
			DefaultParams:  map[string]string{"callTp": "L", "returnType": "XML", "startPage": "1", "display": "100"},
			Fields: []FieldMetadata{
				field("wantedAuthNo", "구인인증번호", true, true), field("title", "채용 제목", true, true),
				field("company", "회사명", true, true), field("wantedInfoUrl", "채용 원문 URL", true, true),
				field("jobsCd", "직종코드", false, true), field("region", "근무 지역", false, true),
				field("salTpNm", "임금 형태", false, true), field("sal", "급여 참고", false, true),
				field("minSal", "최소 임금액", false, false), field("maxSal", "최대 임금액", false, false),
				field("holidayTpNm", "근무 형태", false, false), field("minEdubg", "최소 학력", false, true),
				field("maxEdubg", "최대 학력", false, false), field("career", "경력 조건", false, true),
				field("regDt", "등록일", false, true), field("closeDt", "마감일", false, true),
				field("infoSvc", "정보 제공처", false, true), field("indTpNm", "업종", false, false),
				field("busino", "사업자등록번호", false, false), field("wantedMobileInfoUrl", "모바일 채용 URL", false, false),
				field("zipCd", "근무지 우편번호", false, false), field("strtnmCd", "도로명주소", false, false),
				field("basicAddr", "기본 주소", false, false), field("detailAddr", "상세 주소", false, false),
				field("empTpCd", "고용형태코드", false, true), field("smodifyDtm", "최종 수정일", false, false),
			},
		},
		{
			ID: Work24Training, Name: "고용24 국민내일배움카드 훈련", Dataset: "training", Format: "xml",
			Description: "과정 ID와 회차를 함께 보존합니다. 훈련비 원문은 참고이며 개인별 확정 본인부담금이 아닙니다.",
			SpecURL:     base + "000000000000000000000000000004",
			Endpoint:    "https://www.work24.go.kr/cm/openApi/call/hr/callOpenApiSvcInfo310L01.do", RootPath: "HRDNet.srchList.scn_list",
			RequiredParams: []string{"returnType", "outType", "pageNum", "pageSize", "srchTraStDt", "srchTraEndDt", "sort", "sortCol"},
			DefaultParams:  map[string]string{"returnType": "XML", "outType": "1", "pageNum": "1", "pageSize": "100", "srchTraStDt": now.Format("20060102"), "srchTraEndDt": now.AddDate(0, 3, 0).Format("20060102"), "sort": "ASC", "sortCol": "2"},
			Fields: []FieldMetadata{
				field("trprId", "훈련과정 ID", true, true), field("trprDegr", "훈련과정 회차", true, true),
				field("title", "훈련과정명", true, true), field("subTitle", "훈련기관명", true, true), field("titleLink", "훈련 원문 URL", true, true),
				field("ncsCd", "NCS 코드", false, true), field("address", "주소", false, true),
				field("traStartDate", "훈련 시작일", false, true), field("traEndDate", "훈련 종료일", false, true),
				field("courseMan", "수강비 참고(확정 본인부담금 아님)", false, true), field("realMan", "실제 훈련비 참고(확정 본인부담금 아님)", false, true),
				field("contents", "과정 내용", false, true), field("certificate", "자격증 원문", false, true),
				field("trainTarget", "훈련 대상", false, true), field("trainTargetCd", "훈련 구분", false, false),
				field("instCd", "훈련기관 코드", false, false), field("trainstCstId", "훈련기관 ID", false, false),
				field("trngAreaCd", "지역코드", false, false), field("subTitleLink", "기관 링크", false, false),
				field("telNo", "기관 전화번호", false, false), field("titleIcon", "제목 아이콘", false, false),
				field("wkendSe", "주말·주중 구분", false, false), field("yardMan", "정원", false, false),
				field("regCourseMan", "수강신청 인원", false, false), field("stdgScor", "만족도 점수", false, false),
				field("grade", "등급", false, false), field("eiEmplCnt3", "고용보험 3개월 취업인원", false, false),
				field("eiEmplRate3", "고용보험 3개월 취업률", false, false), field("eiEmplRate6", "고용보험 6개월 취업률", false, false),
				// eiEmplCnt3Gt10 is a retired null-only compatibility field.
			},
		},
	}
	for i := range out {
		out[i].DefaultSelectedFields = []string{}
		for _, f := range out[i].Fields {
			if f.DefaultSelected {
				out[i].DefaultSelectedFields = append(out[i].DefaultSelectedFields, f.Name)
			}
		}
	}
	return out
}

// Presets returns disabled, credential-free starter configurations. Required
// jobCd/jobCont values deliberately remain empty until an administrator supplies
// a real query. Mapping is not used by Work24 adapters; selectedFields is.
func Presets() []Config { return Work24Presets(time.Now()) }
func Work24Presets(now time.Time) []Config {
	out := []Config{}
	for _, m := range work24MetadataAt(now) {
		out = append(out, Config{PresetID: m.ID, Name: m.Name, Type: m.Format, Dataset: m.Dataset, Method: "GET", Endpoint: m.Endpoint, APIKeyParam: "authKey", RootPath: m.RootPath, Params: m.DefaultParams, Mapping: map[string]string{}, SelectedFields: m.DefaultSelectedFields})
	}
	return out
}
