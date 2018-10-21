package ricochetbot

import (
	"fmt"
	"github.com/jes/go-ricochet/application"
)

type RicochetBotContactHandler struct {
	rai   *application.ApplicationInstance
	bot   *RicochetBot
	Onion string
}

func (rbch *RicochetBotContactHandler) ContactRequest(hostname string, name string, message string) string {
	rbch.Onion = hostname
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
	// XXX: call the handler for when an inbound channel is opened, this is a bodge so that the bot can
	// open an outbound channel immediately if it wants to; having an outbound channel always open makes
	// it possible to send messages to a peer without having to first remember to open the outbound channel
	go rbch.bot.LookupPeerByHostname(rbch.Onion).OpenInbound()
}
func (rbch *RicochetBotContactHandler) ContactRequestError() {
}
