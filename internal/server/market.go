package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/hkjang/nextRole/internal/career"
	"github.com/jackc/pgx/v5"
)

type MarketCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type MarketTrend struct {
	Name     string `json:"name"`
	Current  int    `json:"current"`
	Previous int    `json:"previous"`
	Change   int    `json:"change"`
}

type MarketSalary struct {
	Title  string        `json:"title"`
	Salary string        `json:"salary"`
	Source career.Source `json:"source"`
	URL    string        `json:"url"`
}

type MarketSummary struct {
	TotalJobs               int             `json:"totalJobs"`
	Regions                 []MarketCount   `json:"regions"`
	SkillFrequency          []MarketCount   `json:"skillFrequency"`
	Trends                  []MarketTrend   `json:"trends"`
	SalarySamples           []MarketSalary  `json:"salarySamples"`
	Sources                 []career.Source `json:"sources"`
	UpdatedAt               string          `json:"updatedAt"`
	HasBaseline             bool            `json:"hasBaseline"`
	BaselineDate            string          `json:"baselineDate,omitempty"`
	Warnings                []string        `json:"warnings"`
	ExcludedUnknownDeadline int             `json:"excludedUnknownDeadline"`
}

type marketRecord struct {
	Opportunity career.Opportunity
	UpdatedAt   time.Time
}

type marketAggregate struct {
	TotalJobs      int             `json:"totalJobs"`
	Regions        []MarketCount   `json:"regions"`
	SkillFrequency []MarketCount   `json:"skillFrequency"`
	Sources        []career.Source `json:"sources"`
	CriteriaHash   string          `json:"criteriaHash,omitempty"`
}

type marketSnapshot struct {
	Version    int                        `json:"version"`
	Date       string                     `json:"date"`
	CapturedAt time.Time                  `json:"capturedAt"`
	Overall    marketAggregate            `json:"overall"`
	ByJob      map[string]marketAggregate `json:"byJob"`
}

var marketLocation = time.FixedZone("Asia/Seoul", 9*60*60)

func (a *App) marketRecords(ctx context.Context) ([]marketRecord, error) {
	rows, err := a.db.Query(ctx, "SELECT data,updated_at FROM nr_records WHERE kind='jobs' AND owner_id='' ORDER BY updated_at DESC,id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []marketRecord{}
	for rows.Next() {
		var record marketRecord
		var data []byte
		if err := rows.Scan(&data, &record.UpdatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &record.Opportunity); err != nil {
			return nil, errors.New("저장된 채용 데이터 형식을 확인하세요")
		}
		result = append(result, record)
	}
	return result, rows.Err()
}

func (a *App) marketSummary(r *http.Request, job *career.Job) (MarketSummary, error) {
	now := time.Now()
	records, err := a.marketRecords(r.Context())
	if err != nil {
		return MarketSummary{}, err
	}
	summary := summarizeMarket(records, job, now)
	var data []byte
	err = a.db.QueryRow(r.Context(), "SELECT data FROM nr_records WHERE kind='market_snapshot' AND owner_id='' AND id<$1 ORDER BY id DESC LIMIT 1", now.In(marketLocation).Format("2006-01-02")).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		summary.Warnings = append(summary.Warnings, "이전 날짜의 수집 스냅샷이 없어 역량 증감 추세를 표시하지 않습니다.")
		return summary, nil
	}
	if err != nil {
		return MarketSummary{}, err
	}
	var prior marketSnapshot
	if json.Unmarshal(data, &prior) != nil || prior.Version != 1 {
		summary.Warnings = append(summary.Warnings, "비교할 수 있는 이전 스냅샷이 없습니다.")
		return summary, nil
	}
	applyMarketBaseline(&summary, prior, job)
	return summary, nil
}

// captureMarketSnapshot is called after a successful jobs import. Repeating a
// sync on the same Korean calendar day replaces that day's snapshot; a trend is
// only compared with a different date. It records counts, never invented jobs.
func (a *App) captureMarketSnapshot(ctx context.Context) error {
	now := time.Now()
	records, err := a.marketRecords(ctx)
	if err != nil {
		return err
	}
	jobs, err := a.jobs(ctx)
	if err != nil {
		return err
	}
	snapshot := marketSnapshot{Version: 1, Date: now.In(marketLocation).Format("2006-01-02"), CapturedAt: now.UTC(), Overall: marketCounts(summarizeMarket(records, nil, now), ""), ByJob: map[string]marketAggregate{}}
	for i := range jobs {
		job := &jobs[i]
		snapshot.ByJob[job.ID] = marketCounts(summarizeMarket(records, job, now), marketCriteria(job))
	}
	return a.putRecord(ctx, "market_snapshot", "", snapshot.Date, snapshot)
}

func marketCounts(summary MarketSummary, criteria string) marketAggregate {
	return marketAggregate{TotalJobs: summary.TotalJobs, Regions: summary.Regions, SkillFrequency: summary.SkillFrequency, Sources: summary.Sources, CriteriaHash: criteria}
}

func marketCriteria(job *career.Job) string {
	if job == nil {
		return ""
	}
	names := make([]string, 0, len(job.Skills))
	for _, sk := range job.Skills {
		names = append(names, strings.ToLower(career.NormalizeSkill(sk.Name)))
	}
	sort.Strings(names)
	return digest(strings.ToLower(strings.TrimSpace(job.Title)) + "\x00" + strings.Join(names, "\x00"))
}

func summarizeMarket(records []marketRecord, job *career.Job, now time.Time) MarketSummary {
	result := MarketSummary{Regions: []MarketCount{}, SkillFrequency: []MarketCount{}, Trends: []MarketTrend{}, SalarySamples: []MarketSalary{}, Sources: []career.Source{}, Warnings: []string{"수집한 실제 공고 표본의 집계이며 전체 채용시장 규모나 취업 확률이 아닙니다."}}
	regions, skills := map[string]int{}, map[string]int{}
	sources := map[string]career.Source{}
	seen := map[string]bool{}
	var updated time.Time
	for _, record := range records {
		o := record.Opportunity
		if o.Source.Synthetic || !marketMatches(o, job) {
			continue
		}
		active, known := marketDeadline(o.Deadline, now)
		if !known {
			result.ExcludedUnknownDeadline++
			continue
		}
		if !active {
			continue
		}
		// A listing imported through two connectors should not double the
		// observed demand. Canonical source URL is preferred over local ID.
		key := strings.TrimSpace(o.URL)
		if key == "" {
			key = o.ID
		}
		if key != "" && seen[key] {
			continue
		}
		seen[key] = true
		result.TotalJobs++
		if record.UpdatedAt.After(updated) {
			updated = record.UpdatedAt
		}
		for region := range uniqueMarketRegions(o.Region) {
			regions[region]++
		}
		for skill := range uniqueMarketSkills(o.Skills) {
			skills[skill]++
		}
		if len(result.SalarySamples) < 20 && strings.TrimSpace(o.Salary) != "" {
			result.SalarySamples = append(result.SalarySamples, MarketSalary{Title: o.Title, Salary: o.Salary, Source: o.Source, URL: o.URL})
		}
		sources[o.Source.Name+"\x00"+o.Source.URL] = o.Source
	}
	if !updated.IsZero() {
		result.UpdatedAt = updated.UTC().Format(time.RFC3339)
	}
	result.Regions, result.SkillFrequency = marketSortedCounts(regions), marketSortedCounts(skills)
	for _, source := range sources {
		result.Sources = append(result.Sources, source)
	}
	sort.Slice(result.Sources, func(i, j int) bool {
		if result.Sources[i].Name == result.Sources[j].Name {
			return result.Sources[i].URL < result.Sources[j].URL
		}
		return result.Sources[i].Name < result.Sources[j].Name
	})
	if result.ExcludedUnknownDeadline > 0 {
		result.Warnings = append(result.Warnings, "마감일이 없거나 해석할 수 없는 공고는 모집 여부를 확인할 수 없어 집계에서 제외했습니다.")
	}
	if len(result.SalarySamples) > 0 {
		result.Warnings = append(result.Warnings, "급여는 공고에 공개된 원문 표본입니다. 연봉·월급·시급과 통화가 다를 수 있으며 평균 또는 예상 연봉으로 환산하지 않습니다.")
	}
	return result
}

func marketMatches(o career.Opportunity, job *career.Job) bool {
	if job == nil {
		return true
	}
	if strings.EqualFold(strings.TrimSpace(o.Title), strings.TrimSpace(job.Title)) {
		return true
	}
	postingSkills := uniqueMarketSkills(o.Skills)
	required := map[string]bool{}
	for _, skill := range job.Skills {
		required[strings.ToLower(career.NormalizeSkill(skill.Name))] = true
	}
	overlaps := 0
	for skill := range postingSkills {
		if required[strings.ToLower(skill)] {
			overlaps++
		}
	}
	return overlaps >= 2
}

func uniqueMarketSkills(skills []string) map[string]bool {
	result := map[string]bool{}
	seen := map[string]bool{}
	for _, name := range skills {
		name = career.NormalizeSkill(name)
		key := strings.ToLower(name)
		if strings.TrimSpace(name) != "" && !seen[key] {
			result[name], seen[key] = true, true
		}
	}
	return result
}

func uniqueMarketRegions(region string) map[string]bool {
	result := map[string]bool{}
	for _, part := range strings.FieldsFunc(region, func(r rune) bool { return r == ',' || r == ';' || r == '|' || r == '，' }) {
		if part = strings.TrimSpace(part); part != "" {
			result[part] = true
		}
	}
	return result
}

func marketSortedCounts(counts map[string]int) []MarketCount {
	result := make([]MarketCount, 0, len(counts))
	for name, count := range counts {
		result = append(result, MarketCount{Name: name, Count: count})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count == result[j].Count {
			return result[i].Name < result[j].Name
		}
		return result[i].Count > result[j].Count
	})
	return result
}

func marketDeadline(deadline string, now time.Time) (active, known bool) {
	deadline = strings.TrimSpace(deadline)
	switch strings.ToLower(deadline) {
	case "채용시까지", "채용 시까지", "상시", "상시채용", "상시 채용", "open", "rolling":
		return true, true
	case "마감", "채용마감", "채용 마감", "종료", "closed":
		return false, true
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if date, err := time.Parse(layout, deadline); err == nil {
			return !now.After(date), true
		}
	}
	for _, layout := range []string{"2006-01-02", "20060102", "2006.01.02", "2006/01/02", "06-01-02", "06.01.02"} {
		if date, err := time.ParseInLocation(layout, deadline, marketLocation); err == nil {
			return now.Before(date.AddDate(0, 0, 1)), true
		}
	}
	return false, false
}

func applyMarketBaseline(summary *MarketSummary, snapshot marketSnapshot, job *career.Job) {
	prior := snapshot.Overall
	if job != nil {
		var ok bool
		prior, ok = snapshot.ByJob[job.ID]
		if !ok || prior.CriteriaHash != marketCriteria(job) {
			summary.Warnings = append(summary.Warnings, "동일한 목표 직무·역량 기준의 이전 스냅샷이 없어 추세를 표시하지 않습니다.")
			return
		}
	}
	before, current, names := map[string]int{}, map[string]int{}, map[string]bool{}
	for _, skill := range prior.SkillFrequency {
		before[skill.Name], names[skill.Name] = skill.Count, true
	}
	for _, skill := range summary.SkillFrequency {
		current[skill.Name], names[skill.Name] = skill.Count, true
	}
	summary.HasBaseline, summary.BaselineDate = true, snapshot.Date
	for name := range names {
		summary.Trends = append(summary.Trends, MarketTrend{Name: name, Current: current[name], Previous: before[name], Change: current[name] - before[name]})
	}
	sort.Slice(summary.Trends, func(i, j int) bool {
		if summary.Trends[i].Change == summary.Trends[j].Change {
			return summary.Trends[i].Name < summary.Trends[j].Name
		}
		return summary.Trends[i].Change > summary.Trends[j].Change
	})
	summary.Warnings = append(summary.Warnings, "추세는 이전 수집일 대비 공고 수 차이입니다. 수집 범위 변경·공고 만료의 영향이 포함되며 시장 전체 성장률이 아닙니다.")
}
