package api

import (
	"log"
	"net/http"

	"github.com/Formula-SAE/discord/src/internal/messages"
	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type API struct {
	address string
	router  *mux.Router
	db      *gorm.DB

	providerGroup *messages.ProviderGroup

	masterToken string
}

func NewAPI(
	address string,
	router *mux.Router,
	db *gorm.DB,
	providerGroup *messages.ProviderGroup,
	masterToken string,
) *API {
	return &API{
		router:        router,
		db:            db,
		providerGroup: providerGroup,
		address:       address,
		masterToken:   masterToken,
	}
}

func (a *API) initRoutes() {
	log.Println("Initializing routes...")
	a.router.HandleFunc("/on-push", a.handleOnPush).Methods("POST")
	a.router.HandleFunc("/token", a.addTokenToDB).Methods("POST")
}

func (a *API) Start() {
	a.initRoutes()

	http.ListenAndServe(a.address, a.router)
}
