package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
	"golang.org/x/sys/windows/svc"
)

const (
	svcName         = "holly"
	internalVersion = "1.10"
	usage           = "Simple command scheduler (Windows service)"
	copyright       = "(c) 2016 Chew Esmero."
)

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Println("Failed to determine if we are running in an interactive session: %v", err)
		return
	}

	if !isIntSess {
		runService(svcName)
		return
	}

	app := cli.NewApp()
	app.Name = svcName
	app.Usage = usage
	app.Version = internalVersion
	app.Copyright = copyright
	app.Commands = []cli.Command{
		{
			Name:  "install",
			Usage: "install service",
			Action: func(c *cli.Context) error {
				return installService(svcName, svcName)
			},
		},
		{
			Name:  "remove",
			Usage: "uninstall service",
			Action: func(c *cli.Context) error {
				return removeService(svcName)
			},
		},
		{
			Name:  "start",
			Usage: "start service",
			Action: func(c *cli.Context) error {
				return startService(svcName)
			},
		},
		{
			Name:  "stop",
			Usage: "stop service",
			Action: func(c *cli.Context) error {
				return controlService(svcName, svc.Stop, svc.Stopped)
			},
		},
		{
			Name:  "pause",
			Usage: "pause service execution",
			Action: func(c *cli.Context) error {
				return controlService(svcName, svc.Pause, svc.Paused)
			},
		},
		{
			Name:  "continue",
			Usage: "resume service execution",
			Action: func(c *cli.Context) error {
				return controlService(svcName, svc.Continue, svc.Running)
			},
		},
	}

	app.Run(os.Args)
}
