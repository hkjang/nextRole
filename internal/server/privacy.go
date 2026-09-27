package server

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/hkjang/nextRole/internal/career"
	"github.com/hkjang/nextRole/internal/connectors"
	"github.com/jackc/pgx/v5"
)

type DataPolicy struct {
	Notice  string `json:"notice"`
	Version string `json:"version"`
}
type CareerConsent struct {
	Version    string    `json:"version"`
	AcceptedAt time.Time `json:"acceptedAt"`
	NoticeHash string    `json:"noticeHash"`
}

var errConsent = errors.New("경력 정보 수집·분석 안내를 확인하고 동의해 주세요")

func defaultDataPolicy() DataPolicy {
	return DataPolicy{Version: "1", Notice: "NextRole은 경력·프로젝트·보유 기술·자격·희망 직무·희망지역·주당 학습시간과 분석에 필요한 경력·학력·산업 경험을 받아 역량 격차와 경력전환 경로를 계산합니다. 입력 자료는 공개 API 원천자료 및 합성·가공 자료와 구분하여 본인 계정에 저장합니다. 이력서 원본 파일은 저장하지 않으며 추출한 내용 중 이메일·전화번호·주민등록번호 및 이름·주소 등으로 표시된 식별정보를 제외합니다. 자동 제외는 완전하지 않으므로 입력·추출 결과에서 불필요한 개인정보를 직접 확인해 주세요. 관리자가 AI 서버를 활성화한 경우 스트리밍 AI 기능을 실행할 때 식별정보를 제외한 경력 내용이 해당 서버로 전송됩니다. 공공 API에는 개인 경력을 자동 전송하지 않습니다. 동의를 철회하면 저장된 경력·시뮬레이션·실행계획·선호·본인의 검토 요청을 삭제합니다. 계정과 개인 API 키는 유지됩니다. 동의하지 않아도 계정 설정·API 키·관리자 기능을 이용할 수 있습니다."}
}
func (a *App) dataPolicy(ctx context.Context) (DataPolicy, error) {
	p := defaultDataPolicy()
	err := a.config(ctx, "data_policy", &p)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return p, err
}
func (a *App) consent(ctx context.Context, uid string) (*CareerConsent, error) {
	var c CareerConsent
	err := a.getRecord(ctx, "consent", uid, "career", &c)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}
func (a *App) checkConsent(ctx context.Context, uid string) error {
	p, err := a.dataPolicy(ctx)
	if err != nil {
		return err
	}
	c, err := a.consent(ctx, uid)
	if err != nil {
		return err
	}
	if c == nil || c.Version != p.Version || c.NoticeHash != digest(p.Notice) {
		return errConsent
	}
	return nil
}
func (a *App) privacyGet(w http.ResponseWriter, r *http.Request) {
	p, err := a.dataPolicy(r.Context())
	if !a.good(w, err) {
		return
	}
	c, err := a.consent(r.Context(), who(r).User.ID)
	if !a.good(w, err) {
		return
	}
	if c != nil && c.NoticeHash != digest(p.Notice) {
		c = nil
	}
	respond(w, 200, map[string]any{"notice": p.Notice, "version": p.Version, "consent": c})
}
func (a *App) consentAccept(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Version  string `json:"version"`
		Accepted bool   `json:"accepted"`
	}
	if !readJSON(w, r, &in) {
		return
	}
	p, err := a.dataPolicy(r.Context())
	if !a.good(w, err) {
		return
	}
	if !in.Accepted || in.Version != p.Version {
		fail(w, 409, "현재 안내 버전을 확인하고 명시적으로 동의해 주세요")
		return
	}
	c := CareerConsent{Version: p.Version, AcceptedAt: time.Now().UTC(), NoticeHash: digest(p.Notice)}
	if !a.good(w, a.putRecord(r.Context(), "consent", who(r).User.ID, "career", c)) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "privacy.consent", p.Version)
	a.privacyGet(w, r)
}
func (a *App) consentWithdraw(w http.ResponseWriter, r *http.Request) {
	tx, err := a.db.Begin(r.Context())
	if !a.good(w, err) {
		return
	}
	defer tx.Rollback(r.Context())
	uid := who(r).User.ID
	// Delete the consent first. Career writes hold its row lock until commit,
	// so a concurrent request cannot recreate data after withdrawal finishes.
	_, err = tx.Exec(r.Context(), "DELETE FROM nr_records WHERE kind='consent' AND owner_id=$1 AND id='career'", uid)
	if err == nil {
		_, err = tx.Exec(r.Context(), "DELETE FROM nr_records WHERE owner_id=$1 AND kind IN ('profile','simulation','roadmap','feedback')", uid)
	}
	if err == nil {
		_, err = tx.Exec(r.Context(), "DELETE FROM nr_records WHERE kind='approval' AND owner_id='' AND data->>'userId'=$1", uid)
	}
	if err == nil {
		err = tx.Commit(r.Context())
	}
	if !a.good(w, err) {
		return
	}
	a.audit(r.Context(), uid, "privacy.withdraw", "career")
	a.privacyGet(w, r)
}
func (a *App) dataPolicyGet(w http.ResponseWriter, r *http.Request) {
	p, err := a.dataPolicy(r.Context())
	if a.good(w, err) {
		respond(w, 200, p)
	}
}
func (a *App) dataPolicyPut(w http.ResponseWriter, r *http.Request) {
	var in DataPolicy
	if !readJSON(w, r, &in) {
		return
	}
	in.Notice = strings.TrimSpace(in.Notice)
	in.Version = strings.TrimSpace(in.Version)
	if len(in.Notice) < 30 || len(in.Notice) > 20000 || len(in.Version) < 1 || len(in.Version) > 80 {
		fail(w, 400, "수집 안내(30~20000바이트)와 버전(1~80바이트)을 입력하세요")
		return
	}
	old, err := a.dataPolicy(r.Context())
	if !a.good(w, err) {
		return
	}
	if old.Notice != in.Notice && old.Version == in.Version {
		fail(w, 400, "수집 안내를 변경할 때 새 버전을 지정해야 합니다. 이용자가 다시 동의합니다")
		return
	}
	if !a.good(w, a.setConfig(r.Context(), "data_policy", in)) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "privacy.policy.update", in.Version)
	respond(w, 200, in)
}
func careerConsentRoute(r *http.Request) bool {
	if r.Method == "GET" {
		return r.URL.Path == "/api/v1/recommendations" || r.URL.Path == "/api/v1/opportunities"
	}
	switch r.URL.Path {
	case "/api/v1/profile", "/api/v1/profile/parse", "/api/v1/profile/upload", "/api/v1/ai/stream", "/api/v1/simulate", "/api/v1/feedback", "/api/v1/roadmap", "/api/v1/approvals":
		return true
	}
	return false
}
func (a *App) connectorPresets(w http.ResponseWriter, r *http.Request) {
	respond(w, 200, map[string]any{"presets": connectors.Presets(), "metadata": connectors.Work24Metadata()})
}

// Routine recognizers minimize direct identifiers; this is not a claim that
// arbitrary natural-language personal information can be removed perfectly.
var identifierPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)[a-z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-z0-9-]+(?:\.[a-z0-9-]+)+`),
	regexp.MustCompile(`\b[0-9]{6}\s*[-–]\s*[1-8][0-9]{6}\b`),
	regexp.MustCompile(`(?:\+82[- .]?1[016789]|\b01[016789])[- .]?[0-9]{3,4}[- .]?[0-9]{4}\b`),
	regexp.MustCompile(`(?:\+82[- .]?(?:2|[3-6][1-5]|70)|\b0(?:2|[3-6][1-5]|70))[- .][0-9]{3,4}[- .][0-9]{4}\b`),
	regexp.MustCompile(`(?im)^[\t ]*(?:성명|이름|주소|상세주소|주민등록번호|생년월일|전화번호|휴대전화|연락처|이메일|e-mail|email|full name|name|address|date of birth|phone)[\t ]*[:：][^\n]*`),
}

func redactIdentifiers(text string) string {
	for _, p := range identifierPatterns {
		text = p.ReplaceAllString(text, "[식별정보 제외]")
	}
	return text
}
func analysisProfile(p career.Profile) career.Profile {
	name := strings.TrimSpace(p.Name)
	p.Name = ""
	p.Source = career.Source{Kind: "user_input", Provider: "서비스 이용자 본인", Name: "사용자 입력", Dataset: "career_profile", Fields: []string{"currentRole", "yearsExperience", "careerBreakMonths", "education", "region", "domain", "narrative", "skills", "certifications", "preferences", "weeklyHours"}}
	clean := func(s string) string {
		s = redactIdentifiers(s)
		if len([]rune(name)) >= 2 {
			s = strings.ReplaceAll(s, name, "[이름 제외]")
		}
		return s
	}
	p.Narrative = clean(p.Narrative)
	p.CurrentRole = clean(p.CurrentRole)
	p.Education = clean(p.Education)
	p.Region = clean(p.Region)
	p.Domain = clean(p.Domain)
	p.Skills = append([]career.Skill{}, p.Skills...)
	for i := range p.Skills {
		p.Skills[i].Name = clean(p.Skills[i].Name)
	}
	p.Certifications = append([]string{}, p.Certifications...)
	for i := range p.Certifications {
		p.Certifications[i] = clean(p.Certifications[i])
	}
	p.Preferences = append([]string{}, p.Preferences...)
	for i := range p.Preferences {
		p.Preferences[i] = clean(p.Preferences[i])
	}
	return p
}

func cleanScenario(in simulationInput) simulationInput {
	in.Project = redactIdentifiers(in.Project)
	in.Certifications = append([]string(nil), in.Certifications...)
	for i := range in.Certifications {
		in.Certifications[i] = redactIdentifiers(in.Certifications[i])
	}
	in.AddedSkills = append([]career.Skill(nil), in.AddedSkills...)
	for i := range in.AddedSkills {
		in.AddedSkills[i].Name = redactIdentifiers(in.AddedSkills[i].Name)
	}
	return in
}

func (a *App) putPrivateCareerRecord(ctx context.Context, kind, owner, key string, b []byte) error {
	uid := owner
	if kind == "approval" {
		var v struct {
			UserID string `json:"userId"`
		}
		if json.Unmarshal(b, &v) != nil {
			return errConsent
		}
		uid = v.UserID
	}
	p, err := a.dataPolicy(ctx)
	if err != nil {
		return err
	}
	tx, err := a.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var raw []byte
	err = tx.QueryRow(ctx, "SELECT data FROM nr_records WHERE kind='consent' AND owner_id=$1 AND id='career' FOR SHARE", uid).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return errConsent
	}
	if err != nil {
		return err
	}
	var c CareerConsent
	if json.Unmarshal(raw, &c) != nil || c.Version != p.Version || c.NoticeHash != digest(p.Notice) {
		return errConsent
	}
	_, err = tx.Exec(ctx, "INSERT INTO nr_records(kind,owner_id,id,data) VALUES($1,$2,$3,$4) ON CONFLICT(kind,owner_id,id) DO UPDATE SET data=excluded.data,updated_at=now()", kind, owner, key, b)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}
