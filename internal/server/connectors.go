package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/hkjang/nextRole/internal/career"
	"github.com/hkjang/nextRole/internal/connectors"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

func (a *App) connectorConfigs(r *http.Request) ([]connectors.Config, error) {
	cs := []connectors.Config{}
	err := a.config(r.Context(), "connectors", &cs)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return cs, err
}
func (a *App) connectorsList(w http.ResponseWriter, r *http.Request) {
	cs, err := a.connectorConfigs(r)
	if !a.good(w, err) {
		return
	}
	for i := range cs {
		cs[i] = connectors.UpgradeWork24Config(cs[i]).Mask()
	}
	respond(w, 200, cs)
}
func (a *App) connectorsSave(w http.ResponseWriter, r *http.Request) {
	var c connectors.Config
	if !readJSON(w, r, &c) {
		return
	}
	cs, err := a.connectorConfigs(r)
	if !a.good(w, err) {
		return
	}
	idx := -1
	if r.PathValue("id") != "" {
		for i, v := range cs {
			if v.ID == r.PathValue("id") {
				idx = i
			}
		}
		if idx < 0 {
			fail(w, 404, "연동 설정을 찾을 수 없습니다")
			return
		}
		c.ID = cs[idx].ID
		if c.APIKey == "" {
			c.APIKey = cs[idx].APIKey
		}
		if c.DSN == "" {
			c.DSN = cs[idx].DSN
		}
		c.LastSync = cs[idx].LastSync
		c.LastError = cs[idx].LastError
	} else {
		c.ID = id()
	}
	if c.Name == "" || len(c.Name) > 200 {
		fail(w, 400, "연동 이름을 입력하세요")
		return
	}
	c = connectors.UpgradeWork24Config(c)
	if e := connectors.Validate(c); e != nil {
		fail(w, 400, e.Error())
		return
	}
	if idx < 0 {
		cs = append(cs, c)
	} else {
		cs[idx] = c
	}
	if a.good(w, a.setConfig(r.Context(), "connectors", cs)) {
		a.audit(r.Context(), who(r).User.ID, "connector.save", c.ID)
		respond(w, 200, c.Mask())
	}
}
func (a *App) connectorsDelete(w http.ResponseWriter, r *http.Request) {
	cs, err := a.connectorConfigs(r)
	if !a.good(w, err) {
		return
	}
	out := []connectors.Config{}
	found := false
	for _, c := range cs {
		if c.ID != r.PathValue("id") {
			out = append(out, c)
		} else {
			found = true
		}
	}
	if !found {
		fail(w, 404, "연동 설정을 찾을 수 없습니다")
		return
	}
	if a.good(w, a.setConfig(r.Context(), "connectors", out)) {
		a.audit(r.Context(), who(r).User.ID, "connector.delete", r.PathValue("id"))
		ok(w)
	}
}
func (a *App) findConnector(r *http.Request) (connectors.Config, bool, error) {
	configs, err := a.connectorConfigs(r)
	if err != nil {
		return connectors.Config{}, false, err
	}
	for _, c := range configs {
		if c.ID == r.PathValue("id") {
			return c, true, nil
		}
	}
	return connectors.Config{}, false, nil
}
func (a *App) connectorsTest(w http.ResponseWriter, r *http.Request) {
	c, found, e := a.findConnector(r)
	if !a.good(w, e) {
		return
	}
	if !found {
		fail(w, 404, "연동 설정을 찾을 수 없습니다")
		return
	}
	rs, e := connectors.Fetch(r.Context(), c)
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	preview := rs
	if len(preview) > 3 {
		preview = preview[:3]
	}
	respond(w, 200, map[string]any{"ok": true, "count": len(rs), "preview": preview})
}
func (a *App) connectorsSync(w http.ResponseWriter, r *http.Request) {
	c, found, e := a.findConnector(r)
	if !a.good(w, e) {
		return
	}
	if !found {
		fail(w, 404, "연동 설정을 찾을 수 없습니다")
		return
	}
	if !c.Enabled {
		fail(w, 400, "연동을 활성화한 후 동기화하세요")
		return
	}
	rs, e := connectors.Fetch(r.Context(), c)
	if e == nil && len(rs) > 0 {
		e = a.importRecords(r, c.Dataset, rs, c.Name, c.ID)
	}
	cs, configErr := a.connectorConfigs(r)
	if !a.good(w, configErr) {
		return
	}
	for i := range cs {
		if cs[i].ID == c.ID {
			if e != nil {
				cs[i].LastError = e.Error()
			} else {
				cs[i].LastError = ""
				cs[i].LastSync = time.Now().UTC().Format(time.RFC3339)
			}
		}
	}
	if !a.good(w, a.setConfig(r.Context(), "connectors", cs)) {
		return
	}
	if e != nil {
		fail(w, 400, e.Error())
		return
	}
	a.audit(r.Context(), who(r).User.ID, "connector.sync", c.ID)
	respond(w, 200, map[string]any{"ok": true, "count": len(rs)})
}
func (a *App) dataImport(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Dataset string           `json:"dataset"`
		Records []map[string]any `json:"records"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	if e := a.importRecords(r, in.Dataset, in.Records, "관리자 가져오기", "manual"); e != nil {
		fail(w, 400, e.Error())
		return
	}
	a.audit(r.Context(), who(r).User.ID, "data.import", in.Dataset)
	respond(w, 200, map[string]any{"ok": true, "count": len(in.Records)})
}
func (a *App) importRecords(r *http.Request, dataset string, rs []map[string]any, source, prefix string) error {
	if !contains([]string{"jobs", "training", "occupations", "occupation_details", "ncs_units"}, dataset) || len(rs) == 0 || len(rs) > 5000 {
		return fmt.Errorf("데이터 종류와 레코드 수(1~5000)를 확인하세요")
	}
	// Do not modify a caller's batch while namespacing IDs; retries must keep
	// exactly the same identities, including each training course round.
	encoded, err := json.Marshal(rs)
	if err != nil {
		return fmt.Errorf("레코드를 JSON으로 변환할 수 없습니다")
	}
	var batch []map[string]any
	if json.Unmarshal(encoded, &batch) != nil {
		return fmt.Errorf("레코드 형식을 확인하세요")
	}
	rs = batch
	for i, v := range rs {
		src, _ := v["source"].(map[string]any)
		preset, _ := src["dataset"].(string)
		for _, m := range connectors.Work24Metadata() {
			if m.ID != preset {
				continue
			}
			if m.Dataset != dataset {
				return fmt.Errorf("%d번째 고용24 출처와 저장 데이터 종류가 일치하지 않습니다", i+1)
			}
			raw, ok := v["raw"].(map[string]any)
			if !ok {
				return fmt.Errorf("%d번째 고용24 레코드의 선별 원천 필드가 필요합니다", i+1)
			}
			var fields []string
			if values, ok := src["fields"].([]any); ok {
				fields = []string{}
				for _, value := range values {
					name, ok := value.(string)
					if !ok {
						return fmt.Errorf("고용24 원천 필드 목록 형식을 확인하세요")
					}
					fields = append(fields, name)
				}
			}
			projected, err := connectors.Normalize([]map[string]any{raw}, connectors.Config{PresetID: preset, Dataset: dataset, SelectedFields: fields})
			if err != nil {
				return err
			}
			rs[i] = projected[0]
		}
	}
	type record struct {
		id string
		b  []byte
	}
	pending := []record{}
	importIDs := map[string]bool{}
	for i, v := range rs {
		if _, ok := v["id"]; !ok || v["id"] == nil || v["id"] == "" {
			v["id"] = digest(fmt.Sprint(v["title"], v["organization"], v["url"]))[:24]
		}
		rawID := fmt.Sprint(v["id"])
		if len(rawID) > 200 {
			return fmt.Errorf("%d번째 ID가 너무 깁니다", i+1)
		}
		if importIDs[rawID] {
			return fmt.Errorf("%d번째 레코드 ID가 배치 안에서 중복됩니다. 과정·회차 등 고유 필드를 조합하세요", i+1)
		}
		importIDs[rawID] = true
	}
	for i, v := range rs {
		rawID := fmt.Sprint(v["id"])
		v["id"] = prefix + ":" + rawID
		b, _ := json.Marshal(v)
		if dataset == "occupation_details" || dataset == "ncs_units" {
			var original career.SourceRecord
			if json.Unmarshal(b, &original) != nil || strings.TrimSpace(original.Title) == "" || len(original.Raw) == 0 {
				return fmt.Errorf("%d번째 원천자료에 title, raw와 출처가 필요합니다", i+1)
			}
			if dataset == "occupation_details" && original.OccupationCode == "" || dataset == "ncs_units" && original.NCSCode == "" {
				return fmt.Errorf("%d번째 원천자료에 공식 직업코드 또는 NCS 코드가 필요합니다", i+1)
			}
			if !validURL(original.Source.URL) || original.Source.Kind == "" || original.Source.RecordID == "" {
				return fmt.Errorf("%d번째 원천자료의 공식 출처 URL·종류·원천 ID를 확인하세요", i+1)
			}
			if original.Source.Name == "" {
				original.Source.Name = source
			}
			b, _ = json.Marshal(original)
		} else if dataset == "occupations" {
			var j career.Job
			if json.Unmarshal(b, &j) != nil || strings.TrimSpace(j.Title) == "" || len(j.Skills) == 0 {
				return fmt.Errorf("%d번째 직무에 title과 skills[{name,level,weight}]가 필요합니다", i+1)
			}
			for _, sk := range j.Skills {
				if sk.Name == "" || sk.Level < 0 || sk.Level > 5 || sk.Weight < 0 {
					return fmt.Errorf("%d번째 직무 역량 수준을 확인하세요", i+1)
				}
			}
			for k, bridge := range j.BridgeIDs {
				if importIDs[bridge] {
					j.BridgeIDs[k] = prefix + ":" + bridge
				}
			}
			if j.Source.Name == "" {
				j.Source.Name = source
			}
			if j.Source.URL != "" && !validURL(j.Source.URL) {
				return fmt.Errorf("출처 URL 형식을 확인하세요")
			}
			b, _ = json.Marshal(j)
		} else {
			var o career.Opportunity
			if json.Unmarshal(b, &o) != nil || strings.TrimSpace(o.Title) == "" {
				return fmt.Errorf("%d번째 레코드 title과 필드 형식을 확인하세요", i+1)
			}
			if o.Source.Name == "" {
				o.Source.Name = source
			}
			if !o.Source.Synthetic && (!validURL(o.URL) || o.Organization == "") {
				return fmt.Errorf("%d번째 실제 공고·훈련에 유효한 url과 organization이 필요합니다", i+1)
			}
			if o.URL != "" && !validURL(o.URL) {
				return fmt.Errorf("공고 URL 형식을 확인하세요")
			}
			if o.Source.URL != "" && !validURL(o.Source.URL) {
				return fmt.Errorf("출처 URL 형식을 확인하세요")
			}
			if o.Source.Dataset == connectors.Work24Jobs || o.Source.Dataset == connectors.Work24Training || strings.Contains(o.Source.Name, "고용24") || strings.Contains(o.Source.URL, "work24.go.kr") || strings.Contains(o.Source.URL, "openapi.work.go.kr") {
				// Work24 list responses contain no skill requirement field. This
				// also corrects legacy imports that guessed skills from the title.
				o.Skills = []string{}
				o.SkillsOrigin = ""
			} else if len(o.Skills) > 0 {
				o.SkillsOrigin = "admin_reviewed"
			} else {
				o.Skills = []string{}
				o.SkillsOrigin = ""
			}
			b, _ = json.Marshal(o)
		}
		pending = append(pending, record{id: v["id"].(string), b: b})
	}
	tx, e := a.db.Begin(r.Context())
	if e != nil {
		return fmt.Errorf("가져오기 트랜잭션을 시작할 수 없습니다")
	}
	defer tx.Rollback(r.Context())
	if _, e = tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(472093)"); e != nil {
		return fmt.Errorf("원천자료 가져오기 잠금을 얻을 수 없습니다")
	}
	for _, v := range pending {
		if _, e = tx.Exec(r.Context(), "INSERT INTO nr_records(kind,owner_id,id,data) VALUES($1,'',$2,$3) ON CONFLICT(kind,owner_id,id) DO UPDATE SET data=excluded.data,updated_at=now()", dataset, v.id, v.b); e != nil {
			return fmt.Errorf("데이터 저장에 실패했습니다")
		}
	}
	if tx.Commit(r.Context()) != nil {
		return fmt.Errorf("데이터 저장에 실패했습니다")
	}
	if dataset == "jobs" {
		if err := a.captureMarketSnapshot(r.Context()); err != nil {
			a.audit(r.Context(), who(r).User.ID, "market.snapshot.failed", dataset)
		}
	}
	return nil
}
