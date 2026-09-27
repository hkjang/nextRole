package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/hkjang/nextRole/internal/career"
	"github.com/jackc/pgx/v5"
	"io"
	"math"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (a *App) loadProfile(r *http.Request) (career.Profile, error) {
	p := career.Profile{Name: who(r).User.Name, Skills: []career.Skill{}, Certifications: []string{}, Preferences: []string{}, WeeklyHours: 10}
	e := a.getRecord(r.Context(), "profile", who(r).User.ID, "current", &p)
	if e == pgx.ErrNoRows {
		e = nil
	}
	return p, e
}
func (a *App) profileGet(w http.ResponseWriter, r *http.Request) {
	p, e := a.loadProfile(r)
	if a.good(w, e) {
		respond(w, 200, p)
	}
}
func validProfile(p career.Profile) bool {
	if p.CareerBreakMonths < 0 || p.CareerBreakMonths > 600 || len(p.Skills) > 200 || len(p.Narrative) > 100000 || p.YearsExperience < 0 || p.YearsExperience > 80 || p.WeeklyHours < 0 || p.WeeklyHours > 100 || len(p.Certifications) > 100 {
		return false
	}
	for _, s := range p.Skills {
		if strings.TrimSpace(s.Name) == "" || len(s.Name) > 150 || s.Level < 0 || s.Level > 5 || s.Years < 0 || s.Years > 80 || math.IsNaN(s.Level) || !contains([]string{"explicit", "inferred", "review"}, s.Confidence) {
			return false
		}
	}
	return true
}
func (a *App) profilePut(w http.ResponseWriter, r *http.Request) {
	var p career.Profile
	if !readJSON(w, r, &p) {
		return
	}
	for i := range p.Skills {
		p.Skills[i].Name = career.NormalizeSkill(p.Skills[i].Name)
		if p.Skills[i].Confidence == "" {
			p.Skills[i].Confidence = "explicit"
		}
	}
	if !validProfile(p) {
		fail(w, 400, "경력·역량 수준(0~5)·학습 시간을 확인하세요")
		return
	}
	if a.good(w, a.putRecord(r.Context(), "profile", who(r).User.ID, "current", p)) {
		respond(w, 200, p)
	}
}
func (a *App) profileParse(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.Text) == "" || len(in.Text) > 100000 {
		fail(w, 400, "분석할 경력을 100KB 이내로 입력하세요")
		return
	}
	respond(w, 200, career.Parse(in.Text))
}
func (a *App) profileUpload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 11<<20)
	if r.ParseMultipartForm(11<<20) != nil {
		fail(w, 400, "10MB 이하 PDF·DOCX·TXT 파일을 선택하세요")
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	f, h, e := r.FormFile("file")
	if e != nil {
		fail(w, 400, "이력서 파일을 선택하세요")
		return
	}
	defer f.Close()
	b, e := io.ReadAll(io.LimitReader(f, (10<<20)+1))
	if e != nil || len(b) > 10<<20 {
		fail(w, 400, "파일은 최대 10MB입니다")
		return
	}
	text, e := career.ExtractResume(b, h.Filename)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	respond(w, 200, career.Parse(text))
}
func (a *App) jobs(ctx context.Context) ([]career.Job, error) {
	out := career.SeedJobs()
	records, e := a.records(ctx, "occupations", "")
	if e != nil {
		return nil, e
	}
	byID := map[string]int{}
	for i, j := range out {
		byID[j.ID] = i
	}
	for _, b := range records {
		var j career.Job
		if json.Unmarshal(b, &j) != nil {
			continue
		}
		if idx, ok := byID[j.ID]; ok {
			out[idx] = j
		} else {
			byID[j.ID] = len(out)
			out = append(out, j)
		}
	}
	return out, nil
}
func (a *App) job(r *http.Request, jobID string) (career.Job, bool) {
	jobs, e := a.jobs(r.Context())
	if e != nil {
		return career.Job{}, false
	}
	for _, j := range jobs {
		if j.ID == jobID {
			return j, true
		}
	}
	return career.Job{}, false
}
func (a *App) jobsGet(w http.ResponseWriter, r *http.Request) {
	jobs, e := a.jobs(r.Context())
	if !a.good(w, e) {
		return
	}
	q := strings.ToLower(r.URL.Query().Get("q"))
	out := []career.Job{}
	for _, j := range jobs {
		b, _ := json.Marshal(j)
		if q == "" || strings.Contains(strings.ToLower(string(b)), q) {
			out = append(out, j)
		}
	}
	respond(w, 200, out)
}
func weights(s Settings) career.Weights {
	return career.Weights{Skill: s.Scoring.Skill, Transfer: s.Scoring.Transfer, Experience: s.Scoring.Experience, Domain: s.Scoring.Domain, Education: s.Scoring.Education, Preference: s.Scoring.Preference}
}
func (a *App) recommendations(w http.ResponseWriter, r *http.Request) {
	p, e := a.loadProfile(r)
	if !a.good(w, e) {
		return
	}
	jobs, e := a.jobs(r.Context())
	if !a.good(w, e) {
		return
	}
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	feedback, _ := a.records(r.Context(), "feedback", who(r).User.ID)
	disliked := map[string]bool{}
	liked := map[string]bool{}
	for _, b := range feedback {
		var v map[string]string
		_ = json.Unmarshal(b, &v)
		disliked[v["jobId"]] = v["preference"] == "dislike"
		liked[v["jobId"]] = v["preference"] == "like"
	}
	market, e := a.marketRecords(r.Context())
	if !a.good(w, e) {
		return
	}
	list := []career.Simulation{}
	for _, j := range jobs {
		if !disliked[j.ID] {
			sim := career.SimulateWithCatalog(p, j, 6, nil, weights(s), jobs)
			sim.MarketDemand = summarizeMarket(market, &j, time.Now()).TotalJobs
			sim.RankingReason = fmt.Sprintf("적합도 우선, 선호 +3 및 실제 유효 공고 표본 %d건의 수요 가산(최대 +3)을 정렬에만 반영합니다. 적합도 점수는 변경하지 않습니다.", sim.MarketDemand)
			list = append(list, sim)
		}
	}
	sort.SliceStable(list, func(i, j int) bool {
		ai := list[i].Score + math.Min(3, math.Log2(float64(list[i].MarketDemand)+1))
		aj := list[j].Score + math.Min(3, math.Log2(float64(list[j].MarketDemand)+1))
		if liked[list[i].Job.ID] {
			ai += 3
		}
		if liked[list[j].Job.ID] {
			aj += 3
		}
		return ai > aj
	})
	if len(list) > 5 {
		list = list[:5]
	}
	respond(w, 200, list)
}

type simulationInput struct {
	JobID          string         `json:"jobId"`
	Months         int            `json:"months"`
	AddedSkills    []career.Skill `json:"addedSkills"`
	Certifications []string       `json:"certifications"`
	Project        string         `json:"project"`
	Save           bool           `json:"save"`
}

func (a *App) calculate(r *http.Request, in simulationInput) (career.Simulation, error) {
	p, e := a.loadProfile(r)
	if e != nil {
		return career.Simulation{}, e
	}
	j, found := a.job(r, in.JobID)
	if !found {
		return career.Simulation{}, errInvalid
	}
	if in.Months == 0 {
		in.Months = 6
	}
	if !contains([]string{"3", "6", "12"}, map[int]string{3: "3", 6: "6", 12: "12"}[in.Months]) || len(in.AddedSkills) > 100 || len(in.Certifications) > 100 || len(in.Project) > 10000 {
		return career.Simulation{}, errInvalid
	}
	for _, v := range in.AddedSkills {
		if v.Name == "" || len(v.Name) > 150 || v.Level < 0 || v.Level > 5 {
			return career.Simulation{}, errInvalid
		}
	}
	s, e := a.settings(r.Context())
	if e != nil {
		return career.Simulation{}, e
	}
	catalog, e := a.jobs(r.Context())
	if e != nil {
		return career.Simulation{}, e
	}
	baseline := career.SimulateWithCatalog(p, j, in.Months, nil, weights(s), catalog)
	p.Certifications = append(p.Certifications, in.Certifications...)
	if in.Project != "" {
		extracted := career.Parse(in.Project)
		for _, sk := range extracted.Skills {
			sk.Level = math.Min(2, sk.Level)
			in.AddedSkills = append(in.AddedSkills, sk)
		}
	}
	sim := career.SimulateWithCatalog(p, j, in.Months, in.AddedSkills, weights(s), catalog)
	sim.BaselineScore = baseline.Score
	if in.Project != "" {
		sim.Warnings = append(sim.Warnings, "프로젝트 내용의 추출 역량을 가정으로 적용했습니다. 실제 경력 프로필은 변경되지 않습니다.")
	}
	return sim, nil
}

type simulationRecord struct {
	career.Simulation
	Scenario simulationInput `json:"scenario"`
}

func (a *App) simulate(w http.ResponseWriter, r *http.Request) {
	var in simulationInput
	if !readJSON(w, r, &in) {
		return
	}
	sim, e := a.calculate(r, in)
	if e != nil {
		fail(w, 400, "목표 직무·기간(3/6/12개월)·추가 역량을 확인하세요")
		return
	}
	if in.Save {
		sim.ID = id()
		now := time.Now().UTC()
		sim.CreatedAt = &now
		if !a.good(w, a.putRecord(r.Context(), "simulation", who(r).User.ID, sim.ID, simulationRecord{sim, in})) {
			return
		}
	}
	respond(w, 200, simulationRecord{sim, in})
}
func (a *App) simulations(w http.ResponseWriter, r *http.Request) {
	rs, e := a.records(r.Context(), "simulation", who(r).User.ID)
	if a.good(w, e) {
		respond(w, 200, rs)
	}
}
func (a *App) simulationDelete(w http.ResponseWriter, r *http.Request) {
	tag, e := a.db.Exec(r.Context(), "DELETE FROM nr_records WHERE kind='simulation' AND owner_id=$1 AND id=$2", who(r).User.ID, r.PathValue("id"))
	if !a.good(w, e) {
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "시뮬레이션을 찾을 수 없습니다")
		return
	}
	ok(w)
}
func (a *App) opportunities(r *http.Request, jobID string) (map[string]any, error) {
	j, found := a.job(r, jobID)
	if !found {
		return nil, errInvalid
	}
	s, e := a.settings(r.Context())
	if e != nil {
		return nil, e
	}
	recruit := []career.Opportunity{}
	training := []career.Opportunity{}
	profile, _ := a.loadProfile(r)
	for _, kind := range []string{"jobs", "training"} {
		rs, e := a.records(r.Context(), kind, "")
		if e != nil {
			return nil, e
		}
		for _, b := range rs {
			var o career.Opportunity
			if json.Unmarshal(b, &o) != nil {
				continue
			}
			match := strings.Contains(strings.ToLower(o.Title), strings.ToLower(j.Title))
			for _, skill := range j.Skills {
				for _, has := range o.Skills {
					if career.NormalizeSkill(has) == career.NormalizeSkill(skill.Name) {
						match = true
					}
				}
			}
			if !match {
				continue
			}
			if profile.Region != "" && o.Region != "" && !strings.Contains(o.Region, profile.Region) && !strings.Contains(o.Region, "전국") && !strings.Contains(o.Region, "원격") {
				continue
			}
			if o.Deadline != "" {
				t, err := time.Parse("2006-01-02", o.Deadline)
				if err == nil && t.Before(time.Now().Add(-24*time.Hour)) {
					continue
				}
			}
			if kind == "jobs" {
				recruit = append(recruit, o)
			} else {
				training = append(training, o)
			}
		}
	}
	synthetic := false
	if s.General.DemoEnabled {
		sj, st := career.SeedOpportunities(j)
		if len(recruit) == 0 {
			recruit = sj
			synthetic = true
		}
		if len(training) == 0 {
			training = st
			synthetic = true
		}
	}
	counts := map[string]int{}
	for _, o := range recruit {
		if o.Source.Synthetic {
			continue
		}
		seen := map[string]bool{}
		for _, skill := range o.Skills {
			k := career.NormalizeSkill(skill)
			if !seen[k] {
				counts[k]++
				seen[k] = true
			}
		}
	}
	freq := []map[string]any{}
	for name, n := range counts {
		freq = append(freq, map[string]any{"name": name, "count": n})
	}
	sort.Slice(freq, func(i, j int) bool { return freq[i]["count"].(int) > freq[j]["count"].(int) })
	return map[string]any{"jobs": recruit, "training": training, "skillFrequency": freq, "synthetic": synthetic}, nil
}
func (a *App) opportunitiesGet(w http.ResponseWriter, r *http.Request) {
	v, e := a.opportunities(r, r.URL.Query().Get("jobId"))
	if e != nil {
		fail(w, 400, "목표 직무를 선택하세요")
		return
	}
	respond(w, 200, v)
}
func (a *App) feedback(w http.ResponseWriter, r *http.Request) {
	var in struct {
		JobID      string `json:"jobId"`
		Preference string `json:"preference"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	_, found := a.job(r, in.JobID)
	if !found || !contains([]string{"like", "dislike", "neutral"}, in.Preference) {
		fail(w, 400, "직무와 선호를 확인하세요")
		return
	}
	if a.good(w, a.putRecord(r.Context(), "feedback", who(r).User.ID, in.JobID, in)) {
		ok(w)
	}
}
func (a *App) roadmapGet(w http.ResponseWriter, r *http.Request) {
	v := map[string]any{"completed": []string{}}
	e := a.getRecord(r.Context(), "roadmap", who(r).User.ID, "current", &v)
	if e == pgx.ErrNoRows {
		e = nil
	}
	if a.good(w, e) {
		respond(w, 200, v)
	}
}
func (a *App) roadmapPut(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Completed []string `json:"completed"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if len(in.Completed) > 2000 {
		fail(w, 400, "저장 가능한 작업 수를 초과했습니다")
		return
	}
	if a.good(w, a.putRecord(r.Context(), "roadmap", who(r).User.ID, "current", in)) {
		respond(w, 200, in)
	}
}

func (a *App) marketGet(w http.ResponseWriter, r *http.Request) {
	var job *career.Job
	if jobID := r.URL.Query().Get("jobId"); jobID != "" {
		j, found := a.job(r, jobID)
		if !found {
			fail(w, 404, "목표 직무를 찾을 수 없습니다")
			return
		}
		job = &j
	}
	result, e := a.marketSummary(r, job)
	if a.good(w, e) {
		respond(w, 200, result)
	}
}
