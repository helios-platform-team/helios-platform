package database

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

// GenerateSecurePassword generates a cryptographically secure random password.
func GenerateSecurePassword(length int) (string, error) {
	if length <= 0 {
		length = DefaultPasswordLength
	}

	password := make([]byte, length)
	charsetLen := big.NewInt(int64(len(PasswordCharset)))

	for i := 0; i < length; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		password[i] = PasswordCharset[idx.Int64()]
	}

	return string(password), nil
}

// GenerateSecureUsername generates a cryptographically secure random username.
func GenerateSecureUsername(length int) (string, error) {
	if length <= 0 {
		length = DefaultUsernameLength
	}

	username := make([]byte, length)
	lettersOnly := "abcdefghijklmnopqrstuvwxyz"
	lettersLen := big.NewInt(int64(len(lettersOnly)))

	idx, err := rand.Int(rand.Reader, lettersLen)
	if err != nil {
		return "", fmt.Errorf("failed to generate random index: %w", err)
	}
	username[0] = lettersOnly[idx.Int64()]

	charsetLen := big.NewInt(int64(len(UsernameCharset)))
	for i := 1; i < length; i++ {
		idx, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %w", err)
		}
		username[i] = UsernameCharset[idx.Int64()]
	}

	return string(username), nil
}

// GenerateCredentials generates a new set of database credentials.
func GenerateCredentials() (*DatabaseCredentials, error) {
	username, err := GenerateSecureUsername(DefaultUsernameLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate username: %w", err)
	}

	password, err := GenerateSecurePassword(DefaultPasswordLength)
	if err != nil {
		return nil, fmt.Errorf("failed to generate password: %w", err)
	}

	return &DatabaseCredentials{
		Username: username,
		Password: password,
	}, nil
}

// GenerateBase64Token generates a random base64-encoded token.
func GenerateBase64Token(byteLength int) (string, error) {
	if byteLength <= 0 {
		byteLength = 32
	}

	bytes := make([]byte, byteLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}
