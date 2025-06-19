package authclient

import (
	"context"
	"fmt"
	"time"

	"github.com/AlexMickh/speak-chat/pkg/utils/retry"
	"github.com/AlexMickh/speak-protos/pkg/api/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthClient struct {
	conn *grpc.ClientConn
	auth auth.AuthClient
}

func New(addr string) (*AuthClient, error) {
	const op = "grpc.clients.auth.New"

	var conn *grpc.ClientConn
	var authClient auth.AuthClient

	retry.WithDelay(5, 500*time.Millisecond, func() error {
		connect, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return fmt.Errorf("%s: %w", op, err)
		}

		auth := auth.NewAuthClient(conn)

		conn = connect
		authClient = auth

		return nil
	})

	return &AuthClient{
		conn: conn,
		auth: authClient,
	}, nil
}

func (a *AuthClient) GetUserId(ctx context.Context, token string) (string, error) {
	const op = "grpc.clients.auth.GetUserId"

	res, err := a.auth.VerifyToken(ctx, &auth.VerifyTokenRequest{
		AccessToken: token,
	})
	if err != nil {
		return "", fmt.Errorf("%s: %w", op, err)
	}

	return res.GetUserId(), nil
}

func (a *AuthClient) Close() {
	a.conn.Close()
}
