package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"

	"github.com/umputun/remark/app/boot"
)

// Opts with command line flags and env
// nolint:maligned
type Opts struct {
	Config string `short:"f" long:"config" env:"CONFIG" default:"config.yml" description:"config file"`
	Dbg    bool   `long:"dbg" env:"DEBUG" description:"debug mode"`
}

var revision = "unknown"

func main() {
	fmt.Printf("remark42 %s\n", revision)

	var opts Opts
	p := flags.NewParser(&opts, flags.Default)
	if _, e := p.ParseArgs(os.Args[1:]); e != nil {
		os.Exit(1)
	}
	setupLog(opts.Dbg)
	log.Print("[INFO] started remark42")

	conf, err := boot.NewConfig(opts.Config)
	if err != nil {
		log.Fatalf("failed to load %s, %s", opts.Config, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() { // catch signal and invoke graceful termination
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
		<-stop
		log.Print("[WARN] interrupt signal")
		cancel()
	}()

	app, err := boot.NewApplication(conf, revision)
	if err != nil {
		log.Fatalf("[ERROR] failed to setup application, %+v", err)
	}
	app.Run(ctx)
	log.Printf("[INFO] remark42 terminated")
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stdout,
	}

	log.SetFlags(log.Ldate | log.Ltime)

	if dbg {
		log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
		filter.MinLevel = logutils.LogLevel("DEBUG")
	}
	log.SetOutput(filter)
}
