package main

import (
	"sync"

	"github.com/PretendoNetwork/friends/coreevents"
	"github.com/PretendoNetwork/friends/crossnotify"
	"github.com/PretendoNetwork/friends/crosspresence"
	"github.com/PretendoNetwork/friends/grpc"
	"github.com/PretendoNetwork/friends/nex"
)

var wg sync.WaitGroup

func main() {
	wg.Add(3)

	go grpc.StartGRPCServer()
	go nex.StartAuthenticationServer()
	go nex.StartSecureServer()
	go coreevents.Start()
	// Wii U and 3DS presence into the account core, and friends live on other
	// platforms back out to the consoles here.
	go crosspresence.Start(crossnotify.Tick)

	wg.Wait()
}
