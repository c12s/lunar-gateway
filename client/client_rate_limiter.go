package client

import (
	"context"
	"log"
	rl "rate-limiter-service/proto/ratelimiter"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func CallRateLimiter(username string, mtdName string) (bool, error) {
	conn, err := grpc.Dial("rate_limiter_service:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := rl.NewRateLimitServiceClient(conn)

	id := username
	if mtdName != "" {
		id = mtdName + "-" + username
	}

	allowResp, err := client.CanRateLimiterAllowRequest(context.Background(), &rl.AllowRequest{Id: id})
	if err != nil {
		log.Println("Error parsing CanRateLimiterAllowRequest response")
		log.Println(err)
		return false, err
	}

	return allowResp.Allowed, nil
}
