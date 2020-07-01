package server

import (
	"context"
	"fmt"
	"github.com/c12s/lunar-gateway/model/configs"
	stellar "github.com/c12s/stellar-go"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	"strings"
	"time"
)

type LunarServer struct {
	r          *mux.Router
	address    string
	clients    map[string]string
	instrument map[string]string
}

func createBaseRouter(version string) *mux.Router {
	r := mux.NewRouter().StrictSlash(false)
	prefix := strings.Join([]string{"/api", version}, "/")
	return r.PathPrefix(prefix).Subrouter()
}

func NewServer(conf *configs.Config) *LunarServer {
	//create server struct
	server := &LunarServer{
		r:          createBaseRouter(conf.ConfVersion),
		address:    conf.ServerConf.Address,
		clients:    conf.ServicesConf,
		instrument: conf.InstrumentConf,
	}

	//setup routes
	server.setupEndpoints()

	//if all is good return server
	return server
}

func (server *LunarServer) setupEndpoints() {
	server.setupConfigs()
	server.setupSecrets()
	server.setupActions()
	server.setupNamespaces()
	server.setupAuth()
	server.setupTrace()
	server.setupRoles()
	server.setupTopology()
	server.setupNodes()
}

func (server *LunarServer) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n, err := stellar.NewCollector(server.instrument["address"], server.instrument["stopic"])
	if err != nil {
		fmt.Println(err)
		return
	}
	c, err := stellar.InitCollector(server.instrument["location"], n)
	if err != nil {
		fmt.Println(err)
		return
	}
	go c.Start(ctx, 15*time.Second)

	fmt.Println("LunarServer Started")
	http.ListenAndServe(server.address, handlers.LoggingHandler(os.Stdout, server.r))
}
