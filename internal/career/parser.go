package career

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var yearPattern = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)\s*(?:년|years?|yrs?)`)
var overallYears = regexp.MustCompile(`(?i)(?:총\s*경력|경력|experience)\s*[:：]?\s*(\d+(?:\.\d+)?)\s*(?:년|years?|yrs?)`)
var levelPattern = regexp.MustCompile(`(?i)(?:level|레벨|수준)\s*[:：]?\s*([0-5])`)

// Parse extracts only locally recognizable evidence. Inferred capabilities are
// marked review and excluded from scoring until explicitly confirmed by a user.
func Parse(text string) Profile {
	if len(text) > 1<<20 {
		text = text[:1<<20]
		for !utf8.ValidString(text) {
			text = text[:len(text)-1]
		}
	}
	p := Profile{Narrative: text, Skills: []Skill{}, Certifications: []string{}, Preferences: []string{}, WeeklyHours: 6}
	lower := strings.ToLower(text)
	if m := overallYears.FindStringSubmatch(lower); len(m) > 1 {
		p.YearsExperience, _ = strconv.ParseFloat(m[1], 64)
	} else {
		for _, m := range yearPattern.FindAllStringSubmatch(lower, -1) {
			y, _ := strconv.ParseFloat(m[1], 64)
			if y > p.YearsExperience {
				p.YearsExperience = y
			}
		}
	}
	p.YearsExperience = bounded(p.YearsExperience, 0, 80)
	for _, d := range skillDefinitions {
		bestLevel, bestYears := 0.0, 0.0
		found := false
		for _, alias := range d.aliases {
			for _, pos := range occurrences(lower, strings.ToLower(alias)) {
				// Negated or aspirational mentions are not possession evidence.
				tail := runePrefix(lower[pos+len(alias):], 24)
				if negated(tail) {
					continue
				}
				found = true
				years := 0.0
				if m := yearPattern.FindStringSubmatch(runePrefix(tail, 14)); len(m) > 1 {
					years, _ = strconv.ParseFloat(m[1], 64)
				}
				level := 3.0
				if years > 0 {
					switch {
					case years < 1:
						level = 1
					case years < 2:
						level = 2
					case years < 5:
						level = 3
					case years < 8:
						level = 4
					default:
						level = 5
					}
				}
				near := runePrefix(tail, 14)
				// Stop at the next punctuation-delimited skill to avoid borrowing its level.
				if i := strings.IndexAny(near, ",;\n"); i >= 0 {
					near = near[:i]
				}
				switch {
				case containsAny(near, "expert", "전문가", "전문 수준"):
					level = 5
				case containsAny(near, "advanced", "고급", "상급", "숙련"):
					level = 4
				case containsAny(near, "intermediate", "중급"):
					level = 3
				case containsAny(near, "basic", "기초", "초급"):
					level = 2
				case containsAny(near, "beginner", "입문"):
					level = 1
				}
				if m := levelPattern.FindStringSubmatch(near); len(m) > 1 {
					level, _ = strconv.ParseFloat(m[1], 64)
				}
				if level > bestLevel {
					bestLevel = level
				}
				if years > bestYears {
					bestYears = years
				}
			}
		}
		if found {
			p.Skills = append(p.Skills, Skill{Name: d.name, Level: bestLevel, Years: bounded(bestYears, 0, 80), Confidence: "explicit"})
		}
	}
	roleRules := []struct {
		terms []string
		role  string
	}{
		{[]string{"ai platform", "ai 플랫폼"}, "AI 플랫폼 엔지니어"}, {[]string{"data engineer", "데이터 엔지니어"}, "데이터 엔지니어"},
		{[]string{"devops"}, "DevOps 엔지니어"}, {[]string{"backend", "백엔드"}, "백엔드 개발자"}, {[]string{"frontend", "프론트엔드"}, "프론트엔드 개발자"},
		{[]string{"data analyst", "데이터 분석가"}, "데이터 분석가"}, {[]string{"마케터", "marketing", "마케팅"}, "마케팅 담당자"},
		{[]string{"디자이너", "designer"}, "디자이너"}, {[]string{"영업", "sales"}, "영업 담당자"}, {[]string{"회계", "accounting"}, "회계 담당자"},
		{[]string{"물류", "logistics"}, "물류 운영 담당자"}, {[]string{"인사", "human resources"}, "인사 담당자"},
		{[]string{"서비스 운영", "운영 관리"}, "서비스 운영 담당자"}, {[]string{"개발자", "developer"}, "소프트웨어 개발자"},
	}
	firstRole := len(lower) + 1
	for _, r := range roleRules {
		for _, term := range r.terms {
			for _, pos := range occurrences(lower, term) {
				if pos < firstRole {
					p.CurrentRole = r.role
					firstRole = pos
				}
			}
		}
	}

	for _, e := range []string{"박사", "석사", "학사", "전문학사", "고졸", "대졸"} {
		if strings.Contains(lower, e) {
			p.Education = e
			break
		}
	}
	// A professional associate degree contains 학사; restore its precise level.
	if strings.Contains(lower, "전문학사") && !strings.Contains(lower, "석사") && !strings.Contains(lower, "박사") {
		p.Education = "전문학사"
	}
	for _, region := range []string{"서울", "경기", "인천", "부산", "대전", "대구", "광주", "울산", "세종", "제주", "강원", "충북", "충남", "전북", "전남", "경북", "경남", "원격"} {
		if strings.Contains(lower, region) {
			p.Region = region
			break
		}
	}
	for _, domain := range []string{"금융", "제조", "물류", "의료", "교육", "인사", "마케팅", "서비스", "it"} {
		if len(occurrences(lower, domain)) > 0 {
			p.Domain = domain
			break
		}
	}
	for _, cert := range []string{"정보처리기사", "정보보안기사", "빅데이터분석기사", "SQLD", "SQLP", "ADsP", "ADP", "PMP", "CKA", "CKAD", "AWS SAA", "전산회계", "전산세무", "직업상담사", "물류관리사"} {
		if len(occurrences(lower, strings.ToLower(cert))) > 0 {
			p.Certifications = append(p.Certifications, cert)
		}
	}
	hints := []struct {
		terms []string
		skill string
	}{
		{[]string{"백엔드 개발", "backend developer"}, "Backend API"},
		{[]string{"배포 파이프라인", "배포 자동화"}, "CI/CD"},
		{[]string{"팀 리딩", "팀장", "프로젝트 리더"}, "프로젝트 관리"},
		{[]string{"부서 간 협업", "고객과 협의"}, "커뮤니케이션"},
	}
	for _, hint := range hints {
		if !containsAny(lower, hint.terms...) {
			continue
		}
		exists := false
		for _, s := range p.Skills {
			if s.Name == hint.skill {
				exists = true
			}
		}
		if !exists {
			p.Skills = append(p.Skills, Skill{Name: hint.skill, Level: 1, Confidence: "review"})
		}
	}
	sort.SliceStable(p.Skills, func(i, j int) bool { return p.Skills[i].Name < p.Skills[j].Name })
	return p
}

func containsAny(s string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(s, term) {
			return true
		}
	}
	return false
}
func runePrefix(s string, n int) string {
	r := []rune(s)
	if len(r) > n {
		r = r[:n]
	}
	return string(r)
}

func occurrences(s, term string) []int {
	indexes := []int{}
	offset := 0
	if term == "" {
		return indexes
	}
	latin := false
	for _, r := range term {
		if r > 127 {
			latin = false
			break
		}
		if unicode.IsLetter(r) {
			latin = true
		}
	}
	for offset < len(s) {
		rel := strings.Index(s[offset:], term)
		if rel < 0 {
			break
		}
		i := offset + rel
		end := i + len(term)
		valid := true
		if latin {
			if i > 0 {
				r, _ := utf8.DecodeLastRuneInString(s[:i])
				if r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') {
					valid = false
				}
			}
			if end < len(s) {
				r, _ := utf8.DecodeRuneInString(s[end:])
				if r < 128 && (unicode.IsLetter(r) || r == '_') {
					valid = false
				}
			}
		}
		if valid {
			indexes = append(indexes, i)
		}
		offset = end
	}
	return indexes
}

func negated(tail string) bool {
	// Only the phrase immediately following a skill is examined.
	if i := strings.IndexAny(tail, ",;。.!?\n"); i >= 0 {
		tail = tail[:i]
	}
	trimmed := strings.TrimSpace(tail)
	return containsAny(trimmed, "경험 없음", "경험이 없", "미경험", "사용하지 않", "배우고 싶", "학습 예정", "학습예정", "배울 예정", "no experience", "not used") || strings.HasPrefix(trimmed, "없음")
}
