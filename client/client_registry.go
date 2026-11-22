package client

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/fullstorydev/grpcurl"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ClientRegistry struct {
	Clients map[string]Client
}

// maxMsgSize must match the largest request a backend accepts. Supernova's
// BuildImage carries the build context (tar.gz) in one unary message; the
// edge-base context is ~23 MB raw, so 64 MiB leaves headroom. The limit is a
// ceiling, not an allocation: other backends are unaffected.
const maxMsgSize = 64 << 20

// NewClient dials a backend, discovers its reflected services and registers a
// client for it. It returns an error instead of panicking so the gateway can
// retry during startup: Docker may start the gateway before its dependencies
// are ready after a host reboot.
func (registry *ClientRegistry) NewClient(name, address string) error {
	log.Printf("New client %s address %s", name, address)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cc, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(maxMsgSize),
			grpc.MaxCallSendMsgSize(maxMsgSize),
		),
		grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("dial %s (%s): %w", name, address, err)
	}

	refClient := grpcreflect.NewClientAuto(context.Background(), cc)
	descSource := grpcurl.DescriptorSourceFromServer(context.Background(), refClient)
	services, err := descSource.ListServices()
	if err != nil {
		cc.Close()
		return fmt.Errorf("list services for %s (%s): %w", name, address, err)
	}

	filteredServices := make([]string, 0, len(services))
	for _, str := range services {
		if !strings.HasPrefix(str, "grpc.") {
			filteredServices = append(filteredServices, str)
		}
	}
	if len(filteredServices) == 0 {
		cc.Close()
		return fmt.Errorf("service %s (%s) exposes no non-grpc.* services", name, address)
	}

	registry.Clients[name] = Client{refClient: refClient, cc: cc, descSource: descSource, grpcServiceName: filteredServices[0]}
	return nil
}
