package healthcheck

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.catbo.net/muravjov/go2023/grpctest"
	"git.catbo.net/muravjov/go2023/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "git.catbo.net/muravjov/go2023/healthcheck/proto/v1"
)

func invoke(conn *grpc.ClientConn, t *testing.T) HealthcheckType {
	client := pb.NewHealthcheckClient(conn)
	resp, err := client.Invoke(context.Background(), &pb.Request{})
	require.NoError(t, err)

	d := HealthcheckType{}
	err = json.Unmarshal([]byte(resp.Data), &d)
	require.NoError(t, err)
	return d
}

func TestHC(t *testing.T) {
	server := grpc.NewServer()
	// registering should be done before grpcapi.Server.Start() = grpc.Server.Serve()
	RegisterHealthcheckSvc(server, "healthcheck", time.Now(), "test")

	sc, err := grpctest.StartServerClient(server)
	require.NoError(t, err)
	defer sc.Close()

	d := invoke(sc.Conn, t)
	assert.Equal(t, d.App, "healthcheck")
	assert.Equal(t, d.Version, "test")
}

func TestHCRequest(t *testing.T) {
	//t.SkipNow()

	conn, err := grpctest.DialInsecure("localhost:8080")
	require.NoError(t, err)
	defer conn.Close()

	d := invoke(conn, t)
	util.DumpIndent(d)
}
