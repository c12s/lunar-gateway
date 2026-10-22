module gateway

go 1.21.3

require (
	apollo v1.0.0
	github.com/fullstorydev/grpcurl v1.8.8
	github.com/goccy/go-json v0.10.2
	github.com/gorilla/mux v1.8.0
	github.com/jhump/protoreflect v1.15.2
	google.golang.org/grpc v1.65.0
	gopkg.in/yaml.v3 v3.0.1
	rate-limiter-service v1.0.0
)

require (
	github.com/bufbuild/protocompile v0.6.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/net v0.25.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240528184218-531527333157 // indirect
	google.golang.org/protobuf v1.34.1 // indirect
)

replace github.com/c12s/oort => ../oort

replace github.com/c12s/magnetar => ../magnetar

replace apollo => ../apollo

replace rate-limiter-service => ../heliosphere/rate-limiter-service
