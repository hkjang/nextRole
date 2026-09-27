package career

import (
	"encoding/json"
	"math"
	"math/rand"
	"reflect"
	"strings"
	"testing"
)

func testProfile() Profile {
	return Profile{CurrentRole: "백엔드 개발자", YearsExperience: 10, Education: "학사", Domain: "IT", Region: "서울", WeeklyHours: 8, Skills: []Skill{{Name: "java", Level: 5, Confidence: "explicit"}, {Name: "Spring Boot", Level: 4, Confidence: "explicit"}, {Name: "docker", Level: 3, Confidence: "explicit"}, {Name: "Linux", Level: 4, Confidence: "explicit"}, {Name: "Backend API", Level: 5, Confidence: "explicit"}, {Name: "ci/cd", Level: 4, Confidence: "explicit"}}, Certifications: []string{}, Preferences: []string{}}
}

func TestEmptyProfileHasZeroScore(t *testing.T) {
	for _, j := range SeedJobs() {
		s := Simulate(Profile{}, j, 6, nil)
		if s.Score != 0 {
			t.Errorf("%s: empty profile score %v", j.ID, s.Score)
		}
		if s.Difficulty != 100 {
			t.Errorf("empty profile difficulty %v", s.Difficulty)
		}
		if !strings.Contains(strings.Join(s.Warnings, " "), "경력 근거가 비어") {
			t.Error("missing empty profile warning")
		}
	}
}

func TestScoresMonotonicBoundedAndDeterministic(t *testing.T) {
	rng := rand.New(rand.NewSource(721))
	for _, j := range SeedJobs() {
		for trial := 0; trial < 8; trial++ {
			p := testProfile()
			for _, r := range j.Skills {
				p.Skills = append(p.Skills, Skill{Name: r.Name, Level: float64(rng.Intn(6)), Confidence: "explicit"})
			}
			baseline := Simulate(p, j, 6, nil)
			if baseline.Score < 0 || baseline.Score > 100 || math.IsNaN(baseline.Score) {
				t.Fatalf("invalid baseline %v", baseline.Score)
			}
			if !reflect.DeepEqual(baseline, Simulate(p, j, 6, nil)) {
				t.Fatalf("nondeterministic result %s", j.ID)
			}
			for _, r := range j.Skills {
				added := []Skill{{Name: r.Name, Level: 5}}
				after := Simulate(p, j, 6, added)
				if after.Score < baseline.Score {
					t.Fatalf("%s adding %s decreased score %v -> %v", j.ID, r.Name, baseline.Score, after.Score)
				}
				if after.BaselineScore != baseline.Score {
					t.Fatal("incorrect baseline score")
				}
				if after.EstimatedMonths > baseline.EstimatedMonths {
					t.Fatal("adding skill increased direct study duration")
				}
			}
			sum := 0.0
			for _, f := range baseline.Factors {
				sum += f.Score
				if f.Score < 0 || f.Score > f.Max+.1 {
					t.Fatal("invalid factor")
				}
			}
			if math.Abs(sum-baseline.Score) > .3 {
				t.Fatalf("factor sum %.1f differs from score %.1f", sum, baseline.Score)
			}
			for _, r := range baseline.ROI {
				if r.Gain < 0 || r.After < r.Before || math.Abs((r.After-r.Before)-r.Gain) > .001 {
					t.Fatal("invalid ROI")
				}
			}
		}
	}
}

func TestReviewSkillsAreExcludedAndCanBeConfirmed(t *testing.T) {
	j := SeedJobs()[0]
	p := Profile{Skills: []Skill{{Name: "Python", Level: 5, Confidence: "review"}, {Name: "Kubernetes", Level: 5, Confidence: "inferred"}}}
	s := Simulate(p, j, 6, nil)
	if s.Score != 0 {
		t.Fatalf("review skills inflated score %v", s.Score)
	}
	for _, g := range s.Gaps {
		if g.Current != 0 {
			t.Error("review skill included in gap")
		}
	}
	p.Skills[0].Confidence = "explicit"
	after := Simulate(p, j, 6, nil)
	if after.Score <= s.Score {
		t.Fatal("confirmed skill has no effect")
	}
}

func TestAliasNormalizationAndDuplicateSkills(t *testing.T) {
	p := testProfile()
	j := SeedJobs()[0]
	before := Simulate(p, j, 6, nil)
	p.Skills = append(p.Skills, Skill{Name: "도커", Level: 3}, Skill{Name: "DOCKER", Level: 2})
	after := Simulate(p, j, 6, nil)
	if before.Score != after.Score {
		t.Fatal("duplicate aliases inflate score")
	}
	if NormalizeSkill("spring_boot") != "Spring" {
		t.Error("spring alias")
	}
	if NormalizeSkill("쿠버네티스") != "Kubernetes" {
		t.Error("kubernetes alias")
	}
	if SkillSimilarity("Python", "python") != 1 {
		t.Fatal("exact similarity")
	}
	if SkillSimilarity("Python", "Java") <= SkillSimilarity("Python", "회계") {
		t.Fatal("semantic vector features are not useful")
	}
	if SkillSimilarity("Custom Unrelated Skill", "Java") != 0 {
		t.Fatal("unknown skills receive fuzzy credit")
	}
}

func TestSkillVectorTransferDoesNotEliminateGap(t *testing.T) {
	p := Profile{Skills: []Skill{{Name: "Java", Level: 5, Confidence: "explicit"}}}
	j := Job{Skills: []Requirement{{Name: "Python", Level: 4, Weight: 1}}}
	s := Simulate(p, j, 6, nil)
	if s.Gaps[0].Gap != 4 {
		t.Error("semantic similarity incorrectly filled exact gap")
	}
	if s.Factors[0].Score != 0 || s.Factors[1].Score <= 0 || s.Factors[1].Score > 12 {
		t.Error("incorrect bounded transfer credit")
	}
}

func TestHorizonChangesPlanNotScore(t *testing.T) {
	p := testProfile()
	j := SeedJobs()[0]
	short := Simulate(p, j, 3, nil)
	long := Simulate(p, j, 12, nil)
	if short.Score != long.Score {
		t.Fatal("time alone inflated score")
	}
	if short.Plan[len(short.Plan)-1].Month != 3 || long.Plan[len(long.Plan)-1].Month != 12 {
		t.Fatal("plan horizon mismatch")
	}
	p.WeeklyHours = 16
	faster := Simulate(p, j, 6, nil)
	if faster.EstimatedMonths > long.EstimatedMonths {
		t.Fatal("more study hours made path slower")
	}
}

func TestCustomAndInvalidWeights(t *testing.T) {
	j := SeedJobs()[0]
	p := testProfile()
	onlySkills := SimulateWithWeights(p, j, 6, nil, Weights{Skill: 100})
	for _, f := range onlySkills.Factors[1:] {
		if f.Score != 0 || f.Max != 0 {
			t.Fatal("disabled weight still contributes")
		}
	}
	if onlySkills.Factors[0].Max != 100 {
		t.Fatal("custom weight ignored")
	}
	defaultSim := Simulate(p, j, 6, nil)
	for _, weights := range []Weights{{}, {Skill: -1}, {Skill: math.NaN()}, {Skill: math.Inf(1)}} {
		if got := SimulateWithWeights(p, j, 6, nil, weights); got.Score != defaultSim.Score {
			t.Fatal("invalid weights did not use safe defaults")
		}
	}
}

func TestInvalidInputsRemainFinite(t *testing.T) {
	p := Profile{YearsExperience: math.NaN(), WeeklyHours: math.Inf(1), Skills: []Skill{{Name: "Python", Level: math.Inf(1)}}}
	j := Job{Skills: []Requirement{{Name: "Python", Level: math.NaN(), Weight: math.Inf(1)}, {Name: "", Level: 5}, {Name: "SQL", Level: 4, Weight: -1}}}
	s := Simulate(p, j, 99, nil)
	if math.IsNaN(s.Score) || math.IsInf(s.Score, 0) {
		t.Fatal("non-finite score")
	}
	// Job metadata is untrusted input and retained as supplied; numeric engine
	// outputs must remain valid even if validation is bypassed by an internal caller.
	s.Job = Job{}
	if _, err := json.Marshal(s); err != nil {
		t.Fatalf("engine returned invalid JSON: %v", err)
	}
	noModel := Simulate(testProfile(), Job{}, 6, nil)
	if noModel.Score != 0 {
		t.Error("job with no requirements is scored")
	}
}

func TestCatalogAndSyntheticProvenance(t *testing.T) {
	catalog := SeedJobs()
	if len(catalog) < 20 {
		t.Fatal("catalog too narrow")
	}
	ids := map[string]bool{}
	for _, j := range catalog {
		if ids[j.ID] {
			t.Fatal("duplicate ID")
		}
		ids[j.ID] = true
	}
	for _, j := range catalog {
		if !j.Source.Synthetic || len(j.Skills) == 0 {
			t.Fatalf("bad catalog job %s", j.ID)
		}
		for _, id := range j.BridgeIDs {
			if !ids[id] {
				t.Fatalf("unknown bridge %s", id)
			}
		}
		jobs, training := SeedOpportunities(j)
		for _, o := range append(jobs, training...) {
			if !o.Source.Synthetic || o.URL != "" || !strings.Contains(o.Title, "합성") {
				t.Fatal("demo represented as live data")
			}
		}
		s := Simulate(testProfile(), j, 6, nil)
		if len(s.Paths) < 2 {
			t.Fatalf("missing bridge paths %s", j.ID)
		}
		if s.Paths[0].Months != s.EstimatedMonths {
			t.Fatal("inconsistent direct duration")
		}
	}
	result := Recommend(testProfile(), catalog)
	if len(result) != 5 {
		t.Fatal("expected top five")
	}
	for i := 1; i < len(result); i++ {
		if result[i].Score > result[i-1].Score {
			t.Fatal("unsorted recommendations")
		}
	}
}

func TestInputsNotMutated(t *testing.T) {
	p := testProfile()
	before, _ := json.Marshal(p)
	j := SeedJobs()[0]
	Simulate(p, j, 6, []Skill{{Name: "Python", Level: 5}})
	after, _ := json.Marshal(p)
	if string(before) != string(after) {
		t.Fatal("simulation mutated input profile")
	}
}

func TestImportedBridgeCatalog(t *testing.T) {
	bridge := Job{ID: "import:bridge", Title: "기관 승인 중간 직무", Skills: []Requirement{{Name: "SQL", Level: 3, Weight: 1}}, Source: Source{Name: "사내 직무 사전"}}
	target := Job{ID: "import:target", Title: "기관 승인 목표 직무", Skills: []Requirement{{Name: "Python", Level: 4, Weight: 1}, {Name: "SQL", Level: 4, Weight: 1}}, BridgeIDs: []string{"import:bridge"}, Source: Source{Name: "사내 직무 사전"}}
	sim := SimulateWithCatalog(testProfile(), target, 6, nil, DefaultWeights, []Job{target, bridge})
	if len(sim.Paths) != 2 || sim.Paths[1].Roles[1] != bridge.Title {
		t.Fatalf("imported bridge lost: %+v", sim.Paths)
	}
	if strings.Contains(sim.Paths[1].Reason, "합성") {
		t.Fatal("imported real bridge mislabeled synthetic")
	}
}

func TestCareerBreakAddsReentryTasksWithoutScorePenalty(t *testing.T) {
	profile := Profile{YearsExperience: 5, Skills: []Skill{{Name: "Java", Level: 4, Confidence: "explicit"}}, WeeklyHours: 10}
	job := SeedJobs()[0]
	baseline := Simulate(profile, job, 6, nil)
	profile.CareerBreakMonths = 12
	reentry := Simulate(profile, job, 6, nil)
	if baseline.Score != reentry.Score {
		t.Fatal("career break alone must not reduce score")
	}
	if len(reentry.Warnings) <= len(baseline.Warnings) || len(reentry.Plan[0].Tasks) <= len(baseline.Plan[0].Tasks) {
		t.Fatal("missing actionable reentry support")
	}
}

func TestDerivedMappingDoesNotAssumeMissingExperienceOrEducation(t *testing.T) {
	job := Job{ID: "reviewed", Title: "검토된 직무", Source: Source{Kind: "derived"}, Skills: []Requirement{{Name: "SQL", Level: 3, Weight: 1}}, SkillAliases: map[string]string{"질의 언어": "SQL"}}
	p := Profile{YearsExperience: 10, Education: "학사", Skills: []Skill{{Name: "질의 언어", Level: 3, Confidence: "explicit"}}}
	sim := Simulate(p, job, 6, nil)
	if sim.Factors[2].Score != 0 || sim.Factors[4].Score != 0 || sim.Gaps[0].Current != 3 {
		t.Fatalf("reviewed model assumed unavailable requirements or lost aliases: %+v", sim)
	}
	if !strings.Contains(sim.Factors[2].Reason, "기준이 없어") || !strings.Contains(strings.Join(sim.Warnings, " "), "공식 수준이 아니며") {
		t.Fatal("unknown requirements or internal scale were not explained")
	}
}
