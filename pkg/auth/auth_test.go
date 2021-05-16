package auth

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

// The subject for the keys in this test
const subject = "jdtw"

func TestInvalidSign(t *testing.T) {
	tests := []struct {
		desc  string
		pkcs8 func(t *testing.T) []byte
	}{
		{
			desc: "invalid pkcs8",
			pkcs8: func(t *testing.T) []byte {
				return []byte("invalid")
			},
		},
		{
			desc: "invalid key type",
			pkcs8: func(t *testing.T) []byte {
				t.Helper()
				priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
				if err != nil {
					t.Fatalf("ecdsa.GenerateKey failed: %v", err)
				}
				bytes, err := x509.MarshalPKCS8PrivateKey(priv)
				if err != nil {
					t.Fatalf("x509.MarshalPKCS8PrivateKey failed: %v", err)
				}
				return bytes
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			if _, err := SignJWT(tc.pkcs8(t), ""); err == nil { // if NO error
				t.Fatal("SignJWT succeeded, expected err")
			}
		})
	}
}

func TestSignVerify(t *testing.T) {
	const aud = "aud"
	ks, priv := newKeyset(t)
	signed, err := SignJWT(priv, aud)
	if err != nil {
		t.Fatalf("SignJWT failed: %v", err)
	}
	got, err := VerifyJWT(ks, signed, aud)
	if err != nil {
		t.Fatalf("VerifyJWT failed: %v", err)
	}
	if subject != got {
		t.Fatalf("expected subject %q, got %q", subject, got)
	}
	// Verifying the same token with an empty keyset should fail
	if _, err := VerifyJWT(jwk.NewSet(), signed, aud); err == nil { // if NO error
		t.Fatal("expected verify with empty keyset to fail")
	}
}

func TestInvalidTokens(t *testing.T) {
	tests := []struct {
		desc      string
		signAud   string
		signOpts  []TokenOption
		verifyAud string
	}{
		{
			desc: "wrong issuer",
			signOpts: []TokenOption{func(t jwt.Token) {
				t.Set(jwt.IssuerKey, "wrong-issuer")
			}},
		},
		{
			desc:     "expired token",
			signOpts: []TokenOption{WithExpiry(time.Time{}, time.Second)},
		},
		{
			desc:      "wrong audience",
			signAud:   "right-aud",
			verifyAud: "wrong-aud",
		},
	}

	ks, priv := newKeyset(t)
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			signed, err := SignJWT(priv, tc.signAud, tc.signOpts...)
			if err != nil {
				t.Fatalf("SignJwt(%v) failed: %v", tc.signAud, err)
			}
			if _, err := VerifyJWT(ks, signed, tc.verifyAud); err == nil { // if NO error
				t.Fatal("VerifyJWT succeeded, expected err")
			}
		})
	}
}

func newKeyset(t *testing.T) (jwk.Set, []byte) {
	t.Helper()
	pub, priv, err := NewKey(nil, subject)
	if err != nil {
		t.Fatalf("NewKey(%q) failed: %v", subject, err)
	}
	ks := jwk.NewSet()
	ks.Add(pub)
	return ks, priv
}
