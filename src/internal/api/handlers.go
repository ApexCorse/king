package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/Formula-SAE/discord/src/internal/messages"
)

type PushRequestBody struct {
	Content   string `json:"content"`
	Providers []struct {
		Provider string `json:"provider"`
		Channel  string `json:"channel"`
	} `json:"providers"`
}

func (a *API) handleOnPush(w http.ResponseWriter, r *http.Request) {
	authorization := r.Header.Get("Authorization")

	token := strings.Split(authorization, " ")
	if token[0] != "Bearer" || len(token) != 2 {
		err := errors.New("invalid token")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !a.authorizeToken(token[1]) {
		err := errors.New("token not authorized")
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	body := &PushRequestBody{}
	err := json.NewDecoder(r.Body).Decode(body)
	if err != nil {
		err = errors.New("invalid input")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Content == "" || len(body.Providers) == 0 {
		err = errors.New("empty message or no providers")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	messageConfigs := make([]messages.MessageConfig, len(body.Providers))
	for _, p := range body.Providers {
		config := messages.MessageConfig{
			Provider: p.Provider,
			Text:     body.Content,
			Channel:  p.Channel,
		}

		messageConfigs = append(messageConfigs, config)
	}

	a.providerGroup.SendMessage(messageConfigs...)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
