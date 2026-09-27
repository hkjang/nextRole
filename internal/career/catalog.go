package career

// The included catalog is an authored demonstration model, not a copy of NCS
// or Work24. Every built-in record retains synthetic=true when displayed/exported.
var demoSource = Source{Name: "NextRole 오프라인 합성 직무 모델 v1", Synthetic: true}

func req(name string, level, weight float64) Requirement { return Requirement{name, level, weight} }

func SeedJobs() []Job {
	jobs := []Job{
		{ID: "ai-platform", Title: "AI 플랫폼 엔지니어", Category: "개발·인프라", Description: "모델 서빙, GPU 자원과 배포 플랫폼을 안정적으로 운영합니다.", MinExperience: 3, Domain: "IT", Education: "학사", BridgeIDs: []string{"devops", "ai-backend", "data-engineer"}, Skills: []Requirement{req("Python", 4, 1.2), req("Linux", 4, 1), req("Docker", 4, 1), req("Kubernetes", 4, 1.3), req("GPU 운영", 3, 1), req("LLM Serving", 4, 1.3), req("vLLM", 3, .8), req("Observability", 3, .8), req("Backend API", 4, 1), req("CI/CD", 3, .7)}},
		{ID: "data-engineer", Title: "데이터 엔지니어", Category: "데이터·AI", Description: "데이터 파이프라인과 데이터 품질·저장 구조를 설계합니다.", MinExperience: 2, Domain: "IT", Education: "학사", BridgeIDs: []string{"backend", "data-analyst"}, Skills: []Requirement{req("Python", 4, 1.2), req("SQL", 4, 1.3), req("Data Pipeline", 4, 1.3), req("Spark", 3, 1), req("Airflow", 3, .8), req("Docker", 3, .7), req("데이터 모델링", 4, 1), req("Linux", 3, .6)}},
		{ID: "devops", Title: "DevOps 엔지니어", Category: "개발·인프라", Description: "소프트웨어 배포를 자동화하고 서비스 신뢰성을 관리합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"backend", "cloud"}, Skills: []Requirement{req("Linux", 4, 1.2), req("Docker", 4, 1), req("Kubernetes", 4, 1.3), req("CI/CD", 4, 1.2), req("Observability", 3, 1), req("Terraform", 3, .8), req("네트워크", 3, .8), req("Python", 2, .6)}},
		{ID: "ml-engineer", Title: "머신러닝 엔지니어", Category: "데이터·AI", Description: "모델 학습, 평가와 운영 중 성능을 관리합니다.", MinExperience: 3, Domain: "IT", Education: "학사", BridgeIDs: []string{"data-engineer", "data-analyst", "ai-backend"}, Skills: []Requirement{req("Python", 4, 1.2), req("머신러닝", 4, 1.4), req("통계", 4, 1.1), req("PyTorch", 4, 1.2), req("SQL", 3, .7), req("MLOps", 3, 1), req("데이터 분석", 3, 1)}},
		{ID: "ai-backend", Title: "AI 백엔드 개발자", Category: "개발·인프라", Description: "생성형 AI 기능을 제품 API와 검색 흐름에 연결합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"backend", "fullstack"}, Skills: []Requirement{req("Backend API", 4, 1.2), req("Python", 4, 1.1), req("FastAPI", 3, .8), req("LLM API", 4, 1.2), req("RAG", 3, 1), req("SQL", 3, .8), req("Docker", 3, .6), req("보안", 3, .7)}},
		{ID: "backend", Title: "백엔드 개발자", Category: "개발·인프라", Description: "안정적인 서비스 API와 업무 데이터 시스템을 개발합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"qa", "fullstack"}, Skills: []Requirement{req("Java", 4, 1), req("Spring", 4, 1), req("Backend API", 4, 1.2), req("SQL", 4, 1), req("Git", 3, .5), req("Docker", 3, .6), req("테스트 자동화", 3, .8)}},
		{ID: "frontend", Title: "프론트엔드 개발자", Category: "개발·인프라", Description: "웹 사용자 인터페이스와 접근성·성능을 개선합니다.", MinExperience: 1, Domain: "IT", Education: "무관", BridgeIDs: []string{"ux-designer", "qa"}, Skills: []Requirement{req("JavaScript", 4, 1.2), req("TypeScript", 3, 1), req("React", 4, 1.2), req("HTML/CSS", 4, 1), req("웹 접근성", 3, .8), req("Git", 3, .6), req("테스트 자동화", 3, .7)}},
		{ID: "fullstack", Title: "풀스택 개발자", Category: "개발·인프라", Description: "웹 화면부터 API, 데이터베이스까지 제품을 구현합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"frontend", "backend"}, Skills: []Requirement{req("TypeScript", 4, 1), req("React", 3, 1), req("Backend API", 4, 1.2), req("SQL", 3, 1), req("Git", 3, .5), req("Docker", 3, .7), req("테스트 자동화", 3, .7)}},
		{ID: "cloud", Title: "클라우드 엔지니어", Category: "개발·인프라", Description: "클라우드 인프라의 운영, 비용과 보안을 관리합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"it-support", "devops"}, Skills: []Requirement{req("Linux", 4, 1), req("AWS", 4, 1.2), req("네트워크", 4, 1.2), req("Terraform", 3, 1), req("보안", 3, .8), req("Observability", 3, .7), req("Docker", 3, .6)}},
		{ID: "security", Title: "정보보안 담당자", Category: "개발·인프라", Description: "서비스 위험을 점검하고 보안 정책과 사고 대응을 운영합니다.", MinExperience: 2, Domain: "IT", Education: "무관", BridgeIDs: []string{"it-support", "cloud"}, Skills: []Requirement{req("보안", 4, 1.4), req("네트워크", 4, 1.2), req("Linux", 3, 1), req("위험 관리", 4, 1), req("문서 작성", 3, .8), req("Python", 2, .5)}},
		{ID: "qa", Title: "QA·테스트 엔지니어", Category: "개발·인프라", Description: "사용자 시나리오에 따른 품질 검증과 테스트 자동화를 수행합니다.", MinExperience: 1, Domain: "IT", Education: "무관", BridgeIDs: []string{"customer-success", "it-support"}, Skills: []Requirement{req("테스트 자동화", 4, 1.4), req("품질 관리", 4, 1.1), req("SQL", 3, .8), req("Python", 3, .8), req("문서 작성", 3, .7), req("문제 해결", 4, 1)}},
		{ID: "data-analyst", Title: "데이터 분석가", Category: "데이터·AI", Description: "데이터를 분석해 의사결정의 근거와 실험 결과를 전달합니다.", MinExperience: 1, Domain: "분석", Education: "학사", BridgeIDs: []string{"business-analyst", "digital-marketer"}, Skills: []Requirement{req("SQL", 4, 1.3), req("Python", 3, 1), req("데이터 분석", 4, 1.3), req("통계", 3, 1), req("데이터 시각화", 3, .9), req("커뮤니케이션", 3, .6), req("Excel", 3, .6)}},
		{ID: "business-analyst", Title: "비즈니스 분석가", Category: "기획·운영", Description: "업무 요구를 구조화하고 지표를 통해 개선 기회를 찾습니다.", MinExperience: 2, Domain: "비즈니스", Education: "무관", BridgeIDs: []string{"operations", "data-analyst"}, Skills: []Requirement{req("데이터 분석", 3, 1.2), req("SQL", 3, .8), req("요구사항 분석", 4, 1.2), req("문서 작성", 4, 1), req("커뮤니케이션", 4, 1), req("Excel", 4, .8), req("프로젝트 관리", 3, .7)}},
		{ID: "product-manager", Title: "프로덕트 매니저", Category: "기획·운영", Description: "고객 문제를 정의하고 제품 가설과 실행 우선순위를 관리합니다.", MinExperience: 3, Domain: "비즈니스", Education: "무관", BridgeIDs: []string{"business-analyst", "customer-success", "ux-designer"}, Skills: []Requirement{req("제품 기획", 4, 1.3), req("고객 조사", 4, 1.1), req("데이터 분석", 3, 1), req("프로젝트 관리", 4, 1), req("커뮤니케이션", 4, 1), req("요구사항 분석", 4, 1), req("SQL", 2, .5)}},
		{ID: "ux-designer", Title: "UX·UI 디자이너", Category: "디자인", Description: "사용자 조사 결과를 인터페이스와 검증 가능한 프로토타입으로 만듭니다.", MinExperience: 1, Domain: "디자인", Education: "무관", BridgeIDs: []string{"content-marketer", "frontend"}, Skills: []Requirement{req("Figma", 4, 1.2), req("UX 디자인", 4, 1.4), req("고객 조사", 3, 1), req("프로토타이핑", 4, 1), req("웹 접근성", 3, .8), req("커뮤니케이션", 3, .7)}},
		{ID: "digital-marketer", Title: "디지털 마케터", Category: "마케팅·영업", Description: "고객 유입과 전환을 측정하면서 디지털 캠페인을 운영합니다.", MinExperience: 1, Domain: "마케팅", Education: "무관", BridgeIDs: []string{"content-marketer", "sales"}, Skills: []Requirement{req("디지털 마케팅", 4, 1.3), req("데이터 분석", 3, 1), req("광고 운영", 4, 1.1), req("SEO", 3, .8), req("콘텐츠 제작", 3, .8), req("Excel", 3, .7), req("고객 조사", 3, .8)}},
		{ID: "content-marketer", Title: "콘텐츠 마케터", Category: "마케팅·영업", Description: "고객의 검색 의도에 맞는 콘텐츠를 기획하고 성과를 분석합니다.", MinExperience: 1, Domain: "마케팅", Education: "무관", BridgeIDs: []string{"customer-success", "digital-marketer"}, Skills: []Requirement{req("콘텐츠 제작", 4, 1.4), req("문서 작성", 4, 1), req("SEO", 3, 1), req("고객 조사", 3, .8), req("데이터 분석", 2, .7), req("커뮤니케이션", 3, .8)}},
		{ID: "sales", Title: "B2B 영업 담당자", Category: "마케팅·영업", Description: "고객 요구를 이해하고 제안·협상과 거래 관계를 관리합니다.", MinExperience: 1, Domain: "비즈니스", Education: "무관", BridgeIDs: []string{"customer-success", "operations"}, Skills: []Requirement{req("영업", 4, 1.4), req("커뮤니케이션", 4, 1.2), req("협상", 4, 1), req("CRM", 3, .9), req("문서 작성", 3, .7), req("고객 조사", 3, .8)}},
		{ID: "customer-success", Title: "고객 성공 매니저", Category: "기획·운영", Description: "고객 온보딩과 문제 해결을 통해 서비스 이용 성과를 높입니다.", MinExperience: 1, Domain: "서비스", Education: "무관", BridgeIDs: []string{"sales", "operations"}, Skills: []Requirement{req("고객 응대", 4, 1.3), req("커뮤니케이션", 4, 1.2), req("문제 해결", 4, 1), req("CRM", 3, .8), req("문서 작성", 3, .8), req("데이터 분석", 2, .6)}},
		{ID: "operations", Title: "서비스 운영 매니저", Category: "기획·운영", Description: "반복 업무를 표준화하고 운영 지표와 고객 품질을 관리합니다.", MinExperience: 2, Domain: "서비스", Education: "무관", BridgeIDs: []string{"customer-success", "business-analyst"}, Skills: []Requirement{req("운영 관리", 4, 1.3), req("Excel", 4, 1), req("프로세스 개선", 4, 1.1), req("커뮤니케이션", 4, 1), req("문제 해결", 3, .9), req("문서 작성", 3, .7)}},
		{ID: "hr", Title: "인사·교육 담당자", Category: "경영지원", Description: "채용·교육과 구성원 경험을 개선하고 인사 데이터를 관리합니다.", MinExperience: 1, Domain: "인사", Education: "무관", BridgeIDs: []string{"operations", "customer-success"}, Skills: []Requirement{req("인사 관리", 4, 1.3), req("커뮤니케이션", 4, 1.1), req("교육 설계", 3, 1), req("Excel", 3, .8), req("문서 작성", 4, .9), req("노무 기초", 3, .8)}},
		{ID: "accounting", Title: "회계·재무 담당자", Category: "경영지원", Description: "회계 기록과 결산 자료를 관리하고 재무 흐름을 분석합니다.", MinExperience: 1, Domain: "금융", Education: "무관", BridgeIDs: []string{"operations", "business-analyst"}, Skills: []Requirement{req("회계", 4, 1.5), req("Excel", 4, 1), req("ERP", 3, 1), req("재무 분석", 3, 1.1), req("문서 작성", 3, .7), req("세무 기초", 3, .9)}},
		{ID: "logistics", Title: "물류·공급망 운영 담당자", Category: "제조·물류", Description: "재고와 납기, 물류 흐름을 지표로 관리하고 병목을 줄입니다.", MinExperience: 1, Domain: "물류", Education: "무관", BridgeIDs: []string{"operations", "quality"}, Skills: []Requirement{req("물류 관리", 4, 1.3), req("재고 관리", 4, 1.1), req("Excel", 4, .9), req("ERP", 3, .9), req("프로세스 개선", 3, .9), req("커뮤니케이션", 3, .7)}},
		{ID: "quality", Title: "제조 품질 관리 담당자", Category: "제조·물류", Description: "품질 기준을 운영하고 불량 원인과 개선 결과를 추적합니다.", MinExperience: 2, Domain: "제조", Education: "무관", BridgeIDs: []string{"logistics", "operations"}, Skills: []Requirement{req("품질 관리", 4, 1.4), req("통계", 3, 1), req("Excel", 3, .9), req("프로세스 개선", 4, 1.1), req("문제 해결", 4, 1), req("문서 작성", 3, .7)}},
		{ID: "it-support", Title: "IT 지원 담당자", Category: "개발·인프라", Description: "업무 IT 환경의 장애를 진단하고 사용자를 지원합니다.", MinExperience: 0, Domain: "IT", Education: "무관", BridgeIDs: []string{"customer-success", "operations"}, Skills: []Requirement{req("문제 해결", 4, 1.2), req("네트워크", 3, 1), req("Linux", 2, .8), req("고객 응대", 3, 1), req("문서 작성", 3, .8), req("보안", 2, .7)}},
	}
	for i := range jobs {
		jobs[i].Source = demoSource
		jobs[i].Regions = []string{"전국", "서울", "경기", "원격"}
		jobs[i].SalaryRange = "합성 직무 모델: 실측 임금 데이터 없음"
	}
	return jobs
}

// SeedOpportunities returns transparently synthetic, non-application examples.
// Empty URLs are intentional: no fabricated job advertisement or course link.
func SeedOpportunities(job Job) (jobs []Opportunity, training []Opportunity) {
	jobs, training = []Opportunity{}, []Opportunity{}
	source := Source{Name: "NextRole 합성 예시 · 실제 공고·훈련과정 아님", Synthetic: true}
	skills := []string{}
	for _, r := range job.Skills {
		skills = append(skills, r.Name)
	}
	jobs = append(jobs, Opportunity{ID: "demo-job-" + job.ID, Title: "[합성 예시] " + job.Title + " 채용 요건", Organization: "가상 조직", Region: "전국", Skills: skills, Source: source, Description: "화면과 연동 구조를 확인하는 합성 예시입니다. 실제 채용공고가 아니며 지원할 수 없습니다. 관리자가 고용24 또는 조직 데이터 연동을 설정하면 실제 공고를 조회할 수 있습니다."})
	for i, r := range job.Skills {
		if i >= 5 {
			break
		}
		training = append(training, Opportunity{ID: "demo-training-" + job.ID + "-" + NormalizeSkill(r.Name), Title: "[합성 예시] " + r.Name + " 실습 계획", Organization: "NextRole 예시 학습안", Region: "온라인", Skills: []string{r.Name}, Source: source, Description: "실제 개설·모집 중인 교육과정이 아닌 학습 주제 예시입니다. 교육비와 일정은 연동된 실제 교육과정에서 확인하세요."})
	}
	return
}
