package sessions

import (
	"errors"
	"fmt"
	"time"

	"github.com/gandalfmagic/go-token-handler/database"
)

type KeyPair struct {
	Authentication string
	Encryption     string
}

type Configuration struct {
	CookieName     string
	CookieDomain   string
	NewKeyPair     KeyPair
	OldKeyPair     KeyPair
	LoginTimeout   time.Duration
	SessionTimeout time.Duration
	SessionImpl    database.SessionImpl
}

var (
	ErrWrongAuthSecretSize = errors.New("the session authentication key should have a size of 32 or 64 bytes")
	ErrWrongEncSecretSize  = errors.New("the session encryption key should have a size of 16, 24 or 32 bytes")
)

func (mc Configuration) keyPairsAsSlice() ([][]byte, error) {
	if mc.NewKeyPair.Authentication == "" {
		return [][]byte{}, nil
	}

	slice, err := mc.appendSlice([][]byte{}, parseCurrent)
	if err != nil {
		return nil, err
	}

	if mc.OldKeyPair.Authentication == "" {
		return slice, nil
	}

	return mc.appendSlice([][]byte{}, parseOld)
}

func (mc Configuration) appendSlice(slice [][]byte, pt parseType) ([][]byte, error) {
	var kp KeyPair

	switch pt {
	case parseCurrent:
		kp = mc.NewKeyPair
	case parseOld:
		kp = mc.OldKeyPair
	}

	authKey := []byte(kp.Authentication)

	kLen := len(authKey)
	if kLen != 32 && kLen != 64 {
		return nil, fmt.Errorf("%w: %s authentication key", ErrWrongAuthSecretSize, pt)
	}

	slice = append(slice, authKey)

	var encKey []byte
	if kp.Encryption != "" {
		encKey = []byte(kp.Encryption)
	}

	kLen = len(encKey)
	if kLen != 0 && kLen != 16 && kLen != 32 && kLen != 64 {
		return nil, fmt.Errorf("%w: %s encryption key", ErrWrongEncSecretSize, pt)
	}

	return append(slice, encKey), nil
}
