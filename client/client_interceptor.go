package client

import (
	"context"
	"fmt"
	iam "iam-service/proto1"
	"log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InterceptRequest(vault_token string) (string, string, error) {
	conn, err := grpc.Dial("iam-service:8002", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := iam.NewAuthServiceClient(conn)

	getResp, err := client.VerifyToken(context.Background(), &iam.Token{Token: vault_token})
	if err != nil {
		fmt.Println(err)
		return "", "", err
	}

	return getResp.Token.Jwt, getResp.Username, nil
}
