package api

import (
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
}

func NewAPI(
	address string,
	router *mux.Router,
	db *gorm.DB,
	providerGroup *messages.ProviderGroup,
) *API {
	return &API{router: router, db: db, providerGroup: providerGroup, address: address}
}

func (a *API) initRoutes() {
	a.router.HandleFunc("/on-push", a.handleOnPush)
}

func (a *API) Start() {
	a.initRoutes()

	http.ListenAndServe(a.address, a.router)
}
