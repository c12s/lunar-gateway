package main

import (
	"fmt"
	"gateway/config"
	"gateway/startup"
	"os"
)

var path = "config.yml"
var noAuthPath = "no_auth_config.yml"

func main() {
	conf, err := config.LoadConfig(path)
	if err != nil {
		fmt.Println(err)
		return
	}

	noAuthConf, err := config.LoadConfig(noAuthPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	args := os.Args
	useRateLimiter := false
	if len(args) > 1 && args[0] == "sysrl" {
		useRateLimiter = true
	}

	gateway := startup.NewServer(conf, noAuthConf, useRateLimiter)
	gateway.Start()
}
