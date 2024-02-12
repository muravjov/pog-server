package main

import "git.catbo.net/muravjov/go2023/util"

type Config struct {
	ServerAddr string // proxy-over-grpc server address (host:port)

	// TLS settings
	ServerHost string // Host name to which server IP should resolve
	Insecure   bool   // Skip SSL validation? [false]
	SkipVerify bool   // Skip server hostname verification in SSL validation [false]

	ClientListen string // this proxy-over-grpc client address to listen to [host]:port

	ClientPOGAuth string // auth string to connect to server, in the form user:password
}

func MakeConfig() Config {
	cfg := Config{}

	util.StringEnv(&cfg.ServerAddr, "SERVER_ADDR", "")

	util.StringEnv(&cfg.ServerHost, "SERVER_HOST", "")
	util.BoolEnv(&cfg.Insecure, "INSECURE", false)
	util.BoolEnv(&cfg.SkipVerify, "SKIP_VERIFY", false)

	util.StringEnv(&cfg.ClientListen, "CLIENT_LISTEN", ":18080")
	util.StringEnv(&cfg.ClientPOGAuth, "CLIENT_POG_AUTH", "")

	return cfg
}
