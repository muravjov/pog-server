package gstacks

import (
	"context"
	"fmt"
	"testing"

	"git.catbo.net/muravjov/go2023/grpctest"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "git.catbo.net/muravjov/go2023/gstacks/proto/v1"
)

func TestGS(t *testing.T) {
	t.SkipNow()

	fmt.Println(goroutineStacks())
}

func TestHC(t *testing.T) {
	//t.SkipNow()

	server := grpc.NewServer()
	RegisterGStacksSvc(server)

	sc, err := grpctest.StartServerClient(server)
	require.NoError(t, err)
	defer sc.Close()

	client := pb.NewGoroutineStacksClient(sc.Conn)
	resp, err := client.Invoke(context.Background(), &pb.Request{})
	require.NoError(t, err)

	fmt.Println(resp.Data)
}
