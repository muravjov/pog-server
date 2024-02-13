package main

import (
	"time"

	flag "github.com/spf13/pflag"

	"git.catbo.net/muravjov/go2023/grpcproxy"
)

func main() {
	name := flag.String("name", "name", "account name")
	pass := flag.String("password", "password", "account password")
	timeToLive := flag.Duration("timeToLive", time.Hour*24*30*6, "when the account to exprire; default is half of a year")

	flag.Parse()

	grpcproxy.GenAuthItem(*name, *pass, *timeToLive)
}
