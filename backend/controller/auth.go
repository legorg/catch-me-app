package controller

import (
	"net/http"
	"strings"

	"catch-me/backend/game"
)

// requirePlayerToken reads the bearer token. On failure it writes the error
// response and returns false; the service checks whether the token is known.
func requirePlayerToken(w http.ResponseWriter, r *http.Request) (game.PlayerToken, bool) {
	token, err := playerToken(r)
	if err != nil {
		writeGameError(w, err, "invalid token")
		return game.PlayerToken{}, false
	}
	return token, true
}

func playerToken(r *http.Request) (game.PlayerToken, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		return game.PlayerToken{}, game.ErrUnauthorized
	}
	return game.ParsePlayerToken(strings.TrimSpace(token))
}
