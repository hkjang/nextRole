package career

import (
	"strings"
	"testing"
)

func findSkill(p Profile, name string) (Skill, bool) {
	for _, s := range p.Skills {
		if s.Name == name {
			return s, true
		}
	}
	return Skill{}, false
}
func TestParseKoreanCareerAndConfidence(t *testing.T) {
	p := Parse("Java 백엔드 개발 경력 10년. Java 10년, Spring 8년, Docker 4년, Kubernetes 1년. 서울 금융 도메인 학사. 정보처리기사")
	if p.YearsExperience != 10 || p.Region != "서울" || p.Domain != "금융" || p.Education != "학사" {
		t.Fatalf("incorrect profile: %+v", p)
	}
	java, ok := findSkill(p, "Java")
	if !ok || java.Level != 5 || java.Years != 10 || java.Confidence != "explicit" {
		t.Fatalf("incorrect Java %+v", java)
	}
	docker, _ := findSkill(p, "Docker")
	if docker.Level != 3 {
		t.Error("incorrect Docker years mapping")
	}
	k8s, _ := findSkill(p, "Kubernetes")
	if k8s.Level != 2 {
		t.Error("incorrect Kubernetes years mapping")
	}
	inferred, ok := findSkill(p, "Backend API")
	if !ok || inferred.Confidence != "review" {
		t.Fatal("inferred skill must require review")
	}
	if len(p.Certifications) != 1 {
		t.Fatal("missing certification")
	}
}
func TestParserAvoidsSubstringAndNegation(t *testing.T) {
	p := Parse("JavaScript 고급, React 중급. Python 경험 없음, Kubernetes 학습 예정. Django 사용.")
	if _, ok := findSkill(p, "Java"); ok {
		t.Fatal("JavaScript falsely matched Java")
	}
	if _, ok := findSkill(p, "Go"); ok {
		t.Fatal("Django falsely matched Go")
	}
	if _, ok := findSkill(p, "Python"); ok {
		t.Fatal("negated Python treated as possessed")
	}
	if _, ok := findSkill(p, "Kubernetes"); ok {
		t.Fatal("planned Kubernetes treated as possessed")
	}
	js, ok := findSkill(p, "JavaScript")
	if !ok || js.Level != 4 {
		t.Fatal("explicit level not found")
	}
	react, ok := findSkill(p, "React")
	if !ok || react.Level != 3 {
		t.Fatal("react not found")
	}
}
func TestParserEmptyAndBounded(t *testing.T) {
	p := Parse("")
	if len(p.Skills) != 0 || p.YearsExperience != 0 {
		t.Fatal("empty input fabricated evidence")
	}
	if p.Skills == nil || p.Certifications == nil || p.Preferences == nil {
		t.Fatal("nil JSON arrays")
	}
	p = Parse(strings.Repeat("가", 500000))
	if len(p.Narrative) > 1<<20 {
		t.Fatal("narrative bound ignored")
	}
}
