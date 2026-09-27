package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
)

// Mock IdP exercises real discovery, an HTTP authorization redirect, PKCE token
// exchange and JWKS-backed JWT verification. No live provider account is used.
type testIDP struct {
	server      *httptest.Server
	signer      jose.Signer
	wrongSigner jose.Signer
	public      jose.JSONWebKey
	mu          sync.Mutex
	codes       map[string]url.Values
	mutate      func(map[string]any)
	badSig      bool
	omitID      bool
	tokenCalls  int
}

func newTestIDP(t *testing.T) *testIDP {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	options := (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "nextrole-test-key")
	signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, options)
	if err != nil {
		t.Fatal(err)
	}
	wrongSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: wrong}, options)
	if err != nil {
		t.Fatal(err)
	}
	idp := &testIDP{signer: signer, wrongSigner: wrongSigner, public: jose.JSONWebKey{Key: &key.PublicKey, KeyID: "nextrole-test-key", Algorithm: "RS256", Use: "sig"}, codes: map[string]url.Values{}}
	idp.server = httptest.NewServer(http.HandlerFunc(idp.serve))
	t.Cleanup(idp.server.Close)
	return idp
}

func (idp *testIDP) serve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.URL.Path {
	case "/.well-known/openid-configuration":
		_ = json.NewEncoder(w).Encode(map[string]any{
			"issuer": idp.server.URL, "authorization_endpoint": idp.server.URL + "/authorize",
			"token_endpoint": idp.server.URL + "/token", "jwks_uri": idp.server.URL + "/jwks",
			"response_types_supported": []string{"code"}, "subject_types_supported": []string{"public"},
			"id_token_signing_alg_values_supported": []string{"RS256"}, "code_challenge_methods_supported": []string{"S256"},
		})
	case "/jwks":
		_ = json.NewEncoder(w).Encode(jose.JSONWebKeySet{Keys: []jose.JSONWebKey{idp.public}})
	case "/userinfo":
		if r.Header.Get("Authorization") != "Bearer mock-access-token" {
			http.Error(w, "missing access token", 401)
			return
		}
		_, _ = w.Write([]byte(`{"id":900719925474099312345,"email":"oauth-user@example.test","name":"OAuth 사용자"}`))
	case "/authorize":
		q := r.URL.Query()
		if q.Get("client_id") != "nextrole-client" || q.Get("response_type") != "code" || q.Get("state") == "" || q.Get("nonce") == "" || q.Get("code_challenge_method") != "S256" || q.Get("code_challenge") == "" {
			http.Error(w, "authorization request missing security parameters", 400)
			return
		}
		code := token()
		idp.mu.Lock()
		idp.codes[code] = q
		idp.mu.Unlock()
		cb, err := url.Parse(q.Get("redirect_uri"))
		if err != nil {
			http.Error(w, "invalid callback", 400)
			return
		}
		args := cb.Query()
		args.Set("code", code)
		args.Set("state", q.Get("state"))
		cb.RawQuery = args.Encode()
		http.Redirect(w, r, cb.String(), 302)
	case "/token":
		if r.ParseForm() != nil {
			http.Error(w, "bad form", 400)
			return
		}
		idp.mu.Lock()
		defer idp.mu.Unlock()
		idp.tokenCalls++
		user, password, ok := r.BasicAuth()
		if !ok || user != "nextrole-client" || password != "idp-test-secret" || r.Form.Get("grant_type") != "authorization_code" {
			w.WriteHeader(401)
			_, _ = w.Write([]byte(`{"error":"invalid_client"}`))
			return
		}
		q, exists := idp.codes[r.Form.Get("code")]
		delete(idp.codes, r.Form.Get("code"))
		proof := sha256.Sum256([]byte(r.Form.Get("code_verifier")))
		if !exists || r.Form.Get("redirect_uri") != q.Get("redirect_uri") || base64.RawURLEncoding.EncodeToString(proof[:]) != q.Get("code_challenge") {
			w.WriteHeader(400)
			_, _ = w.Write([]byte(`{"error":"invalid_grant"}`))
			return
		}
		claims := map[string]any{"iss": idp.server.URL, "aud": "nextrole-client", "sub": "subject-123", "nonce": q.Get("nonce"), "exp": time.Now().Add(time.Hour).Unix(), "iat": time.Now().Unix(), "email": "oidc-user@example.test", "email_verified": true, "name": "OIDC 사용자", "role": "admin"}
		if idp.mutate != nil {
			idp.mutate(claims)
		}
		signer := idp.signer
		if idp.badSig {
			signer = idp.wrongSigner
		}
		signed, err := jwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			http.Error(w, "JWT creation failed", 500)
			return
		}
		response := map[string]any{"access_token": "mock-access-token", "token_type": "Bearer", "expires_in": 3600}
		if !idp.omitID {
			response["id_token"] = signed
		}
		_ = json.NewEncoder(w).Encode(response)
	default:
		http.NotFound(w, r)
	}
}

func isolatedSSOApp(t *testing.T) (*App, *httptest.Server) {
	t.Helper()
	dsn := os.Getenv("TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEST_POSTGRES_DSN is not set")
	}
	ctx := context.Background()
	base, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schemaName := "sso_test_" + id()
	if _, err := base.Exec(ctx, "CREATE SCHEMA "+schemaName); err != nil {
		base.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = base.Exec(context.Background(), "DROP SCHEMA "+schemaName+" CASCADE")
		base.Close()
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schemaName)
	u.RawQuery = q.Encode()
	t.Setenv("POSTGRES_DSN", u.String())
	t.Setenv("BOOTSTRAP_ADMIN", "admin@example.test")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "NextRole-Test-Password42")
	t.Setenv("ENCRYPTION_KEY", base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{73}, 32)))
	app, err := New(ctx, "sso-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.Close)
	ts := httptest.NewServer(app.Handler())
	t.Cleanup(ts.Close)
	s, err := app.settings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	s.General.BaseURL = ts.URL
	if err := app.setConfig(ctx, "settings", s); err != nil {
		t.Fatal(err)
	}
	return app, ts
}

func newSSOBrowser(t *testing.T) *http.Client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Client{Jar: jar, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Timeout: 5 * time.Second}
}

func ssoGET(t *testing.T, client *http.Client, address string, status int) *http.Response {
	t.Helper()
	response, err := client.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != status {
		b, _ := io.ReadAll(response.Body)
		response.Body.Close()
		t.Fatalf("SSO HTTP status %d, want %d; body: %s", response.StatusCode, status, b)
	}
	return response
}

func beginSSOFlow(t *testing.T, app *App, ts *httptest.Server, provider Provider) (*http.Client, string, *http.Cookie) {
	t.Helper()
	// Each flow models a fresh test scenario; IP rate-limit behavior is tested
	// separately and should not couple the JWT rejection cases to each other.
	app.mu.Lock()
	app.limits = map[string]rateEntry{}
	app.mu.Unlock()
	client := newSSOBrowser(t)
	res := ssoGET(t, client, ts.URL+"/api/v1/auth/sso/"+provider.ID+"/start", 302)
	authURL := res.Header.Get("Location")
	var cookie *http.Cookie
	for _, c := range res.Cookies() {
		if c.Name == "nr_oauth" {
			cookie = c
		}
	}
	res.Body.Close()
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatal("SSO state cookie is missing or not protected")
	}
	authorize := ssoGET(t, client, authURL, 302)
	callback := authorize.Header.Get("Location")
	authorize.Body.Close()
	return client, callback, cookie
}

func TestOIDCFlowAndSecurity(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	idp := newTestIDP(t)
	provider := preset(Provider{ID: "local-oidc", Name: "테스트 IdP", Type: "oidc", Enabled: true, ClientID: "nextrole-client", ClientSecret: "idp-test-secret", Issuer: idp.server.URL})
	if err := app.setConfig(context.Background(), "providers", []Provider{provider}); err != nil {
		t.Fatal(err)
	}
	t.Run("origin requires matching scheme and host", func(t *testing.T) {
		for _, origin := range []string{strings.Replace(ts.URL, "http://", "https://", 1), ts.URL + "/", "null"} {
			req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/auth/logout", nil)
			req.Header.Set("Origin", origin)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			response.Body.Close()
			if response.StatusCode != http.StatusForbidden {
				t.Fatal("mutation accepted an invalid or alternate-scheme origin")
			}
		}
	})
	reset := func() {
		idp.mu.Lock()
		idp.mutate, idp.badSig, idp.omitID = nil, false, false
		idp.mu.Unlock()
	}
	t.Run("complete flow and state replay", func(t *testing.T) {
		reset()
		client, callback, stateCookie := beginSSOFlow(t, app, ts, provider)
		res := ssoGET(t, client, callback, 302)
		if res.Header.Get("Location") != "/" {
			t.Fatal("successful SSO did not enter application")
		}
		var sessionCookie *http.Cookie
		for _, cookie := range res.Cookies() {
			if cookie.Name == "nr_session" {
				sessionCookie = cookie
			}
		}
		res.Body.Close()
		if sessionCookie == nil || !sessionCookie.HttpOnly || sessionCookie.SameSite != http.SameSiteLaxMode {
			t.Fatal("SSO did not create a protected session")
		}
		me := ssoGET(t, client, ts.URL+"/api/v1/me", 200)
		var user User
		if json.NewDecoder(me.Body).Decode(&user) != nil || user.Role != "user" || user.Email != "oidc-user@example.test" {
			t.Fatalf("wrong identity or provider role escalation: %#v", user)
		}
		me.Body.Close()
		// Replay with the original cookie proves server-side single consumption,
		// rather than relying only on deletion of the browser cookie.
		req, _ := http.NewRequest("GET", callback, nil)
		req.AddCookie(stateCookie)
		res, err := (&http.Client{}).Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		if res.StatusCode != 400 {
			t.Fatal("consumed SSO state was replayed")
		}
	})
	for _, tc := range []struct {
		name   string
		mutate func(map[string]any)
		badSig bool
		omitID bool
	}{
		{name: "forged signature", badSig: true},
		{name: "wrong nonce", mutate: func(c map[string]any) { c["nonce"] = "attacker-nonce" }},
		{name: "wrong issuer", mutate: func(c map[string]any) { c["iss"] = "https://attacker.example.test" }},
		{name: "wrong audience", mutate: func(c map[string]any) { c["aud"] = "other-client" }},
		{name: "expired token", mutate: func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() }},
		{name: "missing id token", omitID: true},
		{name: "missing subject", mutate: func(c map[string]any) { delete(c, "sub") }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reset()
			idp.mu.Lock()
			idp.mutate, idp.badSig, idp.omitID = tc.mutate, tc.badSig, tc.omitID
			idp.mu.Unlock()
			client, callback, _ := beginSSOFlow(t, app, ts, provider)
			res := ssoGET(t, client, callback, 401)
			b, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if strings.Contains(string(b), "idp-test-secret") || strings.Contains(string(b), "mock-access-token") {
				t.Fatal("SSO provider credential leaked in error")
			}
			res = ssoGET(t, client, ts.URL+"/api/v1/me", 401)
			res.Body.Close()
		})
	}
	t.Run("state bound to browser", func(t *testing.T) {
		reset()
		_, callback, _ := beginSSOFlow(t, app, ts, provider)
		res := ssoGET(t, newSSOBrowser(t), callback, 400)
		res.Body.Close()
	})
	t.Run("tampered state", func(t *testing.T) {
		reset()
		client, callback, _ := beginSSOFlow(t, app, ts, provider)
		u, _ := url.Parse(callback)
		q := u.Query()
		q.Set("state", "attacker-state")
		u.RawQuery = q.Encode()
		res := ssoGET(t, client, u.String(), 400)
		res.Body.Close()
	})
	t.Run("expired authorization state", func(t *testing.T) {
		reset()
		client, callback, _ := beginSSOFlow(t, app, ts, provider)
		u, _ := url.Parse(callback)
		if _, err := app.db.Exec(context.Background(), "UPDATE nr_oauth_states SET expires_at=now()-interval '1 minute' WHERE hash=$1", digest(u.Query().Get("state"))); err != nil {
			t.Fatal(err)
		}
		res := ssoGET(t, client, callback, 400)
		res.Body.Close()
	})
	t.Run("provider config changes invalidate state", func(t *testing.T) {
		reset()
		client, callback, _ := beginSSOFlow(t, app, ts, provider)
		changed := provider
		changed.ClientID = "different-client"
		if err := app.setConfig(context.Background(), "providers", []Provider{changed}); err != nil {
			t.Fatal(err)
		}
		res := ssoGET(t, client, callback, 400)
		res.Body.Close()
		if err := app.setConfig(context.Background(), "providers", []Provider{provider}); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("email collision never links existing admin", func(t *testing.T) {
		reset()
		idp.mu.Lock()
		idp.mutate = func(c map[string]any) {
			c["sub"], c["email"], c["email_verified"] = "new-attacker-subject", "admin@example.test", false
		}
		idp.mu.Unlock()
		client, callback, _ := beginSSOFlow(t, app, ts, provider)
		res := ssoGET(t, client, callback, 409)
		res.Body.Close()
		res = ssoGET(t, client, ts.URL+"/api/v1/me", 401)
		res.Body.Close()
		var count int
		if err := app.db.QueryRow(context.Background(), "SELECT count(*) FROM nr_identities WHERE subject='new-attacker-subject'").Scan(&count); err != nil || count != 0 {
			t.Fatal("unverified email linked to existing administrator")
		}
	})
	t.Run("disabled SSO user rejected", func(t *testing.T) {
		reset()
		if _, err := app.db.Exec(context.Background(), "UPDATE nr_users SET disabled=true WHERE email='oidc-user@example.test'"); err != nil {
			t.Fatal(err)
		}
		client, callback, _ := beginSSOFlow(t, app, ts, provider)
		res := ssoGET(t, client, callback, 403)
		res.Body.Close()
		res = ssoGET(t, client, ts.URL+"/api/v1/me", 401)
		res.Body.Close()
	})
}

func TestSSORedirectPolicy(t *testing.T) {
	client := ssoContext(context.Background()).Value(oauth2.HTTPClient).(*http.Client)
	first, _ := http.NewRequest(http.MethodGet, "https://idp.example.test/config", nil)
	for _, tc := range []struct {
		address string
		allowed bool
	}{
		{"https://idp.example.test/new-config", true},
		{"http://idp.example.test/new-config", false},
		{"https://attacker.example.test/new-config", false},
	} {
		redirected, _ := http.NewRequest(http.MethodGet, tc.address, nil)
		err := client.CheckRedirect(redirected, []*http.Request{first})
		if (err == nil) != tc.allowed {
			t.Errorf("incorrect redirect policy for %s", tc.address)
		}
	}
}

func TestOAuthNumericSubjectPreservesPrecision(t *testing.T) {
	app, ts := isolatedSSOApp(t)
	idp := newTestIDP(t)
	provider := Provider{ID: "local-oauth", Name: "테스트 OAuth", Type: "oauth2", Enabled: true, ClientID: "nextrole-client", ClientSecret: "idp-test-secret", AuthorizationURL: idp.server.URL + "/authorize", TokenURL: idp.server.URL + "/token", UserInfoURL: idp.server.URL + "/userinfo", SubjectField: "id", EmailField: "email", NameField: "name", Scopes: "profile email"}
	if err := app.setConfig(context.Background(), "providers", []Provider{provider}); err != nil {
		t.Fatal(err)
	}
	client, callback, _ := beginSSOFlow(t, app, ts, provider)
	res := ssoGET(t, client, callback, http.StatusFound)
	res.Body.Close()
	var subject string
	if err := app.db.QueryRow(context.Background(), "SELECT subject FROM nr_identities WHERE provider=$1", provider.ID).Scan(&subject); err != nil {
		t.Fatal(err)
	}
	if subject != "900719925474099312345" {
		t.Fatalf("numeric OAuth subject lost precision: %s", subject)
	}
}
