package api

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

func getAuthorization(r *http.Request) (string, error) {
	log.Println("Getting authorization token...")

	header := r.Header.Get("Authorization")
	log.Printf("Authorization: %s\n", header)

	token := strings.Split(header, " ")
	if len(token) != 2 || token[0] != "Bearer" {
		return "", errors.New("invalid token")
	}

	return token[1], nil
}
