package healthcheck

import (
	"context"
	"encoding/json"
	"time"

	pb "git.catbo.net/muravjov/go2023/healthcheck/proto/v1"
	"google.golang.org/grpc"
)

func RegisterHealthcheckSvc(server *grpc.Server, appName string, startTimestamp time.Time, version string) {
	svc := &healthcheckServer{
		appName:        appName,
		startTimestamp: startTimestamp,
		version:        version,
	}
	pb.RegisterHealthcheckServer(server, svc)
}

type healthcheckServer struct {
	appName        string
	startTimestamp time.Time
	version        string

	pb.UnimplementedHealthcheckServer
}

func (s *healthcheckServer) Invoke(context.Context, *pb.Request) (*pb.Response, error) {
	uptime := time.Since(s.startTimestamp)
	m := HealthcheckType{
		Version:       s.version,
		App:           s.appName,
		UptimeSeconds: int64(uptime.Seconds()),
		Uptime:        uptime.String(),
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return &pb.Response{Data: string(jsonBytes)}, err
}
