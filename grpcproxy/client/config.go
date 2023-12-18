package main

import "git.catbo.net/muravjov/go2023/util"

type Config struct {
	ServerAddr string // Server address (host:port)

	// TLS settings
	ServerHost string // Host name to which server IP should resolve
	Insecure   bool   // Skip SSL validation? [false]
	SkipVerify bool   // Skip server hostname verification in SSL validation [false]
}

func MakeConfig() Config {
	cfg := Config{}

	util.StringEnv(&cfg.ServerAddr, "SERVER_ADDR", "")

	util.StringEnv(&cfg.ServerHost, "SERVER_HOST", "")
	util.BoolEnv(&cfg.Insecure, "INSECURE", false)
	util.BoolEnv(&cfg.SkipVerify, "SKIP_VERIFY", false)

	return cfg
}
