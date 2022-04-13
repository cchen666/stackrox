package validator

import (
	"fmt"

	"github.com/pkg/errors"
	licenseproto "github.com/stackrox/stackrox/generated/shared/license"
	"github.com/stackrox/stackrox/pkg/cryptoutils"
	licensePkg "github.com/stackrox/stackrox/pkg/license"
	"github.com/stackrox/stackrox/pkg/sync"
)

type signingKey struct {
	verifier     cryptoutils.SignatureVerifier
	restrictions *SigningKeyRestrictions
}

func newValidator() *validator {
	return &validator{
		verifiersByKeyID: make(map[string]*signingKey),
	}
}

type validator struct {
	mutex            sync.RWMutex
	verifiersByKeyID map[string]*signingKey
}

func (v *validator) RegisterSigningKey(algo string, raw []byte, restrictions *SigningKeyRestrictions) error {
	keyID := licensePkg.SigningKeyFingerprint(raw)

	verifierCreator := signatureVerifierByName[algo]
	if verifierCreator == nil {
		return fmt.Errorf("invalid signature algorithm %q", algo)
	}

	verifier, err := verifierCreator(raw)
	if err != nil {
		return errors.Wrap(err, "could not create signature verifier from public key data")
	}

	v.mutex.Lock()
	defer v.mutex.Unlock()
	if _, ok := v.verifiersByKeyID[keyID]; ok {
		return fmt.Errorf("could not register key with id %q: already have a key with that id", keyID)
	}

	v.verifiersByKeyID[keyID] = &signingKey{
		verifier:     verifier,
		restrictions: restrictions,
	}

	return nil
}

func (v *validator) getSigningKey(keyID string) *signingKey {
	v.mutex.RLock()
	defer v.mutex.RUnlock()

	return v.verifiersByKeyID[keyID]
}

func (v *validator) ValidateLicenseKey(licenseKey string) (*licenseproto.License, error) {
	licenseBytes, sig, err := licensePkg.ParseLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.Wrap(err, "parsing license key")
	}

	license, err := licensePkg.UnmarshalLicense(licenseBytes)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling license")
	}

	if err := CheckLicenseIsWellFormed(license); err != nil {
		return nil, errors.Wrap(err, "malformed license")
	}

	signingKeyID := license.GetMetadata().GetSigningKeyId()
	signingKey := v.getSigningKey(signingKeyID)

	if signingKey == nil {
		return nil, errors.Errorf("could not validate license: invalid signing key ID %q", signingKeyID)
	}

	if err := signingKey.verifier.Verify(licenseBytes, sig); err != nil {
		return nil, errors.Wrap(err, "verifying license signature")
	}

	if signingKey.restrictions != nil {
		if err := signingKey.restrictions.Check(license.GetRestrictions()); err != nil {
			return nil, errors.Wrap(err, "license violated restrictions for signing key")
		}
	}

	return license, nil
}
