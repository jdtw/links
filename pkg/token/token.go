package token

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	pb "github.com/jdtw/links/proto/token"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const header = "github.com/jdtw/links/pkg/token v0.1.0"

type SigningKey struct {
	key *pb.SigningKey
}

func NewSigningKey() (*SigningKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	keyID := sha256.Sum256([]byte(pub))
	return &SigningKey{&pb.SigningKey{
		Id:         hex.EncodeToString(keyID[:]),
		PrivateKey: []byte(priv)},
	}, nil
}

func UnmarshalSigningKey(serialized []byte) (*SigningKey, error) {
	key := &pb.SigningKey{}
	if err := proto.Unmarshal(serialized, key); err != nil {
		return nil, err
	}
	return &SigningKey{key}, nil
}

func (k *SigningKey) Public() []byte {
	priv := ed25519.PrivateKey(k.key.PrivateKey)
	return priv.Public().(ed25519.PublicKey)
}

func (k *SigningKey) ID() string {
	return k.key.Id
}

func (k *SigningKey) Marshal() ([]byte, error) {
	return proto.Marshal(k.key)
}

type TokenOption func(*pb.Token)

func WithRequestResource(r *http.Request) TokenOption {
	resource := fmt.Sprintf("%s %s%s", r.Method, r.URL.Host, r.URL.Path)
	return WithResource(resource)
}

func WithResource(resource string) TokenOption {
	return func(t *pb.Token) {
		t.Resource = resource
	}
}

// WithExpiry adds an expiration time to the token.
func WithExpiry(now time.Time, exp time.Duration) TokenOption {
	return func(t *pb.Token) {
		t.NotBefore = timestamppb.New(now)
		t.NotAfter = timestamppb.New(now.Add(exp))
	}
}

// Sign signs a token for the given resource. By default, the expiry time is one minute.
func (k *SigningKey) Sign(options ...TokenOption) (string, error) {
	token := &pb.Token{}
	WithExpiry(time.Now(), time.Minute)(token)
	for _, opt := range options {
		opt(token)
	}
	if token.Resource == "" {
		return "", fmt.Errorf("token missing required resource; use one of the With*Resource options to set it")
	}
	bytes, err := proto.Marshal(token)
	if err != nil {
		return "", err
	}

	priv := ed25519.PrivateKey(k.key.PrivateKey)
	// Append the header before signing to prevent any sort of cross-protocol tomfoolery.
	toSign := append([]byte(header), bytes...)
	sig, err := priv.Sign(rand.Reader, toSign, crypto.Hash(0))
	if err != nil {
		return "", err
	}
	bytes, err = proto.Marshal(&pb.SignedToken{
		KeyId:     k.key.Id,
		Signature: sig,
		Token:     bytes,
	})
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

type VerificationKeyset struct {
	keys *pb.VerificationKeyset
}

func NewVerificationKeyset() *VerificationKeyset {
	return &VerificationKeyset{&pb.VerificationKeyset{
		Keys: make(map[string]*pb.VerificationKey),
	}}
}

func UnmarshalVerificationKeyset(serialized []byte) (*VerificationKeyset, error) {
	keyset := &pb.VerificationKeyset{}
	if err := proto.Unmarshal(serialized, keyset); err != nil {
		return nil, err
	}
	return &VerificationKeyset{keyset}, nil
}

func (v *VerificationKeyset) AddKey(id string, subject string, publicKey []byte) error {
	if id == "" {
		return errors.New("missing ID")
	}
	if subject == "" {
		return errors.New("missing subject")
	}
	if got, want := len(publicKey), ed25519.PublicKeySize; got != want {
		return fmt.Errorf("invalid key size; got %d, want %d", got, want)
	}
	v.keys.Keys[id] = &pb.VerificationKey{
		Id:        id,
		Subject:   subject,
		PublicKey: publicKey,
	}
	return nil
}

func (v *VerificationKeyset) Marshal() ([]byte, error) {
	return proto.Marshal(v.keys)
}

type TokenCheck func(*pb.Token) error

func CheckExpiry(now time.Time) TokenCheck {
	return func(t *pb.Token) error {
		if t.NotBefore == nil || t.NotAfter == nil {
			return errors.New("token missing expiry")
		}
		if now.Before(t.NotBefore.AsTime()) {
			return errors.New("token not valid yet")
		}
		if now.After(t.NotAfter.AsTime()) {
			return errors.New("token expired")
		}
		return nil
	}
}

func CheckRequestResource(r *http.Request) TokenCheck {
	resource := fmt.Sprintf("%s %s%s", r.Method, r.Host, r.URL)
	return CheckResource(resource)
}

func CheckResource(resource string) TokenCheck {
	return func(t *pb.Token) error {
		if t.Resource != resource {
			return fmt.Errorf("got resource %q, want %q", t.Resource, resource)
		}
		return nil
	}
}

func (v *VerificationKeyset) Verify(token string, checks ...TokenCheck) (string, error) {
	// Unmarshal the signed token.
	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		return "", err
	}
	signed := &pb.SignedToken{}
	if err := proto.Unmarshal(decoded, signed); err != nil {
		return "", err
	}

	// Fetch the verification key.
	k := v.keys.Keys[signed.KeyId]
	if k == nil {
		return "", fmt.Errorf("key %q not found", signed.KeyId)
	}

	// Verify the signature.
	pub := ed25519.PublicKey(k.PublicKey)
	toVerify := append([]byte(header), signed.Token...)
	if !ed25519.Verify(pub, toVerify, signed.Signature) {
		return "", fmt.Errorf("invalid signature")
	}

	// The token is cryptographically valid. Check contents.
	t := &pb.Token{}
	if err := proto.Unmarshal(signed.Token, t); err != nil {
		return "", err
	}
	for _, check := range checks {
		if err := check(t); err != nil {
			return "", err
		}
	}
	return k.Subject, nil
}
