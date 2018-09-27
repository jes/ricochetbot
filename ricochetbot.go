package ricochetbot

import (
	"crypto/rsa"
	"fmt"
	"github.com/jes/go-ricochet/application"
	"github.com/jes/go-ricochet/channels"
	"log"
	"sync"
)

type RicochetBot struct {
	PrivateKey *rsa.PrivateKey
	Peers      []*Peer
	peerLock   sync.Mutex

	OnConnect        func(*Peer)
	OnNewPeer        func(*Peer) bool
	OnReadyToChat    func(*Peer)
	OnMessage        func(*Peer, string)
	OnContactRequest func(*Peer, string, string) bool
	OnDisconnect     func(*Peer)
}

func (bot *RicochetBot) Connect(onion string) {
	fmt.Println("Connect to ", onion)
}

func (bot *RicochetBot) DeletePeer(peer *Peer) {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	for i, p := range bot.Peers {
		if p == peer {
			bot.Peers[i] = bot.Peers[len(bot.Peers)-1]
			bot.Peers = bot.Peers[:len(bot.Peers)-1]
			return
		}
	}
}

// XXX: what if we have 2 peers with the same hostname?
func (bot *RicochetBot) LookupPeerByHostname(onion string) *Peer {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	for _, peer := range bot.Peers {
		if peer.Onion == onion {
			return peer
		}
	}
	fmt.Println("LookupPeerByHostname: ", onion, " FAILED to find peer")
	return nil
}

func (bot *RicochetBot) Run() {
	af := application.ApplicationInstanceFactory{}
	af.Init()

	af.AddHandler("im.ricochet.contact.request", func(rai *application.ApplicationInstance) func() channels.Handler {
		return func() channels.Handler {
			contact := new(channels.ContactRequestChannel)
			ch := new(RicochetBotContactHandler)
			ch.bot = bot
			ch.rai = rai
			contact.Handler = new(RicochetBotContactHandler)
			return contact
		}
	})

	af.AddHandler("im.ricochet.chat", func(rai *application.ApplicationInstance) func() channels.Handler {
		return func() channels.Handler {
			chat := new(channels.ChatChannel)
			chat.Handler = &Peer{rai: rai, Bot: bot, Onion: rai.RemoteHostname}
			return chat
		}
	})

	af.OnClosed = func(rai *application.ApplicationInstance, err error) {
		if bot.OnDisconnect != nil {
			fmt.Println("Disconnection from ", rai.RemoteHostname)
			peer := bot.LookupPeerByHostname(rai.RemoteHostname)
			if peer != nil {
				bot.OnDisconnect(peer)
				bot.DeletePeer(peer)
			} else {
				fmt.Println("A nil peer disconnected: ", rai.RemoteHostname)
			}
		}
	}

	app := new(application.RicochetApplication)

	app.OnNewPeer = func(rai *application.ApplicationInstance, hostname string) {
		peer := new(Peer)
		peer.Onion = hostname
		peer.rai = rai
		peer.Bot = bot
		bot.Peers = append(bot.Peers, peer)
	}

	cm := new(RicochetBotContactManager)
	cm.bot = bot
	app.Init("APPLICATION", bot.PrivateKey, af, cm)

	al, err := application.SetupOnion("127.0.0.1:9051", "tcp4", "", bot.PrivateKey, 9878)
	if err != nil {
		log.Fatalf("Could not setup Onion: %v", err)
	}

	app.Run(al)
}
