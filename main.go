// +build windows
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/urfave/cli"
	"golang.org/x/sys/windows/svc"
	"gopkg.in/natefinch/lumberjack.v2"
)

const internalVersion = "1.8"
const svcName = "holly"

var (
	mod  *syscall.LazyDLL
	proc *syscall.LazyProc
	rlf  *log.Logger
)

func trace(v ...interface{}) {
	if proc == nil {
		return
	}

	// Log only when trace library is present.
	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	fno := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := fno.ReplaceAllString(fn.Name(), "$1")
	m := fmt.Sprint(v...)
	_, _, _ = proc.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("[" + fnName + "] " + m))))
}

func initRotatingLog(out io.Writer) {
	rlf = log.New(out, "HOLLY: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func main() {
	// Initialize rotating logs
	path, _ := getModuleFileName()
	initRotatingLog(&lumberjack.Logger{
		Dir:        path,
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     30,
	})

	proc = nil
	lib := filepath.Dir(path) + `\disptrace.dll`
	if _, err := os.Stat(lib); os.IsNotExist(err) {
		rlf.Println("Cannot find disptrace.dll.")
	} else {
		mod = syscall.NewLazyDLL("disptrace.dll")
		proc = mod.NewProc("ETWTrace")
	}

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
	app.Usage = "Simple command scheduler (Windows service)"
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
