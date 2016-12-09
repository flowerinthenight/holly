// +build windows
package main

import (
	"log"
	"os"

	"github.com/urfave/cli"
	"golang.org/x/sys/windows/svc"
)

const internalVersion = "1.9"
const svcName = "holly"
const usage = "Simple command scheduler (Windows service)"

func main() {
	isIntSess, err := svc.IsAnInteractiveSession()
	if err != nil {
		log.Println("Failed to determine if we are running in an interactive session: %v", err)
		return
	}

	if !isIntSess {
		runService(svcName, "", false)
		return
	}

	app := cli.NewApp()
	app.Name = svcName
	app.Usage = usage
	app.Version = internalVersion
	app.Copyright = "(c) 2016 Chew Esmero."
	app.Commands = []cli.Command{
		{
			Name:  "debug",
			Usage: "run service (debug enabled)",
			Action: func(c *cli.Context) error {
				runService(svcName, "", true)
				return nil
			},
		},
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
