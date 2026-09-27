package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hkjang/nextRole/internal/career"
	"github.com/jackc/pgx/v5"
)

var sourceDatasets = []string{"occupation_details", "ncs_units", "jobs", "training", "occupations"}

type MappingSkill struct {
	InternalSkillID string   `json:"internalSkillId"`
	Name            string   `json:"name"`
	Aliases         []string `json:"aliases"`
	Level           float64  `json:"level"`
	Weight          float64  `json:"weight"`
	SourceRecordIDs []string `json:"sourceRecordIds"`
	Basis           string   `json:"basis"`
}

type SkillMapping struct {
	JobCodes           []string       `json:"jobCodes"`
	ID                 string         `json:"id"`
	Title              string         `json:"title"`
	OccupationRecordID string         `json:"occupationRecordId"`
	OccupationCode     string         `json:"occupationCode"`
	NCSRecordIDs       []string       `json:"ncsRecordIds"`
	Skills             []MappingSkill `json:"skills"`
	Basis              string         `json:"basis"`
	Version            int            `json:"version"`
	Status             string         `json:"status"`
	ReviewedBy         string         `json:"reviewedBy,omitempty"`
	ReviewedAt         string         `json:"reviewedAt,omitempty"`
	UpdatedAt          string         `json:"updatedAt"`
	// Fingerprints exclude collection timestamps: fetching identical fields does
	// not invalidate a review, while changing their contents does.
	SourceHashes      map[string]string `json:"sourceHashes,omitempty"`
	Valid             bool              `json:"valid"`
	ValidationWarning string            `json:"validationWarning,omitempty"`
}

type sourceDataRow struct {
	ID             string         `json:"id"`
	Dataset        string         `json:"dataset"`
	Title          string         `json:"title"`
	Source         career.Source  `json:"source"`
	OccupationCode string         `json:"occupationCode,omitempty"`
	NCSCode        string         `json:"ncsCode,omitempty"`
	OfficialLevel  string         `json:"officialLevel,omitempty"`
	Raw            map[string]any `json:"raw"`
	UpdatedAt      string         `json:"updatedAt"`
}

func normalizeSource(s career.Source) career.Source {
	if s.Synthetic {
		s.Kind = "synthetic"
	}
	if s.Kind == "" {
		s.Kind = "external"
	}
	return s
}

func (a *App) dataOverview(w http.ResponseWriter, r *http.Request) {
	rows, err := a.db.Query(r.Context(), `SELECT kind, CASE WHEN data->'source'->>'synthetic'='true' THEN 'synthetic' ELSE coalesce(nullif(data->'source'->>'kind',''),'external') END,coalesce(data->'source'->>'provider',''),count(*),max(updated_at) FROM nr_records WHERE owner_id='' AND kind=ANY($1) GROUP BY 1,2,3 ORDER BY 1,2,3`, sourceDatasets)
	if !a.good(w, err) {
		return
	}
	defer rows.Close()
	datasets := []map[string]any{}
	for rows.Next() {
		var dataset, kind, provider string
		var count int
		var updated time.Time
		if !a.good(w, rows.Scan(&dataset, &kind, &provider, &count, &updated)) {
			return
		}
		datasets = append(datasets, map[string]any{"dataset": dataset, "kind": kind, "provider": provider, "count": count, "updatedAt": updated.UTC().Format(time.RFC3339)})
	}
	if !a.good(w, rows.Err()) {
		return
	}
	var count, published int
	if !a.good(w, a.db.QueryRow(r.Context(), `SELECT count(*),count(*) FILTER (WHERE data->>'status'='published') FROM nr_records WHERE kind='skill_mapping' AND owner_id=''`).Scan(&count, &published)) {
		return
	}
	respond(w, 200, map[string]any{"datasets": datasets, "mappingCount": count, "publishedMappingCount": published})
}

func queryInt(r *http.Request, name string, fallback, min, max int) int {
	v, e := strconv.Atoi(r.URL.Query().Get(name))
	if e != nil {
		return fallback
	}
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func (a *App) sourceRecords(w http.ResponseWriter, r *http.Request) {
	dataset := r.URL.Query().Get("dataset")
	if dataset != "" && !contains(sourceDatasets, dataset) {
		fail(w, 400, "원천 데이터 종류를 확인하세요")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) > 500 {
		fail(w, 400, "검색어는 500바이트 이내입니다")
		return
	}
	limit, offset := queryInt(r, "limit", 50, 1, 200), queryInt(r, "offset", 0, 0, 1000000)
	filter := `owner_id='' AND kind=ANY($1) AND ($2='' OR kind=$2) AND ($3='' OR position(lower($3) in lower(id||' '||coalesce(data->>'title','')||' '||coalesce(data->'source'->>'provider','')))>0)`
	var total int
	if !a.good(w, a.db.QueryRow(r.Context(), "SELECT count(*) FROM nr_records WHERE "+filter, sourceDatasets, dataset, q).Scan(&total)) {
		return
	}
	rows, err := a.db.Query(r.Context(), "SELECT kind,id,data,updated_at FROM nr_records WHERE "+filter+" ORDER BY updated_at DESC,id LIMIT $4 OFFSET $5", sourceDatasets, dataset, q, limit, offset)
	if !a.good(w, err) {
		return
	}
	defer rows.Close()
	out := []sourceDataRow{}
	for rows.Next() {
		var row sourceDataRow
		var b []byte
		var updated time.Time
		if !a.good(w, rows.Scan(&row.Dataset, &row.ID, &b, &updated)) {
			return
		}
		var v career.SourceRecord
		if !a.good(w, json.Unmarshal(b, &v)) {
			return
		}
		row.Title, row.Source, row.Raw = v.Title, normalizeSource(v.Source), v.Raw
		row.OccupationCode, row.NCSCode, row.OfficialLevel = v.OccupationCode, v.NCSCode, v.OfficialLevel
		if row.Raw == nil {
			row.Raw = map[string]any{}
		}
		row.UpdatedAt = updated.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	if a.good(w, rows.Err()) {
		respond(w, 200, map[string]any{"records": out, "total": total, "limit": limit, "offset": offset})
	}
}

func (a *App) sourceRecordDelete(w http.ResponseWriter, r *http.Request) {
	dataset, key := r.PathValue("dataset"), r.PathValue("id")
	if !contains(sourceDatasets, dataset) {
		fail(w, 400, "원천 데이터 종류를 확인하세요")
		return
	}
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472093)"); !a.good(w, e) {
		return
	}
	tag, e := tx.Exec(r.Context(), "DELETE FROM nr_records WHERE owner_id='' AND kind=$1 AND id=$2", dataset, key)
	if !a.good(w, e) {
		return
	}
	if tag.RowsAffected() == 0 {
		fail(w, 404, "원천 레코드를 찾을 수 없습니다")
		return
	}
	if !a.good(w, tx.Commit(r.Context())) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "source.delete", dataset+":"+key)
	if dataset == "jobs" {
		_ = a.captureMarketSnapshot(r.Context())
	}
	ok(w)
}

func cleanMapping(m *SkillMapping) error {
	m.Title = strings.TrimSpace(m.Title)
	m.Basis = strings.TrimSpace(m.Basis)
	if m.Title == "" || len(m.Title) > 200 || m.OccupationRecordID == "" || len(m.OccupationRecordID) > 500 || m.Basis == "" || len(m.Basis) > 10000 || len(m.NCSRecordIDs) > 100 || len(m.Skills) == 0 || len(m.Skills) > 200 {
		return errors.New("직무명·원천 직무·매핑 근거·역량 1~200개를 입력하세요")
	}
	if len(m.JobCodes) > 100 {
		return errors.New("채용 직종코드는 100개 이내입니다")
	}
	for i, code := range m.JobCodes {
		code = strings.TrimSpace(code)
		if code == "" || len(code) > 100 {
			return errors.New("채용 직종코드를 확인하세요")
		}
		m.JobCodes[i] = code
	}
	ids, names, aliases := map[string]bool{}, map[string]bool{}, map[string]string{}
	for i := range m.Skills {
		s := &m.Skills[i]
		s.Name = career.NormalizeSkill(s.Name)
		s.InternalSkillID = strings.TrimSpace(s.InternalSkillID)
		s.Basis = strings.TrimSpace(s.Basis)
		if s.InternalSkillID == "" || len(s.InternalSkillID) > 150 || s.Name == "" || len(s.Name) > 150 || s.Level <= 0 || s.Level > 5 || s.Weight <= 0 || s.Weight > 100 || math.IsNaN(s.Level) || math.IsInf(s.Level, 0) || math.IsNaN(s.Weight) || math.IsInf(s.Weight, 0) || s.Basis == "" || len(s.Basis) > 4000 || len(s.SourceRecordIDs) == 0 || len(s.SourceRecordIDs) > 100 || len(s.Aliases) > 50 {
			return errors.New("각 역량에 내부 ID·이름·관리자 수준(0 초과~5)·가중치(0 초과~100)·원천·근거를 입력하세요")
		}
		name := strings.ToLower(s.Name)
		if ids[s.InternalSkillID] || names[name] {
			return errors.New("내부 역량 ID와 이름은 중복될 수 없습니다")
		}
		ids[s.InternalSkillID], names[name] = true, true
		for k, alias := range s.Aliases {
			alias = career.NormalizeSkill(alias)
			key := strings.ToLower(alias)
			if alias == "" || len(alias) > 150 {
				return errors.New("별칭은 비어 있지 않은 150바이트 이내 이름이어야 합니다")
			}
			if target, ok := aliases[key]; ok && target != name {
				return errors.New("같은 별칭을 서로 다른 역량에 연결할 수 없습니다")
			}
			aliases[key] = name
			s.Aliases[k] = alias
		}
	}
	for alias, target := range aliases {
		if names[alias] && alias != target {
			return errors.New("다른 내부 역량 이름을 별칭으로 사용할 수 없습니다")
		}
	}
	return nil
}

type recordReader interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func mappingReferences(ctx context.Context, db recordReader, m SkillMapping) (map[string]career.SourceRecord, map[string]string, error) {
	refs := map[string]string{m.OccupationRecordID: "occupation_details"}
	for _, key := range m.NCSRecordIDs {
		if key == "" || len(key) > 500 || key == m.OccupationRecordID {
			return nil, nil, errors.New("NCS 원천 ID를 확인하세요")
		}
		refs[key] = "ncs_units"
	}
	for _, s := range m.Skills {
		for _, key := range s.SourceRecordIDs {
			if _, ok := refs[key]; !ok {
				return nil, nil, errors.New("역량 원천은 선택한 직무 또는 NCS 원천 레코드여야 합니다")
			}
		}
	}
	records, hashes := map[string]career.SourceRecord{}, map[string]string{}
	for key, dataset := range refs {
		var data []byte
		if e := db.QueryRow(ctx, "SELECT data FROM nr_records WHERE kind=$1 AND owner_id='' AND id=$2", dataset, key).Scan(&data); e != nil {
			return nil, nil, fmt.Errorf("참조 원천이 없거나 읽을 수 없습니다: %s", key)
		}
		var rec career.SourceRecord
		if json.Unmarshal(data, &rec) != nil {
			return nil, nil, errors.New("원천 레코드 형식이 올바르지 않습니다")
		}
		if rec.Source.Synthetic || rec.Source.Kind == "synthetic" {
			return nil, nil, errors.New("합성 데이터는 공공 원천 매핑으로 게시할 수 없습니다")
		}
		records[key] = rec
		rec.Source.RetrievedAt = ""
		rec.Source.Fields = append([]string(nil), rec.Source.Fields...)
		sort.Strings(rec.Source.Fields)
		b, _ := json.Marshal(rec)
		hashes[key] = digest(string(b))
	}
	return records, hashes, nil
}

func mappingIsCurrent(m SkillMapping, hashes map[string]string) bool {
	if m.Status != "published" || m.ReviewedBy == "" || m.ReviewedAt == "" || len(m.SourceHashes) != len(hashes) {
		return false
	}
	for key, hash := range hashes {
		if m.SourceHashes[key] != hash {
			return false
		}
	}
	return true
}
func (a *App) loadMappings(ctx context.Context) ([]SkillMapping, error) {
	rows, e := a.records(ctx, "skill_mapping", "")
	if e != nil {
		return nil, e
	}
	out := []SkillMapping{}
	for _, b := range rows {
		var m SkillMapping
		if json.Unmarshal(b, &m) != nil {
			return nil, errors.New("저장된 매핑 형식이 올바르지 않습니다")
		}
		_, hashes, e := mappingReferences(ctx, a.db, m)
		m.Valid = e == nil && mappingIsCurrent(m, hashes)
		if e != nil {
			m.ValidationWarning = e.Error()
		} else if m.Status == "published" && !m.Valid {
			m.ValidationWarning = "원천 내용이 변경되었습니다. 다시 확인한 뒤 게시하세요."
		} else if m.Status != "published" {
			m.ValidationWarning = "초안은 계산에 사용하지 않습니다."
		}
		out = append(out, m)
	}
	return out, nil
}
func (a *App) mappingsList(w http.ResponseWriter, r *http.Request) {
	m, e := a.loadMappings(r.Context())
	if a.good(w, e) {
		respond(w, 200, m)
	}
}

func writeMapping(ctx context.Context, tx pgx.Tx, m SkillMapping) error {
	b, e := json.Marshal(m)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO nr_records(kind,owner_id,id,data) VALUES('skill_mapping','',$1,$2) ON CONFLICT(kind,owner_id,id) DO UPDATE SET data=excluded.data,updated_at=now()`, m.ID, b); e != nil {
		return e
	}
	_, e = tx.Exec(ctx, `INSERT INTO nr_records(kind,owner_id,id,data) VALUES('mapping_history','',$1,$2)`, fmt.Sprintf("%s:%09d", m.ID, m.Version), b)
	return e
}

func (a *App) mappingsSave(w http.ResponseWriter, r *http.Request) {
	var m SkillMapping
	if !readJSON(w, r, &m) {
		return
	}
	if e := cleanMapping(&m); e != nil {
		fail(w, 400, e.Error())
		return
	}
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472093)"); !a.good(w, e) {
		return
	}
	version := 1
	if key := r.PathValue("id"); key != "" {
		var b []byte
		e = tx.QueryRow(r.Context(), "SELECT data FROM nr_records WHERE kind='skill_mapping' AND owner_id='' AND id=$1 FOR UPDATE", key).Scan(&b)
		if errors.Is(e, pgx.ErrNoRows) {
			fail(w, 404, "매핑을 찾을 수 없습니다")
			return
		}
		if !a.good(w, e) {
			return
		}
		var old SkillMapping
		if !a.good(w, json.Unmarshal(b, &old)) {
			return
		}
		if m.Version != old.Version {
			fail(w, 409, "다른 변경이 있습니다. 최신 매핑을 불러오세요")
			return
		}
		m.ID = old.ID
		version = old.Version + 1
	} else {
		m.ID = id()
	}
	refs, _, e := mappingReferences(r.Context(), tx, m)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	m.OccupationCode = refs[m.OccupationRecordID].OccupationCode
	m.Version, m.Status, m.UpdatedAt = version, "draft", time.Now().UTC().Format(time.RFC3339)
	m.ReviewedBy, m.ReviewedAt, m.SourceHashes, m.Valid, m.ValidationWarning = "", "", nil, false, "초안은 계산에 사용하지 않습니다."
	if !a.good(w, writeMapping(r.Context(), tx, m)) || !a.good(w, tx.Commit(r.Context())) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "mapping.save", m.ID)
	respond(w, 200, m)
}

func (a *App) mappingPublish(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version int `json:"version"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472093)"); !a.good(w, e) {
		return
	}
	var b []byte
	e = tx.QueryRow(r.Context(), "SELECT data FROM nr_records WHERE kind='skill_mapping' AND owner_id='' AND id=$1 FOR UPDATE", r.PathValue("id")).Scan(&b)
	if errors.Is(e, pgx.ErrNoRows) {
		fail(w, 404, "매핑을 찾을 수 없습니다")
		return
	}
	if !a.good(w, e) {
		return
	}
	var m SkillMapping
	if !a.good(w, json.Unmarshal(b, &m)) {
		return
	}
	if in.Version != m.Version {
		fail(w, 409, "다른 변경이 있습니다. 최신 버전을 검토하세요")
		return
	}
	if e = cleanMapping(&m); e != nil {
		fail(w, 400, e.Error())
		return
	}
	refs, hashes, e := mappingReferences(r.Context(), tx, m)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	m.OccupationCode = refs[m.OccupationRecordID].OccupationCode
	m.Version++
	m.Status = "published"
	m.ReviewedBy = who(r).User.ID
	m.ReviewedAt = time.Now().UTC().Format(time.RFC3339)
	m.UpdatedAt = m.ReviewedAt
	m.SourceHashes = hashes
	m.Valid = true
	m.ValidationWarning = ""
	if !a.good(w, writeMapping(r.Context(), tx, m)) || !a.good(w, tx.Commit(r.Context())) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "mapping.publish", m.ID)
	respond(w, 200, m)
}

func (a *App) mappingsDelete(w http.ResponseWriter, r *http.Request) {
	tx, e := a.db.Begin(r.Context())
	if !a.good(w, e) {
		return
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472093)"); !a.good(w, e) {
		return
	}
	var b []byte
	e = tx.QueryRow(r.Context(), "SELECT data FROM nr_records WHERE kind='skill_mapping' AND owner_id='' AND id=$1 FOR UPDATE", r.PathValue("id")).Scan(&b)
	if errors.Is(e, pgx.ErrNoRows) {
		fail(w, 404, "매핑을 찾을 수 없습니다")
		return
	}
	if !a.good(w, e) {
		return
	}
	var m SkillMapping
	if !a.good(w, json.Unmarshal(b, &m)) {
		return
	}
	m.Version++
	m.Status = "deleted"
	m.Valid = false
	m.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if !a.good(w, writeMapping(r.Context(), tx, m)) {
		return
	}
	if _, e = tx.Exec(r.Context(), "DELETE FROM nr_records WHERE kind='skill_mapping' AND owner_id='' AND id=$1", m.ID); !a.good(w, e) {
		return
	}
	if !a.good(w, tx.Commit(r.Context())) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "mapping.delete", m.ID)
	ok(w)
}
func (a *App) mappingHistory(w http.ResponseWriter, r *http.Request) {
	rows, e := a.db.Query(r.Context(), "SELECT data FROM nr_records WHERE kind='mapping_history' AND owner_id='' AND data->>'id'=$1 ORDER BY (data->>'version')::int DESC LIMIT 500", r.PathValue("id"))
	if !a.good(w, e) {
		return
	}
	defer rows.Close()
	out := []json.RawMessage{}
	for rows.Next() {
		var b []byte
		if !a.good(w, rows.Scan(&b)) {
			return
		}
		out = append(out, json.RawMessage(b))
	}
	if a.good(w, rows.Err()) {
		respond(w, 200, out)
	}
}

func derivedJob(m SkillMapping, refs map[string]career.SourceRecord) career.Job {
	source := refs[m.OccupationRecordID]
	ids := []string{m.OccupationRecordID}
	ids = append(ids, m.NCSRecordIDs...)
	sort.Strings(ids)
	j := career.Job{ID: "mapping:" + m.ID, RecruitmentCodes: m.JobCodes, Title: m.Title, Category: "검토된 직무 매핑", Description: source.Description, OccupationCode: source.OccupationCode, SourceCode: source.OccupationCode, NCSCodes: []string{}, Skills: []career.Requirement{}, Regions: []string{}, BridgeIDs: []string{}, SkillAliases: map[string]string{}, Source: career.Source{Kind: "derived", Name: "관리자 검토 직무·역량 매핑", Provider: "NextRole", Dataset: "skill_mapping", RecordID: m.ID, Version: strconv.Itoa(m.Version), Basis: m.Basis, SourceIDs: ids, URL: source.Source.URL, RetrievedAt: m.ReviewedAt}}
	for _, key := range m.NCSRecordIDs {
		for _, code := range []string{refs[key].NCSCode, rawText(refs[key].Raw, "job_sdvn_cd")} {
			if code != "" && !contains(j.NCSCodes, code) {
				j.NCSCodes = append(j.NCSCodes, code)
			}
		}
	}
	for _, sk := range m.Skills {
		j.Skills = append(j.Skills, career.Requirement{Name: sk.Name, Level: sk.Level, Weight: sk.Weight})
		for _, alias := range sk.Aliases {
			j.SkillAliases[alias] = sk.Name
		}
	}
	return j
}
func (a *App) derivedJobs(ctx context.Context) ([]career.Job, error) {
	mappings, e := a.loadMappings(ctx)
	if e != nil {
		return nil, e
	}
	out := []career.Job{}
	for _, m := range mappings {
		if !m.Valid {
			continue
		}
		refs, hashes, e := mappingReferences(ctx, a.db, m)
		if e != nil || !mappingIsCurrent(m, hashes) {
			continue
		}
		out = append(out, derivedJob(m, refs))
	}
	return out, nil
}

func rawText(raw map[string]any, key string) string {
	v, _ := raw[key].(string)
	return strings.TrimSpace(v)
}
