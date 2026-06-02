// Package credentials manages sensitive project credentials via the OS keychain.
// Passwords are stored under service="beam", account=projectName.
// Falls back gracefully to plaintext if the keychain is unavailable.
package credentials

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const service = "beam"

var ErrNotFound = errors.New("credentials: not found")

// Set stores the password for the given project in the OS keychain.
func Set(projectName, password string) error {
	return keyring.Set(service, projectName, password)
}

// Get retrieves the stored password for the given project.
// Returns ErrNotFound if no entry exists.
func Get(projectName string) (string, error) {
	pass, err := keyring.Get(service, projectName)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return pass, err
}

// Delete removes the keychain entry for the given project.
// A missing entry is not treated as an error.
func Delete(projectName string) error {
	err := keyring.Delete(service, projectName)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
