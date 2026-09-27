package server

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/hkjang/nextRole/internal/career"
)

func marketFixture(id, title, region, deadline string, skills []string, synthetic bool) marketRecord {
	return marketRecord{Opportunity: career.Opportunity{ID: id, Title: title, Region: region, Deadline: deadline, Skills: skills, URL: "https://jobs.example.test/" + id, Source: career.Source{Name: "검증 공고", URL: "https://jobs.example.test", Synthetic: synthetic}}, UpdatedAt: time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)}
}

func TestMarketCountsOnlyCurrentRealListings(t *testing.T) {
	now := time.Date(2026, 9, 27, 12, 0, 0, 0, marketLocation)
	records := []marketRecord{
		marketFixture("one", "플랫폼", "서울, 경기,서울", "2026-09-27", []string{"Go", "golang", "Linux"}, false),
		marketFixture("two", "플랫폼", "경기", "채용시까지", []string{"Go", "Python"}, false),
		marketFixture("synthetic", "가상", "서울", "2026-12-31", []string{"Python"}, true),
		marketFixture("expired", "종료", "제주", "2026-09-26", []string{"Excel"}, false),
		marketFixture("unknown", "확인 필요", "제주", "", []string{"Excel"}, false),
	}
	records[0].Opportunity.Salary = "연 5,000~6,000만원"
	records = append(records, records[0]) // same URL from another collection
	summary := summarizeMarket(records, nil, now)
	if summary.TotalJobs != 2 || summary.ExcludedUnknownDeadline != 1 || summary.HasBaseline || len(summary.Trends) != 0 {
		t.Fatalf("incorrect listing eligibility: %+v", summary)
	}
	if summary.Regions[0].Name != "경기" || summary.Regions[0].Count != 2 || len(summary.Regions) != 2 {
		t.Fatalf("region count must be once per posting: %+v", summary.Regions)
	}
	if summary.SkillFrequency[0].Name != "Go" || summary.SkillFrequency[0].Count != 2 || len(summary.SkillFrequency) != 3 {
		t.Fatalf("aliases or duplicates inflated skill frequency: %+v", summary.SkillFrequency)
	}
	if len(summary.SalarySamples) != 1 || summary.SalarySamples[0].Salary != "연 5,000~6,000만원" || summary.SalarySamples[0].Source.Name != "검증 공고" || len(summary.Sources) != 1 {
		t.Fatal("public salary/provenance was lost or fabricated")
	}
}

func TestMarketRelevanceUsesTwoDistinctSkillsOrExactTitle(t *testing.T) {
	job := &career.Job{ID: "platform", Title: "플랫폼 엔지니어", Skills: []career.Requirement{{Name: "Go"}, {Name: "Linux"}}}
	cases := []struct {
		posting career.Opportunity
		want    bool
	}{
		{career.Opportunity{Title: "플랫폼 엔지니어"}, true},
		{career.Opportunity{Title: "운영 개발자", Skills: []string{"golang", "리눅스"}}, true},
		{career.Opportunity{Title: "채용", Skills: []string{"Go", "golang"}}, false},
		{career.Opportunity{Title: "AI 엔지니어", Skills: []string{"Python"}}, false},
	}
	for _, tc := range cases {
		if got := marketMatches(tc.posting, job); got != tc.want {
			t.Fatalf("relevance = %v want %v for %+v", got, tc.want, tc.posting)
		}
	}
}

func TestMarketDeadlineKoreanDateBoundary(t *testing.T) {
	lastMinute := time.Date(2026, 9, 27, 23, 59, 0, 0, marketLocation)
	for _, date := range []string{"2026-09-27", "20260927", "2026.09.27", "2026/09/27", "26-09-27"} {
		if active, known := marketDeadline(date, lastMinute); !known || !active {
			t.Errorf("listing closed before Korean date ended: %s", date)
		}
		if active, known := marketDeadline(date, lastMinute.Add(time.Minute)); !known || active {
			t.Errorf("expired listing included after Korean midnight: %s", date)
		}
	}
	if active, known := marketDeadline("2026-09-27T12:00:00+09:00", lastMinute); !known || active {
		t.Fatal("timestamp deadline not enforced")
	}
	for _, date := range []string{"", "일정 미정", "2026-02-30"} {
		if active, known := marketDeadline(date, lastMinute); active || known {
			t.Fatal("unknown deadline treated as verified active")
		}
	}
}

func TestMarketTrendsNeedComparableHistoricalCounts(t *testing.T) {
	job := &career.Job{ID: "platform", Title: "플랫폼 엔지니어", Skills: []career.Requirement{{Name: "Go"}, {Name: "Linux"}}}
	prior := marketSnapshot{Version: 1, Date: "2026-09-26", Overall: marketAggregate{SkillFrequency: []MarketCount{{Name: "Go", Count: 20}}}, ByJob: map[string]marketAggregate{
		"platform": {CriteriaHash: marketCriteria(job), SkillFrequency: []MarketCount{{Name: "Go", Count: 1}, {Name: "Linux", Count: 3}}},
	}}
	summary := MarketSummary{SkillFrequency: []MarketCount{{Name: "Go", Count: 3}, {Name: "Python", Count: 1}}, Trends: []MarketTrend{}}
	applyMarketBaseline(&summary, prior, job)
	if !summary.HasBaseline || summary.BaselineDate != "2026-09-26" || len(summary.Trends) != 3 {
		t.Fatalf("historical baseline not used: %+v", summary)
	}
	if summary.Trends[0].Name != "Go" || summary.Trends[0].Previous != 1 || summary.Trends[0].Change != 2 {
		t.Fatal("job-specific baseline mixed with overall market or wrong delta")
	}
	if last := summary.Trends[2]; last.Name != "Linux" || last.Current != 0 || last.Change != -3 {
		t.Fatal("disappeared skill demand omitted")
	}
	job.Skills = append(job.Skills, career.Requirement{Name: "Python"})
	changed := MarketSummary{Trends: []MarketTrend{}}
	applyMarketBaseline(&changed, prior, job)
	if changed.HasBaseline || len(changed.Trends) != 0 {
		t.Fatal("changed filtering criteria fabricated a comparable trend")
	}
}

func TestMarketSnapshotPersistence(t *testing.T) {
	app, _ := isolatedSSOApp(t)
	ctx := context.Background()
	deadline := time.Now().In(marketLocation).AddDate(0, 0, 2).Format("2006-01-02")
	one := marketFixture("first", "플랫폼", "서울", deadline, []string{"Go", "Linux"}, false).Opportunity
	if err := app.putRecord(ctx, "jobs", "", one.ID, one); err != nil {
		t.Fatal(err)
	}
	if err := app.captureMarketSnapshot(ctx); err != nil {
		t.Fatal(err)
	}
	today := time.Now().In(marketLocation).Format("2006-01-02")
	var firstSnapshot marketSnapshot
	if err := app.getRecord(ctx, "market_snapshot", "", today, &firstSnapshot); err != nil || firstSnapshot.Overall.TotalJobs != 1 {
		t.Fatalf("first snapshot: %+v %v", firstSnapshot, err)
	}
	request := httptest.NewRequest("GET", "/api/v1/market", nil)
	summary, err := app.marketSummary(request, nil)
	if err != nil || summary.HasBaseline || summary.TotalJobs != 1 {
		t.Fatalf("same-day snapshot was incorrectly used as a historical baseline: %+v %v", summary, err)
	}
	two := marketFixture("second", "플랫폼", "경기", deadline, []string{"Go", "Python"}, false).Opportunity
	if err := app.putRecord(ctx, "jobs", "", two.ID, two); err != nil {
		t.Fatal(err)
	}
	if err := app.captureMarketSnapshot(ctx); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := app.db.QueryRow(ctx, "SELECT count(*) FROM nr_records WHERE kind='market_snapshot'").Scan(&count); err != nil || count != 1 {
		t.Fatal("same-day snapshot created duplicate history")
	}
	// Advance the initial fixture into a prior day to exercise history selection.
	firstSnapshot.Date = time.Now().In(marketLocation).AddDate(0, 0, -1).Format("2006-01-02")
	if err := app.putRecord(ctx, "market_snapshot", "", firstSnapshot.Date, firstSnapshot); err != nil {
		t.Fatal(err)
	}
	summary, err = app.marketSummary(request, nil)
	if err != nil || !summary.HasBaseline || summary.TotalJobs != 2 || summary.BaselineDate != firstSnapshot.Date {
		t.Fatalf("prior-day baseline: %+v %v", summary, err)
	}
	for _, trend := range summary.Trends {
		if trend.Name == "Go" && (trend.Previous != 1 || trend.Current != 2 || trend.Change != 1) {
			t.Fatal("persisted trend counts differ from real import fixture")
		}
	}
}
