package validator

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
)

var (
	once     sync.Once
	instance *oidc.IDTokenVerifier
	initErr  error
)

const (
	googleIssuer = "https://accounts.google.com"
	clientID     = "480046314896-b8hul71m7hsct5v9t0dgohpgqqms5ir1.apps.googleusercontent.com"
)

// GetVerifier obtain a singleton OIDC ID token verifier instance.
// Example:
//
//	ctx := context.Background()
//	verifier, _ := GetVerifier(ctx)
//	idToken, err := verifier.Verify(ctx, rawIDToken)
func GetVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	once.Do(func() {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		provider, err := oidc.NewProvider(ctx, googleIssuer)
		if err != nil {
			initErr = err
			return
		}

		oidcConfig := &oidc.Config{
			ClientID: clientID,
		}
		instance = provider.Verifier(oidcConfig)
	})

	return instance, initErr
}

// ParseBearerToken extracts the raw token string from a Bearer token in the Authorization header.
func ParseBearerToken(authz string) string {
	authz = strings.TrimSpace(authz)
	if authz == "" {
		return ""
	}
	const bearer = "Bearer "
	if len(authz) <= len(bearer) || !strings.EqualFold(authz[:len(bearer)], bearer) {
		return ""
	}
	return strings.TrimSpace(authz[len(bearer):])
}

func IsAuthWhitelistPath(requestPath string, baseUrl string) bool {
	allowed := map[string]struct{}{
		"/":            {},
		"/login":       {},
		"/_healthz":    {},
		"/setCurrency": {},
		"/static/":    {}, 
		"/product/":   {},
	}

	if strings.HasPrefix(requestPath, "/static/") || 
       (baseUrl != "" && strings.HasPrefix(requestPath, baseUrl+"/static/")) {
        return true
    }

	if strings.HasPrefix(requestPath, "/product/") || 
       (baseUrl != "" && strings.HasPrefix(requestPath, baseUrl+"/product/")) {
        return true
    }

	if _, ok := allowed[requestPath]; ok {
		return true
	}

	if baseUrl == "" {
		return false
	}

	if strings.HasPrefix(requestPath, baseUrl) {
		trimmed := strings.TrimPrefix(requestPath, baseUrl)
		if trimmed == "" {
			trimmed = "/"
		} else if !strings.HasPrefix(trimmed, "/") {
			trimmed = "/" + trimmed
		}
		_, ok := allowed[trimmed]
		return ok
	}

	return false
}
