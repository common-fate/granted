package api

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

func NewClientConn(ctx context.Context, serverAddr string) (*grpc.ClientConn, error) {
	useTLS := !strings.HasPrefix(serverAddr, "localhost")
	var opts []grpc.DialOption
	if useTLS {
		config := &tls.Config{
			InsecureSkipVerify: true,
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	opts = append(opts, grpc.WithBlock())

	dialContext, cancelDial := context.WithTimeout(ctx, time.Second*15)
	defer cancelDial()

	cc, err := grpc.DialContext(dialContext, serverAddr, opts...)
	if err != nil {
		return nil, errors.Wrap(err, "dialling gRPC server")
	}

	return cc, nil
}
