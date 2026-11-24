package secrets

import (
	"errors"

	"filippo.io/age"
	"github.com/zalando/go-keyring"
)

const (
	KeyringService = "org.pureterm.app"
	KeyringUser    = "vault-identity"

	// legacyKeyringServices are the service names used under former program
	// names (LeanTerm, LinXTerm); entries found there are moved to KeyringService.
	legacyKeyringService = "org.linxterm.app"
)

var legacyKeyringServices = []string{"org.leanterm.app", legacyKeyringService}

var (
	ErrKeyringUnavailable = errors.New("system keyring is unavailable or locked")
	ErrKeyringNotFound    = errors.New("keyring entry not found")
)

type KeyringManager struct{}

func NewKeyringManager() *KeyringManager {
	return &KeyringManager{}
}

// GetIdentity retrieves the stored X25519 identity from the OS keyring.
func (km *KeyringManager) GetIdentity() (*age.X25519Identity, error) {
	val, err := keyring.Get(KeyringService, KeyringUser)
	if errors.Is(err, keyring.ErrNotFound) {
		for _, legacy := range legacyKeyringServices {
			if legacyVal, legacyErr := keyring.Get(legacy, KeyringUser); legacyErr == nil {
				val, err = legacyVal, nil
				if keyring.Set(KeyringService, KeyringUser, val) == nil {
					_ = keyring.Delete(legacy, KeyringUser)
				}
				break
			}
		}
	}
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return nil, ErrKeyringNotFound
		}
		return nil, fmtKeyringErr(err)
	}

	identity, err := age.ParseX25519Identity(val)
	if err != nil {
		return nil, err
	}
	return identity, nil
}

// SetIdentity stores the X25519 identity string in the OS keyring.
func (km *KeyringManager) SetIdentity(identity *age.X25519Identity) error {
	if identity == nil {
		return errors.New("nil identity")
	}
	err := keyring.Set(KeyringService, KeyringUser, identity.String())
	if err != nil {
		return fmtKeyringErr(err)
	}
	return nil
}

// DeleteIdentity removes the identity from the OS keyring.
func (km *KeyringManager) DeleteIdentity() error {
	err := keyring.Delete(KeyringService, KeyringUser)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return fmtKeyringErr(err)
	}
	return nil
}

// IsAvailable tests if the keyring can be used.
func (km *KeyringManager) IsAvailable() bool {
	// Try a test operation or probe
	_, err := keyring.Get(KeyringService, KeyringUser)
	if err == nil || errors.Is(err, keyring.ErrNotFound) {
		return true
	}
	return false
}

func fmtKeyringErr(err error) error {
	return errors.Join(ErrKeyringUnavailable, err)
}
