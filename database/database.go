package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gandalfmagic/encryption"
)

var (
	ErrSessionsMismatch = errors.New("the old and new session do not match")
)

type SessionImpl interface {
	CloseConnection(ctx context.Context) error
	Add(context.Context, SessionData) (string, error)
	Delete(context.Context, string) error
	Get(context.Context, string) (SessionData, error)
	Update(context.Context, string, SessionData) error
	Purge(ctx context.Context) error
}

type SessionData struct {
	Subject      string
	AccessToken  string
	RefreshToken string
	IDToken      string
	ExpiresAt    time.Time
}

func (d SessionData) IsExpired() bool {
	now := time.Now()
	return now.After(d.ExpiresAt)
}

func encryptArgs(cipher encryption.HexCipher, s SessionData) (SessionData, error) {
	if cipher == nil {
		return s, nil
	}

	encAccessToken, err := cipher.EncryptToHexString([]byte(s.AccessToken))
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot encrypt access token: %w", err)
	}

	encRefreshToken, err := cipher.EncryptToHexString([]byte(s.RefreshToken))
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot encrypt refresh token: %w", err)
	}

	encIDToken, err := cipher.EncryptToHexString([]byte(s.IDToken))
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot encrypt id token: %w", err)
	}

	return SessionData{
		Subject:      s.Subject,
		AccessToken:  encAccessToken,
		RefreshToken: encRefreshToken,
		IDToken:      encIDToken,
		ExpiresAt:    s.ExpiresAt,
	}, nil
}

func decryptArgs(cipher encryption.HexCipher, s SessionData) (SessionData, error) {
	if cipher == nil {
		return s, nil
	}

	var decAccessToken, decRefreshToken, decIDToken []byte
	var err error

	decAccessToken, err = cipher.DecryptFromHexString(s.AccessToken)
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot decrypt access token: %w", err)
	}

	decRefreshToken, err = cipher.DecryptFromHexString(s.RefreshToken)
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot decrypt refresh token: %w", err)
	}

	decIDToken, err = cipher.DecryptFromHexString(s.IDToken)
	if err != nil {
		return SessionData{}, fmt.Errorf("cannot decrypt id token: %w", err)
	}

	return SessionData{
		Subject:      s.Subject,
		AccessToken:  string(decAccessToken),
		RefreshToken: string(decRefreshToken),
		IDToken:      string(decIDToken),
		ExpiresAt:    s.ExpiresAt,
	}, nil
}
