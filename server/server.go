package server

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"rigpig/internal"
	"rigpig/server/api"
	"rigpig/server/console"
	"rigpig/server/remoteAgent"
	"sync"
	"syscall"
	"time"
)

const (
	DEFAULT_ENABLE_WEB_CONSOLE   bool = false
	DEFAULT_ENABLE_CONSOLE       bool = true
	DEFAULT_ENABLE_REMOTE_AGENTS bool = true
	DEFAULT_ENABLE_API           bool = true
)

var TopAlgosResults []*internal.AlgoStats

type Server struct {
	wgServer           sync.WaitGroup
	wgRemoteAgents     sync.WaitGroup
	EnableWebConsole   bool
	EnableConsole      bool
	EnableApi          bool
	EnableRemoteAgents bool
}

func NewServer() *Server {
	return &Server{
		EnableApi:          DEFAULT_ENABLE_API,
		EnableConsole:      DEFAULT_ENABLE_CONSOLE,
		EnableRemoteAgents: DEFAULT_ENABLE_REMOTE_AGENTS,
		EnableWebConsole:   DEFAULT_ENABLE_WEB_CONSOLE,
	}
}

var remoteAgentService = make(chan string)
var apiServerService = make(chan string)
var webConsoleService = make(chan string)
var TopAlgoUpdates = make(chan []internal.AlgoStats)
var CurrencyRates = make(chan []internal.Currency)
var gracefulStop = make(chan os.Signal)

func (s *Server) Start() {

	// Moniter service status

	// Capture OS signals for graceful stop

	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)

	enabledServices := s.countEnabledServices()
	log.Printf("Loading %d services", enabledServices)

	s.wgServer.Add(enabledServices + 2)

	go func() {
		log.Print("Initializing currency updates")
		s.UpdateCurrencyRates()
	}()

	go func() {
		log.Print("Initializing crypto updates")
		s.UpdateCryptoStats()
	}()

	if s.EnableRemoteAgents == true {
		go s.NewRemoteAgentServer(remoteAgentService)
	}
	if s.EnableWebConsole == true {
		go s.webConsole(webConsoleService)
	}
	if s.EnableApi == true {
		go s.NewApiServer(apiServerService)
	}
	if s.EnableConsole == true {
		go s.console()
	}

	for {
		select {
		case algoUpdates := <-TopAlgoUpdates:
			internal.OutputAlgoStats = algoUpdates
			break
		case currencyRates := <-CurrencyRates:
			internal.LatestCurrencyRates = currencyRates
			break
		case <-gracefulStop:
			fmt.Println("EXITING NOW!!!!")
			internal.NotDone = false
		default:
		}
		time.Sleep(time.Second)
	}

	s.wgServer.Wait()
}

func gracefulExit(sig os.Signal) {
	log.Printf("Received signal: %s", sig)
	os.Exit(0)
}

func (s *Server) UpdateCryptoStats() {
	defer s.wgServer.Done()

	sleepTime := time.Minute * 5

	//log.Print("Initializing Algo Updates")
	for  {
		//log.Println("==> Getting latest algo stats...")
		TopAlgoUpdates <- internal.UpdateAlgos()
		//log.Printf("==> UpdateCryptoStats sleeping for %s...", sleepTime)
		time.Sleep(sleepTime)
	}

}

func (s *Server) UpdateCurrencyRates() {
	defer s.wgServer.Done()
	sleepTime := time.Minute * 60

	for internal.NotDone {
		// fetch and store rates here
		time.Sleep(sleepTime)
	}
}

func (s *Server) countEnabledServices() (services int) {
	services = 0
	if s.EnableApi == true {
		services++
	}
	if s.EnableConsole == true {
		services++
	}
	if s.EnableRemoteAgents == true {
		services++
	}
	if s.EnableWebConsole == true {
		services++
	}
	return
}

func AppCleanup() {
	log.Println("==> Cleaning up before exiting!")
	log.Println("==> Cleanup completed!")
}

func (s *Server) NewRemoteAgentServer(remoteAgentService chan string) {
	defer s.wgServer.Done()

	RAServer := remoteAgent.NewRemoteAgentServer()

	err := RAServer.Listen()
	if err != nil {
		log.Println(err)
	}

}

func (s *Server) webConsole(webConsoleService chan string) {
	defer s.wgServer.Done()

	APIServer := api.NewApiServer()
	APIServer.Port = "3001"

	log.Println("Starting Web Console Server...")
	err := APIServer.Listen()
	if err != nil {
		log.Println(err)
	}
}

func (s *Server) NewApiServer(apiServerConsole chan string) {
	defer s.wgServer.Done()

	APIServer := api.NewApiServer()

	err := APIServer.Listen()
	if err != nil {
		log.Println(err)
	}

}

func (s *Server) console() {
	defer s.wgServer.Done()

	console.MakeConsole(TopAlgoUpdates)

}
