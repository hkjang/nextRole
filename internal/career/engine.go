package career

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

func bounded(v, lo, hi float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return lo
	}
	return math.Max(lo, math.Min(hi, v))
}
func round(v float64) float64 { return math.Round(v*10) / 10 }

func normalizeWeights(w Weights) Weights {
	vals := []float64{w.Skill, w.Transfer, w.Experience, w.Domain, w.Education, w.Preference}
	total := 0.0
	for _, v := range vals {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
			return DefaultWeights
		}
		total += v
	}
	if total <= 0 || math.IsInf(total, 0) {
		return DefaultWeights
	}
	return Weights{100 * w.Skill / total, 100 * w.Transfer / total, 100 * w.Experience / total, 100 * w.Domain / total, 100 * w.Education / total, 100 * w.Preference / total}
}

func requirements(job Job) []Requirement {
	result := []Requirement{}
	indices := map[string]int{}
	for _, r := range job.Skills {
		r.Name = NormalizeSkill(r.Name)
		r.Level = bounded(r.Level, 0, 5)
		r.Weight = bounded(r.Weight, 0, 100)
		if r.Name == "" || r.Level == 0 {
			continue
		}
		if r.Weight == 0 {
			r.Weight = 1
		}
		if i, ok := indices[r.Name]; ok {
			result[i].Level = math.Max(result[i].Level, r.Level)
			result[i].Weight = math.Max(result[i].Weight, r.Weight)
			continue
		}
		indices[r.Name] = len(result)
		result = append(result, r)
	}
	return result
}

func profileHasEvidence(p Profile) bool {
	if p.YearsExperience > 0 || strings.TrimSpace(p.CurrentRole) != "" || strings.TrimSpace(p.Education) != "" || strings.TrimSpace(p.Domain) != "" || len(p.Preferences) > 0 || p.Region != "" {
		return true
	}
	for _, l := range verifiedSkills(p) {
		if l > 0 {
			return true
		}
	}
	return false
}

func scoreProfile(p Profile, job Job, w Weights) (float64, []Factor, []Gap, []string) {
	w = normalizeWeights(w)
	skills := verifiedSkills(p)
	reqs := requirements(job)
	gaps := []Gap{}
	strengths := []string{}
	weightTotal, skillTotal, transferTotal := 0.0, 0.0, 0.0
	names := sortedSkillNames(skills)
	for _, r := range reqs {
		current := skills[r.Name]
		coverage := math.Min(current/r.Level, 1)
		best := coverage
		for _, n := range names {
			if n == r.Name {
				continue
			}
			// Partial semantic transfer is conservatively capped: shared concepts are
			// preparation evidence, not proof that a different skill is mastered.
			similarity := SkillSimilarity(n, r.Name) * .6
			best = math.Max(best, similarity*math.Min(skills[n]/r.Level, 1))
		}
		weightTotal += r.Weight
		skillTotal += coverage * r.Weight
		transferTotal += best * r.Weight
		gaps = append(gaps, Gap{r.Name, r.Level, current, math.Max(r.Level-current, 0), r.Weight})
		if current >= r.Level {
			strengths = append(strengths, r.Name)
		}
	}
	skillRatio, transferRatio := 0.0, 0.0
	if weightTotal > 0 {
		skillRatio = skillTotal / weightTotal
		transferRatio = transferTotal / weightTotal
	}
	experience := 0.0
	years := bounded(p.YearsExperience, 0, 80)
	if years > 0 {
		experience = 1
		if job.MinExperience > 0 {
			experience = bounded(years/job.MinExperience, 0, 1)
		}
	}
	domain := domainMatch(p.Domain, job.Domain)
	education := educationMatch(p.Education, job.Education, profileHasEvidence(p))
	preference := preferenceMatch(p, job)
	if len(reqs) == 0 {
		skillRatio = 0
		transferRatio = 0
		experience = 0
		domain = 0
		education = 0
		preference = 0
	}
	factors := []Factor{
		{Name: "기술역량", Score: round(skillRatio * w.Skill), Max: round(w.Skill), Reason: fmt.Sprintf("확인된 보유 수준 ÷ 요구 수준을 중요도로 가중 평균: %.1f%%. 추론 역량은 제외합니다.", skillRatio*100)},
		{Name: "전이 가능 역량", Score: round(transferRatio * w.Transfer), Max: round(w.Transfer), Reason: fmt.Sprintf("명시적 역량과 로컬 의미 특징 벡터의 코사인 유사도: %.1f%%. 다른 기술의 전이 기여는 유사도의 60%%로 제한합니다.", transferRatio*100)},
		{Name: "경력", Score: round(experience * w.Experience), Max: round(w.Experience), Reason: fmt.Sprintf("입력 경력 %.1f년 / 직무 참고 경력 %.1f년. 관련성은 기술·산업 요소에서 별도 반영합니다.", years, bounded(job.MinExperience, 0, 80))},
		{Name: "산업경험", Score: round(domain * w.Domain), Max: round(w.Domain), Reason: fmt.Sprintf("입력 산업 ‘%s’과 직무 산업 ‘%s’의 일치도 %.0f%%. 미입력은 0점입니다.", display(p.Domain), display(job.Domain), domain*100)},
		{Name: "교육·자격", Score: round(education * w.Education), Max: round(w.Education), Reason: fmt.Sprintf("입력 학력 ‘%s’ / 참고 학력 ‘%s’. 자격증은 프로필에 보관하며 직무별 검증 매핑 전에는 점수를 추가하지 않습니다.", display(p.Education), display(job.Education))},
		{Name: "개인 선호", Score: round(preference * w.Preference), Max: round(w.Preference), Reason: fmt.Sprintf("명시한 선호 직무·분류와 희망지역의 일치도 %.0f%%. 지역만 입력하면 이 요소의 최대 50%%입니다.", preference*100)},
	}
	score := 0.0
	for _, f := range factors {
		score += f.Score
	}
	return bounded(round(score), 0, 100), factors, gaps, strengths
}

func display(s string) string {
	if strings.TrimSpace(s) == "" {
		return "미입력"
	}
	return s
}
func domainMatch(a, b string) float64 {
	a, b = key(a), key(b)
	if a == "" || b == "" {
		return 0
	}
	if a == b {
		return 1
	}
	groups := [][]string{{"it", "정보통신", "소프트웨어", "인터넷"}, {"비즈니스", "서비스", "마케팅", "인사", "금융"}, {"제조", "물류", "유통"}, {"분석", "데이터", "it"}}
	for _, g := range groups {
		hasA, hasB := false, false
		for _, x := range g {
			if a == x {
				hasA = true
			}
			if b == x {
				hasB = true
			}
		}
		if hasA && hasB {
			return .5
		}
	}
	return 0
}
func educationLevel(s string) int {
	switch {
	case containsAny(strings.ToLower(s), "박사", "phd", "doctor"):
		return 5
	case containsAny(strings.ToLower(s), "석사", "master"):
		return 4
	case containsAny(strings.ToLower(s), "전문학사", "전문대", "associate"):
		return 2
	case containsAny(strings.ToLower(s), "학사", "대졸", "대학교", "bachelor"):
		return 3
	case containsAny(strings.ToLower(s), "고졸", "고등학교", "high school"):
		return 1
	}
	return 0
}
func educationMatch(got, want string, evidence bool) float64 {
	if strings.TrimSpace(want) == "" || want == "무관" {
		if evidence {
			return 1
		}
		return 0
	}
	g, w := educationLevel(got), educationLevel(want)
	if w == 0 {
		if key(got) == key(want) {
			return 1
		}
		return 0
	}
	return math.Min(float64(g)/float64(w), 1)
}
func preferenceMatch(p Profile, j Job) float64 {
	score := 0.0
	for _, pref := range p.Preferences {
		if pref == j.ID || strings.EqualFold(pref, j.Title) || strings.EqualFold(pref, j.Category) {
			score = .5
			break
		}
	}
	if p.Region != "" {
		for _, r := range j.Regions {
			if r == "전국" || r == p.Region {
				score += .5
				break
			}
		}
	}
	return score
}

func Simulate(profile Profile, job Job, months int, addedSkills []Skill) Simulation {
	return SimulateWithWeights(profile, job, months, addedSkills, DefaultWeights)
}

func SimulateWithWeights(profile Profile, job Job, months int, addedSkills []Skill, w Weights) Simulation {
	return SimulateWithCatalog(profile, job, months, addedSkills, w, SeedJobs())
}

// SimulateWithCatalog resolves intermediate roles against the caller's combined
// catalog so imported occupations preserve their own bridge relationships.
func SimulateWithCatalog(profile Profile, job Job, months int, addedSkills []Skill, w Weights, jobs []Job) Simulation {
	if months != 3 && months != 6 && months != 12 {
		months = 6
	}
	baseline, _, _, _ := scoreProfile(profile, job, w)
	hypothetical := mergeSkills(profile, addedSkills)
	score, factors, gaps, strengths := scoreProfile(hypothetical, job, w)
	sim := Simulation{Job: job, Score: score, BaselineScore: baseline, Difficulty: round(100 - score), Gaps: gaps, Factors: factors, Strengths: strengths, ROI: []ROI{}, Paths: []Path{}, Plan: []Plan{}, Warnings: []string{}, Source: job.Source}
	for _, g := range gaps {
		if g.Gap <= 0 {
			continue
		}
		after, _, _, _ := scoreProfile(mergeSkills(hypothetical, []Skill{{Name: g.Name, Level: g.Required}}), job, w)
		sim.ROI = append(sim.ROI, ROI{g.Name, score, after, round(math.Max(after-score, 0))})
	}
	sort.SliceStable(sim.ROI, func(i, j int) bool {
		if sim.ROI[i].Gain == sim.ROI[j].Gain {
			return sim.ROI[i].Name < sim.ROI[j].Name
		}
		return sim.ROI[i].Gain > sim.ROI[j].Gain
	})
	weekly := bounded(profile.WeeklyHours, 0, 80)
	if weekly == 0 {
		weekly = 6
		sim.Warnings = append(sim.Warnings, "주당 학습시간이 없어 주 6시간을 가정했습니다.")
	}
	hours := gapHours(gaps)
	sim.EstimatedMonths = monthsForHours(hours, weekly)
	sim.Paths = buildPaths(hypothetical, job, gaps, weekly, w, jobs)
	sim.Plan = buildPlan(gaps, sim.ROI, months)
	if profile.CareerBreakMonths > 0 {
		sim.Warnings = append(sim.Warnings, fmt.Sprintf("경력 공백 %d개월을 입력했습니다. 공백 자체로 적합도를 감점하지 않습니다. 최근 실무 도구와 업무 방식의 변화를 확인하고 복귀 프로젝트로 현재 역량을 점검하세요.", profile.CareerBreakMonths))
		if len(sim.Plan) > 0 {
			sim.Plan[0].Tasks = append([]string{"재진입 점검: 목표 직무의 최근 실제 공고 3개에서 도구·업무 변화 확인", "보유 역량을 확인할 작은 복귀 프로젝트를 수행하고 포트폴리오 갱신"}, sim.Plan[0].Tasks...)
		}
	}
	if !profileHasEvidence(profile) {
		sim.Warnings = append(sim.Warnings, "경력 근거가 비어 있습니다. 프로필을 입력하고 추출 결과를 확인한 뒤 결과를 비교하세요.")
	}
	for _, s := range profile.Skills {
		if s.Confidence != "" && s.Confidence != "explicit" {
			sim.Warnings = append(sim.Warnings, "추론·검증필요 역량은 적합도 계산에서 제외했습니다. 내 경력에서 확인한 뒤 명시 역량으로 변경하세요.")
			break
		}
	}
	if len(requirements(job)) == 0 {
		sim.Warnings = append(sim.Warnings, "목표 직무의 유효한 요구역량이 없어 점수를 산정할 수 없습니다.")
	}
	if len(addedSkills) > 0 {
		sim.Warnings = append(sim.Warnings, "추가 역량은 습득을 가정한 시나리오입니다. 실제 프로필이나 검증 상태를 변경하지 않습니다.")
	}
	if float64(months)*weekly*4.345 < hours {
		sim.Warnings = append(sim.Warnings, fmt.Sprintf("선택한 %d개월 동안의 학습 가능 시간(약 %.0f시간)이 참고 학습량(약 %.0f시간)보다 적습니다.", months, float64(months)*weekly*4.345, hours))
	}
	if job.Source.Synthetic {
		sim.Warnings = append(sim.Warnings, "내장 직무 요건은 합성 예시이며 고용24·NCS 실측 데이터가 아닙니다. 관리자가 연동하면 출처가 있는 직무로 계산할 수 있습니다.")
	}
	sim.Warnings = append(sim.Warnings, "적합도와 난이도는 설명 가능한 규칙 점수이며 취업 확률이 아닙니다. 기간·비용은 수준별 참고 학습시간과 20시간당 5만 원의 가정으로 계산한 합성 추정치입니다.")
	sim.Explanation = fmt.Sprintf("%s의 확인된 역량을 %s의 요구 수준과 비교했습니다. 적합도 %.1f점, 역량 부족 항목 %d개이며, 주 %.0f시간 기준 참고 학습기간은 %d개월입니다. 기간을 바꿔도 학습을 완료했다고 가정하지 않으므로 점수는 자동 상승하지 않습니다.", display(profile.CurrentRole), job.Title, score, len(sim.ROI), weekly, sim.EstimatedMonths)
	if len(sim.ROI) > 0 {
		sim.Explanation += fmt.Sprintf(" %s를 요구 수준까지 확보하면 동일 조건에서 %.1f점 상승하는 것으로 계산됩니다.", sim.ROI[0].Name, sim.ROI[0].Gain)
	}
	return sim
}

func gapHours(gaps []Gap) float64 {
	hours := 0.0
	for _, g := range gaps {
		hours += g.Gap * learningHours(g.Name)
	}
	return hours
}
func monthsForHours(hours, weekly float64) int {
	if hours <= 0 {
		return 0
	}
	return int(math.Ceil(hours / (weekly * 4.345)))
}
func costForHours(hours float64) float64 { return math.Ceil(hours/20) * 50000 }

func buildPaths(p Profile, j Job, gaps []Gap, weekly float64, w Weights, jobs []Job) []Path {
	names := []string{}
	reuse := 0.0
	count := 0.0
	for _, g := range gaps {
		if g.Gap > 0 {
			names = append(names, g.Name)
		}
		if g.Required > 0 {
			reuse += math.Min(g.Current/g.Required, 1)
			count++
		}
	}
	if count > 0 {
		reuse = round(100 * reuse / count)
	}
	start := p.CurrentRole
	if strings.TrimSpace(start) == "" {
		start = "현재 경력"
	}
	hours := gapHours(gaps)
	paths := []Path{{ID: "direct", Title: "목표 직무로 직접 전환", Roles: []string{start, j.Title}, Months: monthsForHours(hours, weekly), Skills: names, Reuse: reuse, Cost: costForHours(hours), Reason: "목표 직무의 부족 수준을 모두 학습하는 경로입니다. 기간은 수준별 학습시간 / 주당 학습시간, 비용은 20시간당 5만 원 가정입니다."}}
	catalog := map[string]Job{}
	for _, candidate := range jobs {
		catalog[candidate.ID] = candidate
	}
	for _, id := range j.BridgeIDs {
		bridge, ok := catalog[id]
		if !ok || id == j.ID {
			continue
		}
		_, _, bridgeGaps, _ := scoreProfile(p, bridge, w)
		additions := []Skill{}
		pathSkills := []string{}
		seen := map[string]bool{}
		for _, g := range bridgeGaps {
			if g.Gap > 0 {
				additions = append(additions, Skill{Name: g.Name, Level: g.Required})
				pathSkills = append(pathSkills, g.Name)
				seen[g.Name] = true
			}
		}
		_, _, nextGaps, _ := scoreProfile(mergeSkills(p, additions), j, w)
		for _, g := range nextGaps {
			if g.Gap > 0 && !seen[g.Name] {
				pathSkills = append(pathSkills, g.Name)
				seen[g.Name] = true
			}
		}
		bridgeHours := gapHours(bridgeGaps) + gapHours(nextGaps)
		bridgeReuse := 0.0
		for _, g := range bridgeGaps {
			bridgeReuse += math.Min(g.Current/g.Required, 1)
		}
		if len(bridgeGaps) > 0 {
			bridgeReuse = round(100 * bridgeReuse / float64(len(bridgeGaps)))
		}
		bridgeSource := " 중간 직무 출처: " + bridge.Source.Name + "."
		if bridge.Source.Synthetic {
			bridgeSource += " 합성 예시입니다."
		}
		paths = append(paths, Path{ID: "via-" + id, Title: bridge.Title + " 경유", Roles: []string{start, bridge.Title, j.Title}, Months: monthsForHours(bridgeHours, weekly), Skills: pathSkills, Reuse: bridgeReuse, Cost: costForHours(bridgeHours), Reason: "중간 직무 요구역량을 먼저 확보한 뒤 남은 목표 역량을 학습합니다. 중복 학습은 제외하며 실제 채용·재직 기간은 포함하지 않습니다." + bridgeSource})
		if len(paths) >= 4 {
			break
		}
	}
	return paths
}

func buildPlan(gaps []Gap, roi []ROI, months int) []Plan {
	plans := []Plan{}
	if len(roi) == 0 {
		return []Plan{{Month: months, Title: "실무 근거와 채용 조건 점검", Tasks: []string{"보유 역량을 보여 주는 프로젝트와 성과 자료 정리", "실제 채용 요건·자격 조건과 희망지역 확인", "학습 근거와 경력 프로필 최신화"}, Skills: []string{}}}
	}
	// Prioritize score gain per estimated study hour; retain raw score gains in ROI.
	ordered := append([]ROI(nil), roi...)
	lookup := map[string]Gap{}
	for _, g := range gaps {
		lookup[g.Name] = g
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		a, b := lookup[ordered[i].Name], lookup[ordered[j].Name]
		return ordered[i].Gain/math.Max(a.Gap*learningHours(a.Name), 1) > ordered[j].Gain/math.Max(b.Gap*learningHours(b.Name), 1)
	})
	stages := 3
	for stage := 0; stage < stages; stage++ {
		stageSkills := []string{}
		for i, r := range ordered {
			if i*stages/len(ordered) == stage {
				stageSkills = append(stageSkills, r.Name)
			}
		}
		tasks := []string{}
		if len(stageSkills) > 0 {
			tasks = append(tasks, strings.Join(stageSkills, ", ")+" 학습·실습 후 결과물로 수준 확인")
		}
		titles := []string{"핵심 역량과 기초 실습", "직무 과제와 통합 프로젝트", "성과 검증과 전환 준비"}
		switch stage {
		case 0:
			tasks = append(tasks, "학습시간을 확보하고 현재 수준을 진단", "출처가 확인된 교육과정의 비용·일정 비교")
		case 1:
			tasks = append(tasks, "목표 직무의 실무 문제를 재현하는 프로젝트 수행", "동료 피드백을 받고 부족 부분 보완")
		case 2:
			tasks = append(tasks, "완료한 역량을 내 경력에 반영하고 재시뮬레이션", "프로젝트 포트폴리오·이력서 정리 및 실제 공고 요건 확인")
		}
		plans = append(plans, Plan{Month: int(math.Ceil(float64((stage+1)*months) / 3)), Title: titles[stage], Tasks: tasks, Skills: stageSkills})
	}
	return plans
}

func Recommend(profile Profile, jobs []Job) []Simulation {
	return RecommendWithWeights(profile, jobs, DefaultWeights)
}
func RecommendWithWeights(profile Profile, jobs []Job, w Weights) []Simulation {
	results := make([]Simulation, 0, len(jobs))
	for _, job := range jobs {
		results = append(results, SimulateWithCatalog(profile, job, 6, nil, w, jobs))
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score == results[j].Score {
			if results[i].EstimatedMonths == results[j].EstimatedMonths {
				return results[i].Job.ID < results[j].Job.ID
			}
			return results[i].EstimatedMonths < results[j].EstimatedMonths
		}
		return results[i].Score > results[j].Score
	})
	if len(results) > 5 {
		results = results[:5]
	}
	return results
}
