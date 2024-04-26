module gateway

go 1.21.3

require (
	github.com/fullstorydev/grpcurl v1.8.8
	github.com/goccy/go-json v0.10.2
	github.com/gorilla/mux v1.8.0
	github.com/jhump/protoreflect v1.15.2
	google.golang.org/grpc v1.63.2
	gopkg.in/yaml.v3 v3.0.1
	apollo v1.0.0
	rate-limiter-service v1.0.0
)

require (
	github.com/bufbuild/protocompile v0.6.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.19.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sync v0.6.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240401170217-c3f982113cda // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240325203815-454cdb8f5daa // indirect
	google.golang.org/protobuf v1.33.0 // indirect
)

replace github.com/c12s/oort => ../oort

replace github.com/c12s/magnetar => ../magnetar

replace apollo => ../apollo

replace rate-limiter-service => ../heliosphere/rate-limiter-service
