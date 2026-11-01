package startup

import (
	"context"
	"fmt"
	"gateway/client"
	"gateway/config"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
)

const (
	defaultStartupTimeout = 5 * time.Minute
	startupRetryInterval  = 3 * time.Second
)

type Server struct {
	config         *config.Config
	noAuthConfig   *config.Config
	noAuthMethods  []string
	useRateLimiter bool
}

func NewServer(config *config.Config, noAuthConfig *config.Config, useRateLimiter bool) *Server {
	return &Server{config: config,
		noAuthConfig:   noAuthConfig,
		noAuthMethods:  nil,
		useRateLimiter: useRateLimiter}
}

func (s *Server) Start() {
	clientRegistry := s.prepareClients()
	s.noAuthMethods = getNoAuthMethods(s.noAuthConfig.Groups["core"]["v1"])
	router := s.prepareRoutes(clientRegistry)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", s.config.Gateway.Port), router))
}

func (s *Server) prepareClients() *client.ClientRegistry {
	log.Println("Preparing clients")
	clientRegistry := &client.ClientRegistry{
		Clients: make(map[string]client.Client),
	}

	deadline := time.Now().Add(startupTimeout())
	for k, v := range s.config.Services {
		if err := waitForClient(clientRegistry, k, v, deadline); err != nil {
			log.Fatalf("gateway startup failed: %v", err)
		}
	}

	return clientRegistry
}

// waitForClient retries a backend until it is reachable and reflects at least
// one service. Docker starts containers in arbitrary order after a daemon or
// host restart, so the gateway must not assume its dependencies are ready.
func waitForClient(registry *client.ClientRegistry, name, address string, deadline time.Time) error {
	var lastErr error
	for {
		err := registry.NewClient(name, address)
		if err == nil {
			return nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %s at %s: %w", name, address, lastErr)
		}
		log.Printf("waiting for %s at %s (%v); retrying in %s", name, address, lastErr, startupRetryInterval)
		time.Sleep(startupRetryInterval)
	}
}

func startupTimeout() time.Duration {
	raw := os.Getenv("GATEWAY_STARTUP_TIMEOUT")
	if raw == "" {
		return defaultStartupTimeout
	}
	timeout, err := time.ParseDuration(raw)
	if err != nil || timeout <= 0 {
		log.Printf("invalid GATEWAY_STARTUP_TIMEOUT=%q; using %s", raw, defaultStartupTimeout)
		return defaultStartupTimeout
	}
	return timeout
}
func (s *Server) prepareRoutes(clientRegistry *client.ClientRegistry) *mux.Router {
	router := mux.NewRouter().PathPrefix(s.config.Gateway.Route).Subrouter()

	for group, versions := range s.config.Groups {
		groupRouter := router.PathPrefix("/" + group).Subrouter()
		for version, methods := range versions {
			versionRouter := groupRouter.PathPrefix("/" + version).Subrouter()
			for mtdName, mtdConf := range methods {
				log.Printf("Name %s Conf %+v", mtdName, mtdConf)
				client := clientRegistry.Clients[mtdConf.Service]
				versionRouter.Path(mtdConf.MethodRoute).HandlerFunc(s.methodInterceptor(mtdName, client)).Methods(mtdConf.Type)
			}
		}
	}

	return router
}

func (s *Server) methodInterceptor(mtdName string, client client.Client) http.HandlerFunc {
	var h http.HandlerFunc

	if isNoAuthMethod(mtdName, s.noAuthMethods) {
		h = client.InvokeGrpcMethod
	} else {
		h = client.WrapGrpcMethod
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), "mtdName", mtdName)

		if s.useRateLimiter {
			systemAllowed := client.WithSystemRateLimiter(w, r)
			if !systemAllowed {
				http.Error(w, os.Getenv("rl.system.message"), http.StatusBadRequest)
				return
			}
		}

		h.ServeHTTP(w, r.WithContext(ctx))

	})
}

func getNoAuthMethods(noAuthMap map[string]config.MethodConfig) []string {
	keys := make([]string, 0, len(noAuthMap))
	for key := range noAuthMap {
		keys = append(keys, key)
	}
	return keys
}

func isNoAuthMethod(mtd string, noAuthMethods []string) bool {
	for _, element := range noAuthMethods {
		if element == mtd {
			return true
		}
	}
	return false
}
