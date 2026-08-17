package auth

import (
	"errors"
	"fmt"
	"net/http"
	"os"
)

// TokenEnv overrides the stored credential entirely — the escape hatch for CI and for
// short-lived tokens that are not worth persisting.
const TokenEnv = "WABA_ACCESS_TOKEN"

// Bearer is the one Authenticator the Graph API needs: every WhatsApp Cloud API call is
// authorized by "Authorization: Bearer <access-token>". A single-method API scales the
// provider pattern down to this.
type Bearer struct {
	Token string
}

// Apply sets the Authorization header on an outgoing request.
func (b *Bearer) Apply(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+b.Token)
}

// ResolveToken returns the access token for an account, applying precedence:
// WABA_ACCESS_TOKEN > the credential store. The account name is the store key.
func ResolveToken(store Store, account string) (string, error) {
	if t := os.Getenv(TokenEnv); t != "" {
		return t, nil
	}
	cred, err := store.Get(account)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", fmt.Errorf("no access token for account %q — run `waba auth login`, or set %s", account, TokenEnv)
		}
		return "", err
	}
	if cred.Token == "" {
		return "", fmt.Errorf("stored credential for account %q is empty — run `waba auth login`", account)
	}
	return cred.Token, nil
}
