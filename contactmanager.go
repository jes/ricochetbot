package ricochetbot

import (
	"crypto/rsa"
    "fmt"
)

// RicochetBotContactManager implements the contact manager interface an presumes
// all connections are allowed.
type RicochetBotContactManager struct {
    bot *RicochetBot
}

// LookupContact returns that a contact is known and allowed to communicate for all cases.
func (rbcm *RicochetBotContactManager) LookupContact(hostname string, publicKey rsa.PublicKey) (allowed, known bool) {
    status := true
    if rbcm.bot.OnContactRequest != nil {
        status = rbcm.bot.OnNewPeer(rbcm.bot.LookupPeerByHostname(hostname))
    }
	return status, status
}

func (rbcm *RicochetBotContactManager) ContactRequest(hostname string, name string, message string) string {
    accept := rbcm.bot.OnContactRequest(rbcm.bot.LookupPeerByHostname(hostname), name, message)
    fmt.Println("contactmanager.go: ContactRequest(", name, ", ", message, ")")
    if accept {
        return "Accepted"
    } else {
        return "Rejected"
    }
}
