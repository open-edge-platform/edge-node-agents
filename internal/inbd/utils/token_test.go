package utils

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIsTokenExpired_ValidToken(t *testing.T) {
	// Create a token with a future expiration time
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(10 * time.Minute).Unix(), // Expires in 10 minutes
	})
	tokenString, _ := token.SignedString([]byte("secret"))

	// Call the function
	isExpired, err := IsTokenExpired(tokenString)

	// Assert the token is not expired
	assert.NoError(t, err)
	assert.False(t, isExpired)
}

func TestIsTokenExpired_ExpiredToken(t *testing.T) {
	// Create a token with a past expiration time
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(-10 * time.Minute).Unix(), // Expired 10 minutes ago
	})
	tokenString, _ := token.SignedString([]byte("secret"))

	// Call the function
	isExpired, err := IsTokenExpired(tokenString)

	// Assert the token is expired
	assert.NoError(t, err)
	assert.True(t, isExpired)
}

func TestIsTokenExpired_NoExpClaim(t *testing.T) {
	// Create a token without an "exp" claim
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{})
	tokenString, _ := token.SignedString([]byte("secret"))

	// Call the function
	isExpired, err := IsTokenExpired(tokenString)

	// Assert an error is returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token does not have a valid 'exp' claim")
	assert.False(t, isExpired)
}

func TestIsTokenExpired_InvalidToken(t *testing.T) {
	// Create an invalid token string
	tokenString := "invalid.token.string"

	// Call the function
	isExpired, err := IsTokenExpired(tokenString)

	// Assert an error is returned
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error parsing token")
	assert.False(t, isExpired)
}

const jwtTokenPath = "/etc/intel_edge_node/tokens/release-service/access_token"

func TestReadJWTToken_EmptyToken(t *testing.T) {
	fs := afero.NewMemMapFs()
	tokenPath := "/tmp/token"
	// Create an empty token file
	err := afero.WriteFile(fs, tokenPath, []byte(""), 0644)
	assert.NoError(t, err)

	// Call ReadJWTToken with a dummy isTokenExpiredFunc
	token, err := ReadJWTToken(fs, tokenPath, func(string) (bool, error) {
		return false, nil
	})

	assert.NoError(t, err)
	assert.Equal(t, "", token)
}

func TestReadJWTToken(t *testing.T) {
	fs := afero.NewMemMapFs()

	mockIsTokenExpired := func(token string) (bool, error) {
		if token == "expired-token" {
			return true, nil
		}
		if token == "valid-token" {
			return false, nil
		}
		return false, fmt.Errorf("unexpected token")
	}

	t.Run("successful read with valid token", func(t *testing.T) {
		err := afero.WriteFile(fs, jwtTokenPath, []byte("valid-token"), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		token, err := ReadJWTToken(fs, jwtTokenPath, mockIsTokenExpired)
		assert.NoError(t, err)
		assert.Equal(t, "valid-token", token)
	})

	t.Run("read expired token", func(t *testing.T) {
		err := afero.WriteFile(fs, jwtTokenPath, []byte("expired-token"), 0644)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		token, err := ReadJWTToken(fs, jwtTokenPath, mockIsTokenExpired)
		assert.Error(t, err)
		assert.Equal(t, "", token)
		assert.Contains(t, err.Error(), "token is expired")
	})

	t.Run("file not found", func(t *testing.T) {
		err := fs.Remove(jwtTokenPath)
		if err != nil {
			t.Logf("Warning: failed to remove file: %v", err)
		}
		token, err := ReadJWTToken(fs, jwtTokenPath, mockIsTokenExpired)
		assert.Error(t, err)
		assert.Equal(t, "", token)
		assert.True(t, os.IsNotExist(err))
	})
}
