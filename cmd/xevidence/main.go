package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/godeamon/chain-xevidence/config"
	elog "github.com/godeamon/chain-xevidence/log"
	"github.com/godeamon/chain-xevidence/worker"
	"gopkg.in/yaml.v2"
)

var (
	cfgpath = flag.String("config", "conf/config.yaml", "path of config file")
)

func main() {
	flag.Parse()
	config := config.DefaultConfig()
	yamlFile, err := os.ReadFile(*cfgpath)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, config)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(config)
	elog.InitLog()
	elog.Log.Debug("XEvidence", "configPath", *cfgpath, "config", config)
	mgr, err := worker.NewManager(config, *cfgpath)
	if err != nil {
		log.Fatal(err)
	}
	mgr.Start()
	log.Print("Evidence started.")
	defer mgr.Stop()

	sigint := make(chan os.Signal, 1)
	signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)

	<-sigint

	log.Print("Interrupted, evidence exieded.")
}
