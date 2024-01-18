package healthcheck

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"git.catbo.net/muravjov/go2023/grpctest"
	"git.catbo.net/muravjov/go2023/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	pb "git.catbo.net/muravjov/go2023/healthcheck/proto/v1"
)

func TestProxy(t *testing.T) {
	//t.SkipNow()

	server := grpc.NewServer()
	// registering should be done before grpcapi.Server.Start() = grpc.Server.Serve()
	RegisterHealthcheckSvc(server, "healthcheck", time.Now())

	sc, err := grpctest.StartServerClient(server)
	require.NoError(t, err)
	defer sc.Close()

	client := pb.NewHealthcheckClient(sc.Conn)

	resp, err := client.Invoke(context.Background(), &pb.Request{})
	require.NoError(t, err)

	d := HealthcheckType{}
	err = json.Unmarshal([]byte(resp.Data), &d)
	require.NoError(t, err)

	util.DumpIndent(d)
}
