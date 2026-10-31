// Package authentication validates bearer JWTs and exposes their claims.
package authentication

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// headerAuthorization names the request header containing bearer credentials.
	headerAuthorization = "Authorization"
	// headerWWWAuthenticate names the response header advertising the authentication scheme.
	headerWWWAuthenticate = "WWW-Authenticate"
	// bearerScheme is the authentication scheme used for JWT credentials.
	bearerScheme = "Bearer"
)

var (
	// ErrInvalidConfiguration indicates that the JWT key resolver or allowed signing algorithms are missing.
	ErrInvalidConfiguration = errors.New("JWT key resolver and allowed algorithms are required")
)

// claimsKey isolates authenticated claims from other context values.
type claimsKey struct{}

// ClaimsFromContext returns all claims from a successfully authenticated JWT.
func ClaimsFromContext(ctx context.Context) (jwt.MapClaims, bool) {
	var (
		claims, ok = ctx.Value(claimsKey{}).(jwt.MapClaims)
	)

	return claims, ok
}

// New accepts a verification-key resolver and an explicit signing algorithm allowlist.
// Additional options can require an issuer, audience, or expiration claim.
func New(keyFunc jwt.Keyfunc, algorithms []string, options ...jwt.ParserOption) (func(http.Handler) http.Handler, error) {
	var (
		allowedMethods = append([]string(nil), algorithms...)
		parserOptions  = append([]jwt.ParserOption(nil), options...)
		parser         *jwt.Parser
	)

	if keyFunc == nil || len(algorithms) == 0 {
		return nil, ErrInvalidConfiguration
	}

	parserOptions = append(parserOptions, jwt.WithValidMethods(allowedMethods))
	parser = jwt.NewParser(parserOptions...)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var (
				header = strings.Fields(r.Header.Get(headerAuthorization))
				claims = jwt.MapClaims{}
				token  *jwt.Token
				err    error
			)

			if len(r.Header.Values(headerAuthorization)) != 1 || len(header) != 2 || !strings.EqualFold(header[0], bearerScheme) {
				unauthorized(w)
				return
			}

			token, err = parser.ParseWithClaims(header[1], claims, keyFunc)
			if err != nil || !token.Valid {
				unauthorized(w)
				return
			}

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		})
	}, nil
}

// unauthorized advertises bearer authentication and returns HTTP 401.
func unauthorized(w http.ResponseWriter) {
	w.Header().Set(headerWWWAuthenticate, bearerScheme)
	http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
}
