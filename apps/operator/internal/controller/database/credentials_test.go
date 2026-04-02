package database

import (
	"encoding/base64"
	"testing"
)

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"short password", 8},
		{"long password", 64},
		{"standard length", DefaultPasswordLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateSecurePassword(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecurePassword failed: %v", err)
			}

			expectedLen := tt.length
			if expectedLen <= 0 {
				expectedLen = DefaultPasswordLength
			}

			if len(password) != expectedLen {
				t.Errorf("Expected password length %d, got %d", expectedLen, len(password))
			}

			for _, c := range password {
				found := false
				for _, allowed := range PasswordCharset {
					if c == allowed {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Invalid character %c in password", c)
				}
			}
		})
	}
}

func TestGenerateSecureUsername(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"short username", 8},
		{"long username", 32},
		{"standard length", DefaultUsernameLength},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			username, err := GenerateSecureUsername(tt.length)
			if err != nil {
				t.Fatalf("GenerateSecureUsername failed: %v", err)
			}

			expectedLen := tt.length
			if expectedLen <= 0 {
				expectedLen = DefaultUsernameLength
			}

			if len(username) != expectedLen {
				t.Errorf("Expected username length %d, got %d", expectedLen, len(username))
			}

			firstChar := username[0]
			if firstChar < 'a' || firstChar > 'z' {
				t.Errorf("First character %c must be a lowercase letter", firstChar)
			}

			for _, c := range username {
				found := false
				for _, allowed := range UsernameCharset {
					if c == allowed {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Invalid character %c in username", c)
				}
			}
		})
	}
}

func TestGenerateCredentials(t *testing.T) {
	creds, err := GenerateCredentials()
	if err != nil {
		t.Fatalf("GenerateCredentials failed: %v", err)
	}

	if creds.Username == "" {
		t.Error("Username should not be empty")
	}

	if creds.Password == "" {
		t.Error("Password should not be empty")
	}

	if len(creds.Username) != DefaultUsernameLength {
		t.Errorf("Expected username length %d, got %d", DefaultUsernameLength, len(creds.Username))
	}

	if len(creds.Password) != DefaultPasswordLength {
		t.Errorf("Expected password length %d, got %d", DefaultPasswordLength, len(creds.Password))
	}
}

func TestGenerateCredentialsUniqueness(t *testing.T) {
	credentials := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		creds, err := GenerateCredentials()
		if err != nil {
			t.Fatalf("GenerateCredentials failed on iteration %d: %v", i, err)
		}

		key := creds.Username + ":" + creds.Password
		if credentials[key] {
			t.Errorf("Duplicate credentials generated on iteration %d", i)
		}
		credentials[key] = true
	}
}

func TestGenerateBase64Token(t *testing.T) {
	tests := []struct {
		name       string
		byteLength int
	}{
		{"default", 0},
		{"16 bytes", 16},
		{"32 bytes", 32},
		{"64 bytes", 64},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := GenerateBase64Token(tt.byteLength)
			if err != nil {
				t.Fatalf("GenerateBase64Token failed: %v", err)
			}

			if token == "" {
				t.Error("Token should not be empty")
			}

			expectedLen := tt.byteLength
			if expectedLen <= 0 {
				expectedLen = 32
			}
			expectedBase64Len := ((expectedLen + 2) / 3) * 4
			if len(token) != expectedBase64Len {
				t.Errorf("Expected base64 length %d, got %d", expectedBase64Len, len(token))
			}

			decoded, decodeErr := base64.StdEncoding.DecodeString(token)
			if decodeErr != nil {
				t.Fatalf("Token is not valid base64: %v", decodeErr)
			}
			if len(decoded) != expectedLen {
				t.Errorf("Expected decoded token length %d, got %d", expectedLen, len(decoded))
			}
		})
	}
}
