package ricochetbot

import (
	"crypto/rsa"
	"github.com/jes/go-ricochet/application"
	"github.com/jes/go-ricochet/channels"
	"log"
	"sync"
)

type RicochetBot struct {
	PrivateKey               *rsa.PrivateKey
	Peers                    []*Peer
	peerLock                 sync.Mutex
	TorControlAddress        string
	TorControlType           string
	TorControlAuthentication string
	TorSocksAddress          string

	app *application.RicochetApplication

	OnConnect        func(*Peer)
	OnNewPeer        func(*Peer) bool
	OnReadyToChat    func(*Peer)
	OnMessage        func(*Peer, string)
	OnContactRequest func(*Peer, string, string) bool
	OnDisconnect     func(*Peer)
}

func (bot *RicochetBot) Connect(onion string, message string) error {
	if bot.LookupPeerByHostname(onion) != nil {
		return nil
	}
	instance, err := bot.app.Open(onion, message)
	if err != nil {
		return err
	}

	bot.AddPeer(instance, onion)
	return nil
}

func (bot *RicochetBot) Shutdown() {
	bot.app.Shutdown()
}

func (bot *RicochetBot) AddPeer(rai *application.ApplicationInstance, hostname string) *Peer {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	// if we already have this peer, just return it
	for _, peer := range bot.Peers {
		if peer.Onion == hostname && peer.rai == rai {
			return peer
		}
	}

	peer := new(Peer)
	peer.Onion = hostname
	peer.rai = rai
	peer.Bot = bot
	bot.Peers = append(bot.Peers, peer)
	return peer
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

// delete & disconncet from other peers with the same onion as peer.Onion
func (bot *RicochetBot) DedupPeers(peer *Peer) {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	for i := 0; i < len(bot.Peers); i++ {
		p := bot.Peers[i]
		if p.Onion == peer.Onion && p != peer {
			// remove this peer by swapping the final peer into its place and then
			// removing the final peer (this works even on the final entry)
			bot.Peers[i].Disconnect()
			bot.Peers[i] = bot.Peers[len(bot.Peers)-1]
			bot.Peers = bot.Peers[:len(bot.Peers)-1]

			// re-process this index in case it also needs deleting
			i--
		}
	}
}

// reutrn first peer with this hostname
func (bot *RicochetBot) LookupPeerByHostname(onion string) *Peer {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	for _, peer := range bot.Peers {
		if peer.Onion == onion {
			return peer
		}
	}
	return nil
}

func (bot *RicochetBot) LookupAllPeersByHostname(onion string) []*Peer {
	bot.peerLock.Lock()
	defer bot.peerLock.Unlock()

	peers := make([]*Peer, 0)

	for _, peer := range bot.Peers {
		if peer.Onion == onion {
			peers = append(peers, peer)
		}
	}
	return peers
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
			contact.Handler = ch
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
		peers := bot.LookupAllPeersByHostname(rai.RemoteHostname)
		if bot.OnDisconnect != nil && len(peers) == 1 {
			// only when there's exactly 1 peer of this name has our state changed from "connected" to "not connected"
			bot.OnDisconnect(peers[0])
		}
		if len(peers) > 0 {
			bot.DeletePeer(peers[0])
		}
	}

	bot.app = new(application.RicochetApplication)
	cm := new(RicochetBotContactManager)
	cm.bot = bot
	bot.app.Init("", bot.PrivateKey, af, cm)

	if bot.TorSocksAddress != "" {
		bot.app.SOCKSProxy = bot.TorSocksAddress
	}

	bot.app.MakeContactHandler = func(rai *application.ApplicationInstance) channels.ContactRequestChannelHandler {
		ch := new(RicochetBotContactHandler)
		ch.Onion = rai.RemoteHostname
		ch.bot = bot
		ch.rai = rai
		return ch
	}

	bot.app.OnNewPeer = func(rai *application.ApplicationInstance, hostname string) {
		bot.AddPeer(rai, hostname)
	}

	bot.app.OnAuthenticated = func(rai *application.ApplicationInstance, known bool) {
		bot.AddPeer(rai, rai.RemoteHostname)
		peer := bot.LookupPeerByHostname(rai.RemoteHostname)

		if bot.OnConnect != nil {
			go bot.OnConnect(peer)
		}

		if known {
			// XXX: call the handler for when an inbound channel is opened, this is a bodge so that the bot can
			// open an outbound channel immediately if it wants to; having an outbound channel always open makes
			// it possible to send messages to a peer without having to first remember to open the outbound channel
			go peer.OpenInbound()
		}
	}

	if bot.TorControlAddress == "" {
		bot.TorControlAddress = "127.0.0.1:9051"
	}
	if bot.TorControlType == "" {
		bot.TorControlType = "tcp4"
	}

	al, err := application.SetupOnion(bot.TorControlAddress, bot.TorControlType, bot.TorControlAuthentication, bot.PrivateKey, 9878)
	if err != nil {
		log.Fatalf("Could not setup Onion: %v", err)
	}

	bot.app.Run(al)
}
