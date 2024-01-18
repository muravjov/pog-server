package healthcheck

import (
	"context"
	"encoding/json"
	"time"

	pb "git.catbo.net/muravjov/go2023/healthcheck/proto/v1"
	"google.golang.org/grpc"
)

func RegisterHealthcheckSvc(server *grpc.Server, appName string, startTimestamp time.Time) {
	svc := &healthcheckServer{
		appName:        appName,
		startTimestamp: startTimestamp,
	}
	pb.RegisterHealthcheckServer(server, svc)
}

type healthcheckServer struct {
	appName        string
	startTimestamp time.Time

	pb.UnimplementedHealthcheckServer
}

func (s *healthcheckServer) Invoke(context.Context, *pb.Request) (*pb.Response, error) {
	uptime := time.Since(s.startTimestamp)
	m := HealthcheckType{
		"version":        Version,
		"app":            s.appName,
		"uptime_seconds": int64(uptime.Seconds()),
		"uptime":         uptime.String(),
	}

	jsonBytes, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}

	return &pb.Response{Data: string(jsonBytes)}, err
}
