package token

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
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
