package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type app struct {
	clientID     string
	clientSecret string
	pkce         bool
	redirectURI  string

	verifier        *oidc.IDTokenVerifier
	provider        *oidc.Provider
	scopesSupported []string

	// Does the provider use "offline_access" scope to request a refresh token
	// or does it use "access_type=offline" (e.g. Google)?
	offlineAsScope bool

	client *http.Client

	// Device flow state
	// Only one session is possible at a time
	// Since it is an example, we don't bother locking', this is a simplicity tradeoff
	deviceFlowMutex sync.Mutex
	deviceFlowData  struct {
		sessionID       string // Unique ID for current flow session
		deviceCode      string
		userCode        string
		verificationURI string
		pollInterval    int
		token           *oauth2.Token
	}
}

var (
	// The code verifier for the PKCE request, that the app originally generated before the authorization request.
	// It is a cryptographically random string using the characters A-Z, a-z, 0-9, and the punctuation characters -._~
	// (hyphen, period, underscore, and tilde), between 43 and 128 characters long.
	// The server will later use the code verifier to verify the authorization code exchange, ensuring that the client
	// that initiated the authorization request is the same client that is exchanging the authorization code for tokens.
	//
	// See https://www.oauth.com/oauth2-servers/pkce/authorization-request/
	codeVerifier string
	// codeChallenge is derived from the code verifier. There are two methods to derive:
	//   - S256 method, it's the base64url-encoded SHA256 hash of the code verifier.
	//   - Plain method, clients that do not have the ability to perform a SHA256 hash are permitted to
	//     use the plain code verifier string as the challenge, but this is not recommended.
	// The server should recognize the code_challenge int the request, either store in the DB, or if using self-encoded authorization codes then it
	// can be included in the code itself. See https://www.oauth.com/oauth2-servers/authorization/the-authorization-response/ for more details.
	codeChallenge string
)

// exampleAppState is a static state value used for demonstration purposes.
// TODO In a production application, this should be a securely generated random string to prevent CSRF attacks.
const exampleAppState = "I want to run away and never come back"

func init() {
	codeVerifier = oauth2.GenerateVerifier()
	codeChallenge = oauth2.S256ChallengeFromVerifier(codeVerifier)
}

func runAuth(r *mux.Router, a *app) {

	var (
		issuerURL string
		listen    string
		tlsCert   string
		tlsKey    string
		rootCAs   string
		debug     bool
	)

	flag.StringVar(&a.clientID, "client-id", "example-app", "OAuth2 client ID")
	flag.StringVar(&a.clientSecret, "client-secret", "ThisIsNotASecureSecret", "OAuth2 client secret")
	flag.BoolVar(&a.pkce, "pkce", true, "Use PKCE flow")
	flag.StringVar(&a.redirectURI, "redirect-uri", "https://dex:5556/dex/callback", "Callback URL")
	flag.StringVar(&issuerURL, "issuer", "http://dex:5556/dex", "URL of OIDC issuer")  // must https. used on auto discovery
	flag.StringVar(&listen, "listen", "http://localhost:8080", "Address to listen at") // homepage
	flag.StringVar(&tlsCert, "tls-cert", "", "X509 cert file")
	flag.StringVar(&tlsKey, "tls-key", "", "Private key file")
	flag.StringVar(&rootCAs, "issuer-root-ca", "", "Root CAs for issuer")
	flag.BoolVar(&debug, "debug", false, "Debug mode")

	flag.Parse()

	if envID := os.Getenv("GITHUB_CLIENT_ID"); envID != "" {
		a.clientID = envID
	} else {
		fmt.Fprintf(os.Stderr, "Error: GITHUB_CLIENT_ID is not set in your environment.\n")
		os.Exit(1)
	}
	if envSecret := os.Getenv("GITHUB_CLIENT_SECRET"); envSecret != "" {
		a.clientSecret = envSecret
	} else {
		fmt.Fprintf(os.Stderr, "Error: GITHUB_CLIENT_SECRET is not set in your environment.\n")
		os.Exit(1)
	}

	u, err := url.Parse(a.redirectURI)
	if err != nil {
		fmt.Fprintf(os.Stderr, "parse redirect-uri: %v\n", err)
		os.Exit(1)
	}
	// listenURL, err := url.Parse(listen)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "parse listen address: %v\n", err)
	// 	os.Exit(1)
	// }

	if rootCAs != "" {
		client, err := httpClientForRootCAs(rootCAs)
		if err != nil {
			fmt.Fprintf(os.Stderr, "root CAs error: %v\n", err)
			os.Exit(1)
		}
		a.client = client
		fmt.Println("[HTTPS] Using custom root CAs from file:", rootCAs)
		fmt.Println()
	}
	if debug {
		transport := http.DefaultTransport
		if a.client != nil {
			transport = a.client.Transport
		}
		a.client = &http.Client{Transport: debugTransport{transport}}
	}
	if a.client == nil {
		a.client = http.DefaultClient
	}

	ctx := oidc.ClientContext(context.Background(), a.client)
	provider, err := oidc.NewProvider(ctx, issuerURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to query provider: %v\n", err)
		os.Exit(1)
	}
	log.Info("Successfully connected to OIDC provider at ", issuerURL)

	var s struct {
		ScopesSupported []string `json:"scopes_supported"`
	}
	if err := provider.Claims(&s); err != nil {
		fmt.Fprintf(os.Stderr, "failed to parse provider scopes: %v\n", err)
		os.Exit(1)
	}

	a.offlineAsScope = len(s.ScopesSupported) == 0
	if !a.offlineAsScope {
		for _, scope := range s.ScopesSupported {
			if scope == oidc.ScopeOfflineAccess {
				a.offlineAsScope = true
				break
			}
		}
	}

	a.provider = provider
	a.verifier = provider.Verifier(&oidc.Config{ClientID: a.clientID})
	a.scopesSupported = s.ScopesSupported

	r.HandleFunc(baseUrl+"/login", a.handleLogin)
	r.HandleFunc(u.Path, a.handleCallback)

	log.Printf("listening on %s", listen)

	// TODO HTTPS support
	// switch listenURL.Scheme {
	// case "http":
	// 	err = http.ListenAndServe(listenURL.Host, nil)
	// case "https":
	// 	err = http.ListenAndServeTLS(listenURL.Host, tlsCert, tlsKey, nil)
	// default:
	// 	err = fmt.Errorf("unsupported scheme: %q", listenURL.Scheme)
	// }

	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "error: %v\n", err)
	// 	os.Exit(2)
	// }
}

// handleLogin initiates the OAuth2 authorization code flow by constructing the appropriate scopes,
// handling PKCE parameters, and redirecting the user to the provider's authorization endpoint.
// It also supports adding a connector_id for providers like Dex and handles offline access based on
// the presence of the "offline_access" scope.
//
// The redirect URL would look like:
// htttps://AuthURL?client_id=...&code_challenge=...&code_challenge_method=S256&connector_id=...&redirect_uri=...&scope=...&state=...
//
// which will triger [handleAuthorization] function in Dex.
func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, fmt.Sprintf("failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	// Only use scopes that are checked in the form
	scopes := r.Form["extra_scopes"]
	// crossClients := r.Form["cross_client"]

	// Build complete scope list with audience scopes
	// scopes = buildScopes(scopes, crossClients)

	connectorID := ""
	if id := r.FormValue("connector_id"); id != "" {
		connectorID = id
	}

	authCodeURL := ""

	var authCodeOptions []oauth2.AuthCodeOption

	if a.pkce {
		authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_challenge", codeChallenge))
		authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_challenge_method", "S256"))
	}

	// Check if offline_access scope is present to determine offline access mode
	hasOfflineAccess := false
	for _, scope := range scopes {
		if scope == "offline_access" {
			hasOfflineAccess = true
			break
		}
	}

	if hasOfflineAccess && !a.offlineAsScope {
		// Provider uses access_type=offline instead of offline_access scope
		authCodeOptions = append(authCodeOptions, oauth2.AccessTypeOffline)
		// Remove offline_access from scopes as it's not supported
		filteredScopes := make([]string, 0, len(scopes))
		for _, scope := range scopes {
			if scope != "offline_access" {
				filteredScopes = append(filteredScopes, scope)
			}
		}
		scopes = filteredScopes
	}

	authCodeURL = a.oauth2Config(scopes).AuthCodeURL(exampleAppState, authCodeOptions...)

	// Parse the auth code URL and safely add connector_id parameter if provided
	u, err := url.Parse(authCodeURL)
	if err != nil {
		http.Error(w, "Failed to parse auth URL", http.StatusInternalServerError)
		return
	}

	if connectorID != "" {
		query := u.Query()
		query.Set("connector_id", connectorID)
		u.RawQuery = query.Encode()
	}

	http.Redirect(w, r, u.String(), http.StatusSeeOther)
}

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	var (
		err   error
		token *oauth2.Token // 这个变量实际上包含了ID Token, Access Token, Refresh Token等信息
	)

	ctx := oidc.ClientContext(r.Context(), a.client)
	oauth2Config := a.oauth2Config(nil)
	switch r.Method {
	case http.MethodGet:
		// Authorization redirect callback from OAuth2 auth flow.
		if errMsg := r.FormValue("error"); errMsg != "" {
			http.Error(w, errMsg+": "+r.FormValue("error_description"), http.StatusBadRequest)
			return
		}
		code := r.FormValue("code")
		if code == "" {
			http.Error(w, fmt.Sprintf("no code in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		if state := r.FormValue("state"); state != exampleAppState {
			http.Error(w, fmt.Sprintf("expected state %q got %q", exampleAppState, state), http.StatusBadRequest)
			return
		}

		var authCodeOptions []oauth2.AuthCodeOption
		if a.pkce {
			authCodeOptions = append(authCodeOptions, oauth2.SetAuthURLParam("code_verifier", codeVerifier))
		}

		token, err = oauth2Config.Exchange(ctx, code, authCodeOptions...)
	case http.MethodPost:
		// Form request from frontend to refresh a token.
		refresh := r.FormValue("refresh_token")
		if refresh == "" {
			http.Error(w, fmt.Sprintf("no refresh_token in request: %q", r.Form), http.StatusBadRequest)
			return
		}
		t := &oauth2.Token{
			RefreshToken: refresh,
			Expiry:       time.Now().Add(-time.Hour),
		}
		token, err = oauth2Config.TokenSource(ctx, t).Token()
	default:
		http.Error(w, fmt.Sprintf("method not implemented: %s", r.Method), http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get token: %v", err), http.StatusInternalServerError)
		return
	}

	// parseAndRenderToken(w, r, a, token)
	fmt.Printf("Token: \n %v\n", token)
}
