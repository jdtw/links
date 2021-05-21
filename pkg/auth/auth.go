package auth

import (
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/lestrrat-go/jwx/jwa"
	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jws"
	"github.com/lestrrat-go/jwx/jwt"
)

const (
	keySub = "sub"
	issuer = "github.com/jdtw/links/pkg/auth"
)

// ReadKeyset reads a JSON-encoded keyset from a file.
func ReadKeyset(path string) (jwk.Set, error) {
	bs, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ks := jwk.NewSet()
	if err := json.Unmarshal(bs, ks); err != nil {
		return nil, err
	}
	return ks, nil
}

// WriteKeyset writes a JSON-encoded keyset to a file.
func WriteKeyset(ks jwk.Set, path string) error {
	bs, err := json.Marshal(ks)
	if err != nil {
		return err
	}
	return os.WriteFile(path, bs, fs.ModePerm)
}

// NewKey generates a new ed25519 keypair and returns the
// public key as a jwk.Key and the private key in PKCS8 format.
func NewKey(rand io.Reader, sub string) (pub jwk.Key, priv []byte, err error) {
	rawPub, rawPriv, err := ed25519.GenerateKey(rand)
	if err != nil {
		return
	}
	pub, err = jwk.New(rawPub)
	if err != nil {
		return
	}
	pub.Set(jwk.KeyIDKey, kid(rawPub))
	pub.Set(jwk.AlgorithmKey, jwa.EdDSA)
	pub.Set(jwk.KeyUsageKey, "sig")
	pub.Set(keySub, sub)

	priv, err = x509.MarshalPKCS8PrivateKey(rawPriv)
	return
}

type TokenOption func(jwt.Token)

// WithExpiry adds an expiration time to the JWT.
func WithExpiry(now time.Time, exp time.Duration) TokenOption {
	return func(t jwt.Token) {
		t.Set(jwt.IssuedAtKey, now)
		t.Set(jwt.NotBeforeKey, now)
		t.Set(jwt.ExpirationKey, now.Add(exp))
	}
}

// SignJWT signs a JWT for the given audience with the pkcs8 private key.
// Only Ed25519 private keys are supported.
func SignJWT(pkcs8 []byte, aud string, options ...TokenOption) ([]byte, error) {
	key, err := x509.ParsePKCS8PrivateKey(pkcs8)
	if err != nil {
		return nil, err
	}

	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("expected ed25519 private key, got %T", key)
	}

	hdrs := jws.NewHeaders()
	hdrs.Set(jws.KeyIDKey, kid(priv))

	t := jwt.New()
	t.Set(jwt.IssuerKey, issuer)
	t.Set(jwt.AudienceKey, aud)

	now := time.Now()
	t.Set(jwt.IssuedAtKey, now)
	t.Set(jwt.NotBeforeKey, now)
	t.Set(jwt.ExpirationKey, now.Add(time.Minute*10))

	for _, o := range options {
		o(t)
	}

	return jwt.Sign(t, jwa.EdDSA, priv, jwt.WithHeaders(hdrs))
}

// VerifyJWT verifies the token against the given keyset and audience.
// Returns the subject from the verification key.
func VerifyJWT(ks jwk.Set, s []byte, aud string) (string, error) {
	key, err := findKey(s, ks)
	if err != nil {
		return "", err
	}
	var rawKey interface{}
	if err := key.Raw(&rawKey); err != nil {
		return "", err
	}
	if _, err = jwt.Parse(s,
		jwt.WithVerify(
			jwa.SignatureAlgorithm(key.Algorithm()),
			rawKey,
		),
		jwt.WithValidate(true),
		jwt.WithAcceptableSkew(5*time.Second),
		jwt.WithIssuer(issuer),
		jwt.WithAudience(aud),
	); err != nil {
		return "", err
	}
	i, ok := key.Get(keySub)
	if !ok {
		return "", fmt.Errorf("key missing subject")
	}
	sub, ok := i.(string)
	if !ok {
		return "", fmt.Errorf("invalid subject type: %T", i)
	}
	return sub, nil
}

func findKey(s []byte, ks jwk.Set) (jwk.Key, error) {
	msg, err := jws.Parse(s)
	if err != nil {
		return nil, err
	}

	sigs := msg.Signatures()
	if n := len(sigs); n != 1 {
		return nil, fmt.Errorf("expected 1 signature, got %d", n)
	}

	hdrs := sigs[0].ProtectedHeaders()
	kid := hdrs.KeyID()
	if kid == "" {
		return nil, fmt.Errorf("signature header missing key ID")
	}

	key, ok := ks.LookupKeyID(kid)
	if !ok {
		return nil, fmt.Errorf("key %q not found", kid)
	}

	// We use the key algorithm to verify the signature, not
	// the algorithm from the header. Make sure it is present.
	if key.Algorithm() == "" {
		return nil, fmt.Errorf("key missing algorithm")
	}

	// Make sure the key usage is 'sig'
	usage, ok := key.Get(jwk.KeyUsageKey)
	if !ok {
		return nil, fmt.Errorf("key missing usage")
	}
	u, ok := usage.(string)
	if !ok {
		return nil, fmt.Errorf("invalid usage type: %T", usage)
	}
	if u != "sig" {
		return nil, fmt.Errorf("invalid usage %q", u)
	}

	return key, nil
}

func kid(key interface{}) string {
	var pub []byte
	switch k := key.(type) {
	case ed25519.PrivateKey:
		pub = []byte(k.Public().(ed25519.PublicKey))
	case ed25519.PublicKey:
		pub = []byte(k)
	default:
		log.Fatalf("unsupported key type: %T", k)
	}
	digest := sha256.Sum256(pub)
	return hex.EncodeToString(digest[:])
}

func ClientAudience(r *http.Request) string {
	return fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
}

func ServerAudience(r *http.Request) string {
	return fmt.Sprintf("%s %s%s", r.Method, r.Host, r.URL)
}
