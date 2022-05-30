package tokentest

import (
	"testing"

	"jdtw.dev/token"
)

func GenerateKey(t *testing.T, subject string) (*token.VerificationKeyset, *token.SigningKey) {
	t.Helper()
	keyset := token.NewVerificationKeyset()
	verifier, signer, err := token.GenerateKey(subject)
	if err != nil {
		t.Fatal(err)
	}
	if err := keyset.Add(verifier); err != nil {
		t.Fatal(err)
	}
	return keyset, signer
}
