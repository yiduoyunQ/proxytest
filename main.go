// proxytest project main.go
package main

import (
	"fmt"
	"os"
	"path"

	log "github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "upproxy health check"
	app.Version = "0.0.2"

	app.Author = "qjr"
	app.Email = "qiujirong@unionpay.com"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "log-level, l",
			Value: "debug",
			Usage: fmt.Sprintf("Log level (options: debug, info, warn, error, fatal, panic)"),
		},
		cli.BoolFlag{
			Name:  "debug, d",
			Usage: fmt.Sprintf("debug mode"),
		},
	}

	// logs
	app.Before = func(c *cli.Context) error {
		log.SetOutput(os.Stderr)
		level, err := log.ParseLevel(c.String("log-level"))
		if err != nil {
			log.Fatalf(err.Error())
		}
		log.SetLevel(level)

		// If a log level wasn't specified and we are running in debug mode,
		// enforce log-level=debug.
		if !c.IsSet("log-level") && !c.IsSet("l") && c.Bool("debug") {
			log.SetLevel(log.DebugLevel)
		}

		return nil
	}

	app.Commands = commands
	arguments := make([]string, len(os.Args[1:])+2)
	arguments[0] = ""
	arguments[1] = "proxyHealthCheck"
	copy(arguments[2:], os.Args[1:])

	if err := app.Run(arguments); err != nil {
		log.Fatal(err)
	}

}
