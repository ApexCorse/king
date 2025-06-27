package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/Formula-SAE/discord/src/internal/messages"
)

type PushRequestBody struct {
	Content   string `json:"content"`
	Providers []struct {
		Provider string `json:"provider"`
		Channel  string `json:"channel"`
	} `json:"providers"`
}

type AddTokenRequestBody struct {
	Token string `json:"token"`
}

func (a *API) handleOnPush(w http.ResponseWriter, r *http.Request) {
	token, err := getAuthorization(r)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if !a.authorizeToken(token) {
		err := errors.New("token not authorized")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	body := &PushRequestBody{}
	err = json.NewDecoder(r.Body).Decode(body)
	log.Printf("Request: %+v\n", body)
	if err != nil {
		err = errors.New("invalid input")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if body.Content == "" || len(body.Providers) == 0 {
		err = errors.New("empty message or no providers")
		log.Println(err.Error())
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

	log.Printf("Message configurations: %+v\n", messageConfigs)

	a.providerGroup.SendMessage(messageConfigs...)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (a *API) addTokenToDB(w http.ResponseWriter, r *http.Request) {
	token, err := getAuthorization(r)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	if token != a.masterToken {
		err = errors.New("invalid token")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	body := &AddTokenRequestBody{}
	err = json.NewDecoder(r.Body).Decode(body)
	log.Printf("Request: %+v\n", body)

	if err != nil || body.Token == "" {
		err = errors.New("bad request")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	savedToken := &Token{Token: body.Token}
	result := a.db.Create(savedToken)

	if result.Error != nil {
		err = errors.New("operation failed")
		log.Println(err.Error())
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}
	log.Printf("Created token in DB: %s\n", savedToken.Token)

	response := map[string]string{
		"message": "Token created",
	}
	bytes, err := json.Marshal(response)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(bytes)
}
