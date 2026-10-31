// Package authorization is a placeholder for application authorization policies.
package authorization

import "net/http"

// New currently permits every authenticated request.
func New() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		// TODO: inspect authentication.ClaimsFromContext and enforce policies.
		return next
	}
}
