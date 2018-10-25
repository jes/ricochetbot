package ricochetbot

import (
	"github.com/jes/go-ricochet/application"
	"github.com/jes/go-ricochet/channels"
	"log"
	"time"
)

type Peer struct {
	Onion string // empty string until they've authenticated
	rai   *application.ApplicationInstance
	Bot   *RicochetBot
}

// We always want bidirectional chat channels
func (peer *Peer) OpenInbound() {
	log.Println("OpenInbound() ChatChannel handler called...")
	outboundChatChannel := peer.rai.Connection.Channel("im.ricochet.chat", channels.Outbound)
	if outboundChatChannel == nil {
		peer.rai.Connection.Do(func() error {
			peer.rai.Connection.RequestOpenChannel("im.ricochet.chat",
				&channels.ChatChannel{
					Handler: peer,
				})
			return nil
		})
	}
}

func (peer *Peer) OpenedOutbound() {
	if peer.Bot.OnReadyToChat != nil {
		peer.Bot.OnReadyToChat(peer)
		peer.Bot.DedupPeers(peer)
	}
}

func (peer *Peer) ChatMessage(messageID uint32, when time.Time, message string) bool {
	log.Printf("ChatMessage(from: %v, %v", peer.rai.RemoteHostname, message)
	if peer.Bot.OnMessage != nil {
		peer.Bot.OnMessage(peer, message)
	}
	return true
}

func (peer *Peer) SendMessage(message string) {
	SendMessage(peer.rai, message)
}

func SendMessage(rai *application.ApplicationInstance, message string) {
	log.Printf("SendMessage(to: %v, %v)\n", rai.RemoteHostname, message)
	rai.Connection.Do(func() error {

		log.Printf("Finding Chat Channel")
		channel := rai.Connection.Channel("im.ricochet.chat", channels.Outbound)
		if channel != nil {
			log.Printf("Found Chat Channel")
			chatchannel, ok := channel.Handler.(*channels.ChatChannel)
			if ok {
				chatchannel.SendMessage(message)
			}
		} else {
			log.Printf("Could not find chat channel")
		}
		return nil
	})
}

func (peer *Peer) ChatMessageAck(messageID uint32, accepted bool) {

}

func (peer *Peer) Disconnect() {
	peer.rai.Connection.Conn.Close()
}
