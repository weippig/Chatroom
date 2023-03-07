package main

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p"
)

// DiscoveryInterval is how often we re-publish our mDNS records.
const DiscoveryInterval = time.Hour

// DiscoveryServiceTag is used in our mDNS advertisements to discover other chat peers.
const DiscoveryServiceTag = "pubsub-chat-example"

func main() {
	var nickname string
	fmt.Println("Enter Your Nickname: ")
	fmt.Scanln(&nickname)

	var room string
	fmt.Println("Enter Your Room Name: ")
	fmt.Scanln(&room)

	// create a new libp2p Host that listens on a random TCP port
	h, err := libp2p.New(libp2p.ListenAddrStrings("/ip4/0.0.0.0/tcp/0"))
	if err != nil {
		panic(err)
	}

	fmt.Println("room:", room)
	fmt.Println("nick:", nickname)
	fmt.Println("host:", h.ID().Pretty())
}
