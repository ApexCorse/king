package api

import (
	"errors"
	"net/http"
	"strings"
)

func getAuthorization(r *http.Request) (string, error) {
	header := r.Header.Get("Authorization")

	token := strings.Split(header, " ")
	if len(token) != 2 || token[0] != "Bearer" {
		return "", errors.New("invalid token")
	}

	return token[1], nil
}
