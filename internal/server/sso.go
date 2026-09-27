package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/jackc/pgx/v5"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strings"
	"time"
)

type Provider struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	Type             string `json:"type"`
	Enabled          bool   `json:"enabled"`
	ClientID         string `json:"clientId"`
	ClientSecret     string `json:"clientSecret"`
	HasSecret        bool   `json:"hasSecret"`
	Issuer           string `json:"issuer"`
	AuthorizationURL string `json:"authorizationUrl"`
	TokenURL         string `json:"tokenUrl"`
	UserInfoURL      string `json:"userInfoUrl"`
	Scopes           string `json:"scopes"`
	SubjectField     string `json:"subjectField"`
	EmailField       string `json:"emailField"`
	NameField        string `json:"nameField"`
}

func (a *App) providers(r *http.Request) ([]Provider, error) {
	out := []Provider{}
	err := a.config(r.Context(), "providers", &out)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return out, err
}
func preset(p Provider) Provider {
	switch p.Type {
	case "google":
		p.Issuer = "https://accounts.google.com"
	case "linkedin":
		p.Issuer = "https://www.linkedin.com"
	case "kakao":
		p.Issuer = "https://kauth.kakao.com"
	case "naver":
		p.Issuer = ""
		p.AuthorizationURL = "https://nid.naver.com/oauth2.0/authorize"
		p.TokenURL = "https://nid.naver.com/oauth2.0/token"
		p.UserInfoURL = "https://openapi.naver.com/v1/nid/me"
		p.SubjectField = "response.id"
		p.EmailField = "response.email"
		p.NameField = "response.name"
	}
	if p.Scopes == "" {
		if p.Type == "kakao" {
			p.Scopes = "openid profile_nickname account_email"
		} else if p.Type != "naver" {
			p.Scopes = "openid profile email"
		}
	}
	if p.SubjectField == "" {
		p.SubjectField = "sub"
	}
	if p.EmailField == "" {
		p.EmailField = "email"
	}
	if p.NameField == "" {
		p.NameField = "name"
	}
	return p
}
func (a *App) providersList(w http.ResponseWriter, r *http.Request) {
	ps, err := a.providers(r)
	if !a.good(w, err) {
		return
	}
	for i := range ps {
		ps[i].HasSecret = ps[i].ClientSecret != ""
		ps[i].ClientSecret = ""
	}
	respond(w, 200, ps)
}
func (a *App) providersSave(w http.ResponseWriter, r *http.Request) {
	var p Provider
	if !readJSON(w, r, &p) {
		return
	}
	ps, err := a.providers(r)
	if !a.good(w, err) {
		return
	}
	idx := -1
	if r.PathValue("id") != "" {
		for i, v := range ps {
			if v.ID == r.PathValue("id") {
				idx = i
			}
		}
		if idx < 0 {
			fail(w, 404, "SSO 설정을 찾을 수 없습니다")
			return
		}
		p.ID = ps[idx].ID
		if p.ClientSecret == "" {
			p.ClientSecret = ps[idx].ClientSecret
		}
	} else {
		p.ID = id()
	}
	p = preset(p)
	if p.Name == "" || len(p.Name) > 200 || !contains([]string{"google", "linkedin", "kakao", "naver", "keycloak", "oidc", "oauth2"}, p.Type) {
		fail(w, 400, "공급자 이름과 종류를 확인하세요")
		return
	}
	if p.Enabled {
		if p.ClientID == "" {
			fail(w, 400, "Client ID가 필요합니다")
			return
		}
		if p.Type == "oauth2" || p.Type == "naver" {
			if !validURL(p.AuthorizationURL) || !validURL(p.TokenURL) || !validURL(p.UserInfoURL) {
				fail(w, 400, "OAuth 주소를 확인하세요")
				return
			}
		} else if !validURL(p.Issuer) {
			fail(w, 400, "OIDC Issuer 주소가 필요합니다")
			return
		}
	}
	if idx >= 0 {
		ps[idx] = p
	} else {
		ps = append(ps, p)
	}
	if !a.good(w, a.setConfig(r.Context(), "providers", ps)) {
		return
	}
	a.audit(r.Context(), who(r).User.ID, "sso.save", p.ID)
	p.HasSecret = p.ClientSecret != ""
	p.ClientSecret = ""
	respond(w, 200, p)
}
func (a *App) providersDelete(w http.ResponseWriter, r *http.Request) {
	ps, err := a.providers(r)
	if !a.good(w, err) {
		return
	}
	out := []Provider{}
	found := false
	for _, p := range ps {
		if p.ID != r.PathValue("id") {
			out = append(out, p)
		} else {
			found = true
		}
	}
	if !found {
		fail(w, 404, "SSO 설정을 찾을 수 없습니다")
		return
	}
	if a.good(w, a.setConfig(r.Context(), "providers", out)) {
		a.audit(r.Context(), who(r).User.ID, "sso.delete", r.PathValue("id"))
		ok(w)
	}
}

var errProviderNotFound = errors.New("provider not found")

func (a *App) findProvider(r *http.Request) (Provider, error) {
	providers, err := a.providers(r)
	if err != nil {
		return Provider{}, err
	}
	for _, p := range providers {
		if p.ID == r.PathValue("id") && p.Enabled {
			return p, nil
		}
	}
	return Provider{}, errProviderNotFound
}
func ssoConfig(ctx context.Context, p Provider, callback string) (*oauth2.Config, *oidc.Provider, error) {
	conf := &oauth2.Config{ClientID: p.ClientID, ClientSecret: p.ClientSecret, RedirectURL: callback, Scopes: strings.Fields(strings.ReplaceAll(p.Scopes, ",", " "))}
	var provider *oidc.Provider
	var e error
	if p.Type != "oauth2" && p.Type != "naver" {
		if p.Type == "linkedin" {
			pc := oidc.ProviderConfig{IssuerURL: "https://www.linkedin.com", AuthURL: "https://www.linkedin.com/oauth/v2/authorization", TokenURL: "https://www.linkedin.com/oauth/v2/accessToken", UserInfoURL: "https://api.linkedin.com/v2/userinfo", JWKSURL: "https://www.linkedin.com/oauth/openid/jwks", Algorithms: []string{"RS256"}}
			provider = pc.NewProvider(ctx)
		} else {
			provider, e = oidc.NewProvider(ctx, p.Issuer)
			if e != nil {
				return nil, nil, e
			}
		}
		conf.Endpoint = provider.Endpoint()
	} else {
		conf.Endpoint = oauth2.Endpoint{AuthURL: p.AuthorizationURL, TokenURL: p.TokenURL}
	}
	if p.Type == "kakao" || p.Type == "naver" {
		conf.Endpoint.AuthStyle = oauth2.AuthStyleInParams
	}
	return conf, provider, nil
}

type ssoState struct {
	Provider   string `json:"provider"`
	Verifier   string `json:"verifier"`
	Nonce      string `json:"nonce"`
	Callback   string `json:"callback"`
	ConfigHash string `json:"configHash"`
}

func providerHash(p Provider) string { b, _ := json.Marshal(p); return digest(string(b)) }
func ssoContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) > 3 {
			return errors.New("redirect limit")
		}
		if len(via) > 0 && (req.URL.Host != via[0].URL.Host || req.URL.Scheme != via[0].URL.Scheme) {
			return errors.New("cross host redirect blocked")
		}
		return nil
	}})
}
func (a *App) ssoStart(w http.ResponseWriter, r *http.Request) {
	if !a.rate(r) {
		fail(w, 429, "잠시 뒤 다시 시도하세요")
		return
	}
	p, e := a.findProvider(r)
	if e != nil {
		if errors.Is(e, errProviderNotFound) {
			fail(w, 404, "활성화된 SSO 공급자를 찾을 수 없습니다")
		} else {
			a.good(w, e)
		}
		return
	}
	s, e := a.settings(r.Context())
	if !a.good(w, e) {
		return
	}
	base := s.General.BaseURL
	if base == "" {
		fail(w, 400, "관리자가 일반 설정에서 서비스 기본 URL을 먼저 입력해야 합니다")
		return
	}
	cb := base + "/api/v1/auth/sso/" + p.ID + "/callback"
	ctx, cancel := context.WithTimeout(ssoContext(r.Context()), 25*time.Second)
	defer cancel()
	conf, _, e := ssoConfig(ctx, p, cb)
	if e != nil {
		fail(w, 502, "SSO 메타데이터를 불러올 수 없습니다. Issuer와 네트워크를 확인하세요")
		return
	}
	state := token()
	data := ssoState{Provider: p.ID, Verifier: oauth2.GenerateVerifier(), Nonce: token(), Callback: cb, ConfigHash: providerHash(p)}
	b, _ := json.Marshal(data)
	_, e = a.db.Exec(r.Context(), "INSERT INTO nr_oauth_states(hash,data,expires_at) VALUES($1,$2,$3)", digest(state), a.encrypt(b), time.Now().Add(10*time.Minute))
	if !a.good(w, e) {
		return
	}
	_, _ = a.db.Exec(r.Context(), "DELETE FROM nr_oauth_states WHERE expires_at<now()")
	http.SetCookie(w, &http.Cookie{Name: "nr_oauth", Value: digest(state), Path: "/api/v1/auth/sso/", HttpOnly: true, Secure: strings.HasPrefix(base, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 600})
	opts := []oauth2.AuthCodeOption{oidc.Nonce(data.Nonce)}
	if p.Type != "naver" {
		opts = append(opts, oauth2.S256ChallengeOption(data.Verifier))
	}
	if p.Type == "kakao" {
		opts = append(opts, oauth2.SetAuthURLParam("scope", strings.Join(conf.Scopes, ",")))
	}
	http.Redirect(w, r, conf.AuthCodeURL(state, opts...), http.StatusFound)
}
func (a *App) ssoCallback(w http.ResponseWriter, r *http.Request) {
	p, e := a.findProvider(r)
	if e != nil {
		if errors.Is(e, errProviderNotFound) {
			fail(w, 400, "SSO 설정을 찾을 수 없습니다")
		} else {
			a.good(w, e)
		}
		return
	}
	state := r.URL.Query().Get("state")
	cookie, e := r.Cookie("nr_oauth")
	if e != nil || state == "" || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(digest(state))) != 1 {
		fail(w, 400, "SSO 요청 상태가 일치하지 않습니다. 로그인을 다시 시작하세요")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "nr_oauth", Value: "", Path: "/api/v1/auth/sso/", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
	var encrypted []byte
	e = a.db.QueryRow(r.Context(), "DELETE FROM nr_oauth_states WHERE hash=$1 AND expires_at>now() RETURNING data", digest(state)).Scan(&encrypted)
	if e != nil {
		fail(w, 400, "SSO 요청이 만료되었거나 이미 사용되었습니다")
		return
	}
	raw, e := a.decrypt(encrypted)
	var st ssoState
	if e != nil || json.Unmarshal(raw, &st) != nil || st.Provider != p.ID || st.ConfigHash != providerHash(p) {
		fail(w, 400, "SSO 설정이 변경되었습니다. 다시 로그인하세요")
		return
	}
	if r.URL.Query().Get("error") != "" || r.URL.Query().Get("code") == "" {
		http.Redirect(w, r, "/login?error=sso_denied", http.StatusFound)
		return
	}
	ctx, cancel := context.WithTimeout(ssoContext(r.Context()), 30*time.Second)
	defer cancel()
	conf, provider, e := ssoConfig(ctx, p, st.Callback)
	if e != nil {
		fail(w, 502, "SSO 공급자 연결에 실패했습니다")
		return
	}
	opts := []oauth2.AuthCodeOption{}
	if p.Type == "naver" {
		opts = append(opts, oauth2.SetAuthURLParam("state", state))
	} else {
		opts = append(opts, oauth2.VerifierOption(st.Verifier))
	}
	tok, e := conf.Exchange(ctx, r.URL.Query().Get("code"), opts...)
	if e != nil {
		fail(w, 502, "SSO 인증 코드 교환에 실패했습니다. Client ID/Secret과 콜백 URL을 확인하세요")
		return
	}
	claims := map[string]any{}
	if provider != nil {
		rawID, ok := tok.Extra("id_token").(string)
		if !ok {
			fail(w, 401, "OIDC ID 토큰이 없습니다")
			return
		}
		verified, e := provider.Verifier(&oidc.Config{ClientID: p.ClientID}).Verify(ctx, rawID)
		if e != nil || verified.Nonce != st.Nonce {
			fail(w, 401, "OIDC 서명·발급자·대상·만료 또는 nonce 검증에 실패했습니다")
			return
		}
		if verified.Claims(&claims) != nil {
			fail(w, 401, "OIDC 사용자 정보를 읽을 수 없습니다")
			return
		}
		claims["sub"] = verified.Subject
	} else {
		req, _ := http.NewRequestWithContext(ctx, "GET", p.UserInfoURL, nil)
		req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
		client := ctx.Value(oauth2.HTTPClient).(*http.Client)
		res, e := client.Do(req)
		if e != nil {
			fail(w, 502, "SSO 사용자 정보 조회에 실패했습니다")
			return
		}
		defer res.Body.Close()
		decoder := json.NewDecoder(io.LimitReader(res.Body, 1<<20))
		decoder.UseNumber()
		if res.StatusCode != 200 || decoder.Decode(&claims) != nil {
			fail(w, 502, "SSO 사용자 정보 응답이 올바르지 않습니다")
			return
		}
	}
	sub := claimString(claims, p.SubjectField)
	if sub == "" {
		fail(w, 401, "SSO 고유 사용자 ID가 없습니다")
		return
	}
	var uid string
	e = a.db.QueryRow(r.Context(), "SELECT user_id FROM nr_identities WHERE provider=$1 AND subject=$2", p.ID, sub).Scan(&uid)
	if e != nil {
		email := strings.ToLower(claimString(claims, p.EmailField))
		if email == "" {
			email = "sso-" + digest(p.ID + ":" + sub)[:24] + "@identity.nextrole.local"
		}
		name := claimString(claims, p.NameField)
		if name == "" {
			name = claimString(claims, "nickname")
		}
		if name == "" {
			name = "SSO 사용자"
		}
		tx, e := a.db.Begin(ctx)
		if !a.good(w, e) {
			return
		}
		defer tx.Rollback(ctx)
		uid = id()
		_, e = tx.Exec(ctx, "INSERT INTO nr_users(id,email,name,role) VALUES($1,$2,$3,'user')", uid, email, name)
		if e != nil {
			fail(w, 409, "동일한 이메일 계정이 있습니다. 보안을 위해 자동 연결하지 않습니다. 기존 계정으로 로그인하세요")
			return
		}
		_, e = tx.Exec(ctx, "INSERT INTO nr_identities(provider,subject,user_id) VALUES($1,$2,$3)", p.ID, sub, uid)
		if e == nil {
			e = tx.Commit(ctx)
		}
		if !a.good(w, e) {
			return
		}
	}
	u, e := a.user(r.Context(), uid)
	if !a.good(w, e) {
		return
	}
	if u.Disabled {
		fail(w, 403, "비활성화된 계정입니다")
		return
	}
	if a.session(w, r, u) {
		a.audit(r.Context(), uid, "auth.sso", p.ID)
		http.Redirect(w, r, "/", http.StatusFound)
	}
}
func claimString(m map[string]any, path string) string {
	var v any = m
	for _, part := range strings.Split(path, ".") {
		mm, ok := v.(map[string]any)
		if !ok {
			return ""
		}
		v = mm[part]
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return fmt.Sprintf("%.0f", x)
	case json.Number:
		return x.String()
	}
	return ""
}
