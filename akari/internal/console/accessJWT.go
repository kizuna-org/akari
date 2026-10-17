package console

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	accessIssuerEnv                 = "AKARI_ACCESS_JWT_ISSUER"
	accessAudienceEnv               = "AKARI_ACCESS_JWT_AUD"
	accessKeyCacheTTL               = 5 * time.Minute
	accessBitsPerByte               = 8
	accessJWKSFetchTimeout          = 5 * time.Second
	accessJWKSFailureBackoff        = 5 * time.Second
	accessUnknownKidRefreshInterval = 30 * time.Second
)

var (
	errAccessJWTConfigIncomplete = errors.New("access JWT issuer and audience must be configured together")
	errAccessJWTIssuerInvalid    = errors.New("access JWT issuer must be an HTTPS origin")
	errAccessJWTMethodInvalid    = errors.New("unexpected JWT signing method")
	errAccessJWTKeyIDMissing     = errors.New("JWT key ID is missing")
	errAccessJWTKeyMissing       = errors.New("JWT signing key was not found")
	errAccessJWKSStatus          = errors.New("cloudflare Access returned an unsuccessful signing keys response")
	errAccessJWKSRefreshBackoff  = errors.New("cloudflare Access signing keys refresh is in backoff")
	errAccessJWKSSize            = errors.New("cloudflare Access signing keys response is too large")
	errAccessJWKExponent         = errors.New("invalid cloudflare Access RSA exponent")
	errAccessJWKKey              = errors.New("invalid cloudflare Access RSA key")
	errAccessJWKSNoKeys          = errors.New("cloudflare Access returned no valid RSA signing keys")
)

type accessJWTVerifier struct {
	issuer   string
	audience string
	client   *http.Client

	mutex                 sync.Mutex
	refreshGate           chan struct{}
	generation            uint64
	refreshErr            error
	keys                  map[string]*rsa.PublicKey
	cacheTill             time.Time
	refreshRetryAfter     time.Time
	lastUnknownKidRefresh time.Time
}

type accessJWKS struct {
	Keys []struct {
		KeyID string `json:"kid"`
		Type  string `json:"kty"`
		Alg   string `json:"alg"`
		N     string `json:"n"`
		E     string `json:"e"`
	} `json:"keys"`
}

func newAccessJWTVerifierFromEnvironment() (*accessJWTVerifier, bool, error) {
	issuer := strings.TrimRight(strings.TrimSpace(os.Getenv(accessIssuerEnv)), "/")

	audience := strings.TrimSpace(os.Getenv(accessAudienceEnv))
	if issuer == "" && audience == "" {
		return nil, false, nil
	}

	if issuer == "" || audience == "" {
		return nil, true, fmt.Errorf("%w: %s and %s", errAccessJWTConfigIncomplete, accessIssuerEnv, accessAudienceEnv)
	}

	if !isHTTPSOrigin(issuer) {
		return nil, true, fmt.Errorf("%w: %s", errAccessJWTIssuerInvalid, accessIssuerEnv)
	}

	return &accessJWTVerifier{
		issuer:   issuer,
		audience: audience,
		client: &http.Client{Timeout: accessJWKSFetchTimeout, Transport: http.DefaultTransport,
			CheckRedirect: nil, Jar: nil},
		mutex:                 sync.Mutex{},
		refreshGate:           make(chan struct{}, 1),
		generation:            0,
		refreshErr:            nil,
		keys:                  make(map[string]*rsa.PublicKey),
		cacheTill:             time.Time{},
		refreshRetryAfter:     time.Time{},
		lastUnknownKidRefresh: time.Time{},
	}, true, nil
}

func isHTTPSOrigin(issuer string) bool {
	parsed, err := url.Parse(issuer)
	if err != nil {
		return false
	}

	return parsed.Scheme == "https" && parsed.Host != "" && parsed.Path == "" &&
		parsed.RawQuery == "" && parsed.Fragment == ""
}

func (verifier *accessJWTVerifier) middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		assertion := request.Header.Get("Cf-Access-Jwt-Assertion")
		if assertion == "" || verifier.verify(request.Context(), assertion) != nil {
			http.Error(writer, "Cloudflare Access authentication required", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(writer, request)
	})
}

func (verifier *accessJWTVerifier) verify(ctx context.Context, assertion string) error {
	claims := new(jwt.RegisteredClaims)
	_, err := jwt.ParseWithClaims(assertion, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, errAccessJWTMethodInvalid
		}

		keyID, ok := token.Header["kid"].(string)

		if !ok || keyID == "" {
			return nil, errAccessJWTKeyIDMissing
		}

		return verifier.publicKey(ctx, keyID)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}),
		jwt.WithIssuer(verifier.issuer), jwt.WithAudience(verifier.audience),
		jwt.WithExpirationRequired(), jwt.WithIssuedAt())

	return err
}

func (verifier *accessJWTVerifier) publicKey(ctx context.Context, keyID string) (*rsa.PublicKey, error) {
	verifier.mutex.Lock()
	key := verifier.keys[keyID]
	now := time.Now()
	needsRefresh := verifier.cacheNeedsRefresh(now)

	if !needsRefresh && key != nil {
		verifier.mutex.Unlock()

		return key, nil
	}

	if needsRefresh && now.Before(verifier.refreshRetryAfter) {
		verifier.mutex.Unlock()

		return nil, errAccessJWKSRefreshBackoff
	}

	if !needsRefresh {
		if now.Sub(verifier.lastUnknownKidRefresh) < accessUnknownKidRefreshInterval &&
			len(verifier.refreshGate) == 0 {
			verifier.mutex.Unlock()

			return nil, errAccessJWTKeyMissing
		}
	}

	generation := verifier.generation
	verifier.mutex.Unlock()

	err := verifier.refreshKeys(ctx, generation, !needsRefresh)
	if err != nil {
		return nil, err
	}

	verifier.mutex.Lock()
	defer verifier.mutex.Unlock()

	key = verifier.keys[keyID]

	if key == nil {
		return nil, errAccessJWTKeyMissing
	}

	return key, nil
}

func (verifier *accessJWTVerifier) cacheNeedsRefresh(now time.Time) bool {
	return now.After(verifier.cacheTill) || len(verifier.keys) == 0
}

func (verifier *accessJWTVerifier) refreshKeys(ctx context.Context, generation uint64, unknownKey bool) error {
	select {
	case verifier.refreshGate <- struct{}{}:
		defer func() { <-verifier.refreshGate }()
	case <-ctx.Done():
		return ctx.Err()
	}

	verifier.mutex.Lock()

	if verifier.generation != generation {
		err := verifier.refreshErr
		verifier.mutex.Unlock()

		return err
	}

	if time.Now().Before(verifier.refreshRetryAfter) {
		verifier.mutex.Unlock()

		return errAccessJWKSRefreshBackoff
	}

	now := time.Now()
	refreshingWarmCache := !verifier.cacheNeedsRefresh(now)

	if verifier.unknownRefreshInCooldown(unknownKey, now) {
		verifier.mutex.Unlock()

		return nil
	}
	verifier.mutex.Unlock()

	keys, result := verifier.fetchKeys(ctx)

	if ctx.Err() != nil {
		return ctx.Err()
	}

	verifier.mutex.Lock()
	defer verifier.mutex.Unlock()

	verifier.generation++
	verifier.refreshErr = result

	if result != nil {
		verifier.refreshRetryAfter = time.Now().Add(accessJWKSFailureBackoff)

		return result
	}

	verifier.keys = keys
	verifier.cacheTill = time.Now().Add(accessKeyCacheTTL)
	verifier.refreshRetryAfter = time.Time{}

	if refreshingWarmCache {
		verifier.lastUnknownKidRefresh = time.Now()
	}

	return nil
}

func (verifier *accessJWTVerifier) unknownRefreshInCooldown(unknownKey bool, now time.Time) bool {
	return unknownKey && !verifier.cacheNeedsRefresh(now) &&
		now.Sub(verifier.lastUnknownKidRefresh) < accessUnknownKidRefreshInterval
}

func (verifier *accessJWTVerifier) fetchKeys(ctx context.Context) (map[string]*rsa.PublicKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet,
		verifier.issuer+"/cdn-cgi/access/certs", nil)
	if err != nil {
		return nil, err
	}

	response, err := verifier.client.Do(request)
	if err != nil {
		return nil, err
	}

	defer func() { _ = response.Body.Close() }()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: HTTP %d", errAccessJWKSStatus, response.StatusCode)
	}

	const maxJWKSSize = 1 << 20

	body, err := io.ReadAll(io.LimitReader(response.Body, maxJWKSSize+1))
	if err != nil {
		return nil, err
	}

	if len(body) > maxJWKSSize {
		return nil, errAccessJWKSSize
	}

	keys, err := decodeAccessJWKS(body)
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func decodeAccessJWKS(body []byte) (map[string]*rsa.PublicKey, error) {
	var set accessJWKS

	err := json.Unmarshal(body, &set)
	if err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey, len(set.Keys))

	for _, jwk := range set.Keys {
		if jwk.Type != "RSA" || jwk.Alg != "RS256" || jwk.KeyID == "" {
			continue
		}

		key, err := decodeAccessRSAKey(jwk.N, jwk.E)
		if err != nil {
			return nil, err
		}

		keys[jwk.KeyID] = key
	}

	if len(keys) == 0 {
		return nil, errAccessJWKSNoKeys
	}

	return keys, nil
}

func decodeAccessRSAKey(modulusText, exponentText string) (*rsa.PublicKey, error) {
	modulus, err := base64.RawURLEncoding.DecodeString(modulusText)
	if err != nil {
		return nil, fmt.Errorf("decode Cloudflare Access RSA modulus: %w", err)
	}

	exponentBytes, err := base64.RawURLEncoding.DecodeString(exponentText)
	if err != nil || len(exponentBytes) == 0 || len(exponentBytes) > 4 {
		return nil, errAccessJWKExponent
	}

	exponent := 0
	for _, value := range exponentBytes {
		exponent = exponent<<accessBitsPerByte | int(value)
	}

	if exponent < 3 || exponent%2 == 0 || len(modulus) < 256 {
		return nil, errAccessJWKKey
	}

	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponent}, nil
}
