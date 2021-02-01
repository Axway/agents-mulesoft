package anypoint

import (
	"github.com/Axway/agent-sdk/pkg/util/log"
)

// Auth represents the authentication information.
type Auth struct {
}

// GetToken todo
func (a *Auth) GetToken() {

}

func (a *Auth) authenticate() {
	log.Info("Logging into Mulesoft Anypoint Exchange")

}
