package ricochetbot

import (
	"fmt"
	"github.com/jes/go-ricochet/application"
)

type RicochetBotContactHandler struct {
	rai *application.ApplicationInstance
	bot *RicochetBot
}

func (rbch *RicochetBotContactHandler) ContactRequest(hostname string, name string, message string) string {
	fmt.Println("contacthandler.go: ContactRequest(", name, ", ", message, ")")
	accept := rbch.bot.OnContactRequest(rbch.bot.LookupPeerByHostname(hostname), name, message)
	if accept {
		return "Accepted"
	} else {
		return "Pending"
	}
}
func (rbch *RicochetBotContactHandler) ContactRequestRejected() {
}
func (rbch *RicochetBotContactHandler) ContactRequestAccepted() {
}
func (rbch *RicochetBotContactHandler) ContactRequestError() {
}
