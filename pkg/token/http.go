package token

import (
	"fmt"
	"net/http"
	"strings"
)

// Our custom authorization scheme for the header.
const scheme = "ProtoEd25519 "

// AuthorizeRequest signs a token for the given HTTP request and adds it to the Authorization header.
func (s *SigningKey) AuthorizeRequest(req *http.Request, options ...TokenOption) error {
	token, err := s.Sign(append(options, withRequestResource(req))...)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", scheme+token)
	return nil
}

// AuthorizeRequest verifies the token in the Authorization header of the given HTTP request.
func (v *VerificationKeyset) AuthorizeRequest(r *http.Request, checks ...TokenCheck) (string, error) {
	authz := r.Header["Authorization"]
	if len(authz) != 1 {
		return "", fmt.Errorf("expected 1 authorization header, got %d", len(authz))
	}
	if !strings.HasPrefix(authz[0], scheme) {
		return "", fmt.Errorf("authorization header %q missing %q prefix", authz[0], scheme)
	}
	t := strings.TrimPrefix(authz[0], scheme)
	return v.Verify(t, append(checks, checkRequestResource(r))...)
}

func withRequestResource(r *http.Request) TokenOption {
	resource := fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
	return WithResource(resource)
}

func checkRequestResource(r *http.Request) TokenCheck {
	resource := fmt.Sprintf("%s %s%s", r.Method, r.Host, r.URL)
	return CheckResource(resource)
}
