package jwtUtils

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/golang-jwt/jwt/v5/request"
)

const (
	// issuer is the issuer of the jwt token.
	Issuer = "userservice"
	// Signing key section. For now, this is only used for signing, not for verifying since we only
	// have 1 version. But it will be used to maintain backward compatibility if we change the signing mechanism.
	KeyID = "v1"
	// AccessTokenAudienceName is the audience name of the access token.
	AccessTokenAudienceName = "user.access-token"
	AccessTokenDuration     = 24 * time.Hour
)

var (
	ErrJWTValidate       = errors.New("failed to validate jwt token")
	ErrJWTUserIDNotFound = errors.New("user id not found/malformed  in token")
)

type ClaimsMessage struct {
	Name string `json:"name"`
	jwt.RegisteredClaims
}

func ValidateAccessToken(req *http.Request, secret string) (*jwt.Token, error) {
	extractor := request.AuthorizationHeaderExtractor
	keyFunc := func(t *jwt.Token) (any, error) {
		if kid, ok := t.Header["kid"].(string); ok {
			if kid == "v1" {
				return []byte(secret), nil
			}
		}
		return nil, fmt.Errorf("unexpected access token kid=%v", t.Header["kid"])
	}

	// It is heavily
	// encouraged to use this option in order to prevent attacks such as
	// https://auth0.com/blog/critical-vulnerabilities-in-json-web-token-libraries/.
	parserOption := jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()})
	parser := jwt.NewParser(parserOption)

	var options []request.ParseFromRequestOption
	options = append(options, request.WithClaims(ClaimsMessage{}))
	options = append(options, request.WithParser(parser))

	token, err := request.ParseFromRequest(req, extractor, keyFunc, options...)
	if err != nil {
		return nil, err
	}
	return token, nil
}

func GetUserIDFromToken(token *jwt.Token) (int64, error) {
	userID, err := token.Claims.GetSubject()
	if err != nil {
		return 0, err
	}

	parsedUserID, err := strconv.ParseInt(userID, 10, 64)
	if err != nil {
		return 0, err
	}
	return parsedUserID, nil

}

// GenerateAccessToken generates an access token.
func GenerateAccessToken(username string, userID int64, secret []byte) (string, error) {
	return generateToken(username, userID, AccessTokenAudienceName, secret)
}

// generateToken generates a jwt token.
func generateToken(username string, userID int64, audience string, secret []byte) (string, error) {
	now := time.Now()
	registeredClaims := jwt.RegisteredClaims{
		Issuer:   Issuer,
		Audience: jwt.ClaimStrings{audience},
		IssuedAt: jwt.NewNumericDate(now),
		Subject:  fmt.Sprint(userID),
	}

	registeredClaims.ExpiresAt = jwt.NewNumericDate(now.Add(AccessTokenDuration))
	// Declare the token with the HS256 algorithm used for signing, and the claims.
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &ClaimsMessage{
		Name:             username,
		RegisteredClaims: registeredClaims,
	})
	token.Header["kid"] = KeyID

	// Create the JWT string.
	tokenString, err := token.SignedString(secret)
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
