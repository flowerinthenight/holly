// +build windows
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	// "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode/utf16"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/tylerb/graceful"
	"github.com/urfave/negroni"
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/debug"
	"golang.org/x/sys/windows/svc/eventlog"
)

var elog debug.Log

type svccontext struct {
	conf string
	busy int32 // 0 = idle; 1 = busy
}

// readLines reads a whole file into memory and returns a slice of its lines.
func readLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// Get the full name (with path) of the executing module.
func getModuleFileName() (string, error) {
	var sysproc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("GetModuleFileNameW")
	b := make([]uint16, syscall.MAX_PATH)
	r, _, err := sysproc.Call(0, uintptr(unsafe.Pointer(&b[0])), uintptr(len(b)))
	n := uint32(r)
	if n == 0 {
		return "", err
	}

	return string(utf16.Decode(b[0:n])), nil
}

func localExec(args []string) (string, error) {
	var outStr string
	var out bytes.Buffer
	var err error
	switch len(args) {
	case 1:
		c := exec.Command(args[0])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 2:
		c := exec.Command(args[0], args[1])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 3:
		c := exec.Command(args[0], args[1], args[2])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 4:
		c := exec.Command(args[0], args[1], args[2], args[3])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 5:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 6:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 7:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 8:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 9:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 10:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 11:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 12:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 13:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 14:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	case 15:
		c := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14])
		c.Stdout = &out
		if err = c.Run(); err != nil {
			trace(err.Error())
		} else {
			outStr = out.String()
		}
	}

	return outStr, err
}

func setUpdateSelfAfterReboot(old string, new string) error {
	var MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	var sysproc = syscall.MustLoadDLL("kernel32.dll").MustFindProc("MoveFileExW")
	o, err := syscall.UTF16PtrFromString(old)
	if err != nil {
		trace(err.Error())
	}

	n, err := syscall.UTF16PtrFromString(new)
	if err != nil {
		trace(err.Error())
	}

	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(o)), 0, uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))
	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(o)), uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))
	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(n)), 0, uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))

	return nil
}

func rebootSystem() error {
	c := exec.Command("shutdown", "/r", "/t", "10")
	if err := c.Run(); err != nil {
		trace(err.Error())
		return err
	}

	return nil
}

func handlePostUpdateGitlabRunner(m *svccontext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer file.Close()
		str := fmt.Sprintf("Handler.Header: %v", handler.Header)
		trace(str)
		path, _ := getModuleFileName()
		_, fstr := filepath.Split(handler.Filename)
		fstr = filepath.Dir(path) + `\` + fstr
		f, err := os.Create(fstr)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer f.Close()
		io.Copy(f, file)

		// Replace the runner exe
		retry := 10
		runner := `c:\runner\gitlab-ci-multi-runner-windows-amd64.exe`
		trace(runner + ` --> ` + fstr)
		for i := 0; i < retry; i++ {
			c := exec.Command(runner, "stop")
			if err := c.Run(); err != nil {
				trace("retry:", i, err)
				if i >= retry-1 {
					http.Error(w, "stop: "+err.Error(), 500)
					return
				}
			} else {
				break
			}
		}

		for i := 0; i < retry; i++ {
			c := exec.Command("cmd", "/c", "copy", "/Y", fstr, filepath.Dir(runner)+`\`)
			if err := c.Run(); err != nil {
				trace("retry:", i, err)
				if i >= retry-1 {
					http.Error(w, "copy: "+err.Error(), 500)
					return
				}
			} else {
				break
			}
		}

		// Restart service regardless of update result status
		defer func() {
			for i := 0; i < retry; i++ {
				c := exec.Command(runner, "start")
				if err := c.Run(); err != nil {
					trace("retry:", i, err)
					if i >= retry-1 {
						http.Error(w, "start: "+err.Error(), 500)
						return
					}
				} else {
					break
				}
			}
		}()

		fmt.Fprintf(w, "GitLab runner updated.")
	})
}

// Main service function.
func handleMainExecute(m *svccontext) error {
	// Set busy/idle state
	atomic.StoreInt32(&m.busy, 1)
	defer atomic.StoreInt32(&m.busy, 0)

	path, err := getModuleFileName()
	if err != nil {
		trace(err.Error())
		return err
	}

	dir, _ := filepath.Abs(filepath.Dir(path))
	lines, err := readLines(dir + "\\run.conf")
	if err != nil {
		trace(err.Error())
		return err
	}

	var start, end []int
	for _, str := range lines {
		s := strings.TrimSpace(str)
		var s2 []string
		if len(str) > 0 {
			if s[0] != '#' {
				trace(s)
				items := strings.Split(s, " ")
				if strings.TrimSpace(items[0]) == "sync" && strings.TrimSpace(items[1]) == "=" {
					for i, e := range items {
						if len(e) > 0 {
							if e[0] == '"' {
								start = append(start, i)
							}

							if e[len(e)-1] == '"' {
								end = append(end, i)
							}
						}
					}

					// Extract double-quoted arguments
					trace("Double-quoted arguments indeces:")
					tr := fmt.Sprintf("  start:%v, end:%v", start, end)
					trace(tr)
					for i, e := range start {
						s2 = append(s2, strings.Join(items[e:end[i]+1], " "))
					}

					// Reconstruct arguments list
					var items2 []string
					skip := false
					j := 0
					for _, e := range items {
						if len(e) > 0 {
							if e[0] == '"' {
								items2 = append(items2, s2[j])
								j += 1
								skip = true
							} else {
								if e[len(e)-1] == '"' {
									skip = false
								} else {
									if !skip {
										items2 = append(items2, e)
									}
								}
							}
						}
					}

					trace("Arguments list:")
					items2 = append(items2[:0], items2[2:]...)
					for _, e := range items2 {
						trace("  " + e)
					}

					localExec(items2)

					// Clear up for reuse
					s2 = nil
					start = nil
					end = nil
					trace("\n")
				}
			}
		}
	}

	return nil
}

func handlePostUpdateSelf(m *svccontext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		// By default, we reboot after setup update. To cancel, we need reboot=false param.
		reboot := true
		rb, ok := q["reboot"]
		if ok {
			if rb[0] == "false" {
				reboot = false
			}
		}

		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer file.Close()
		str := fmt.Sprintf("Handler.Header: %v", handler.Header)
		trace(str)
		path, _ := getModuleFileName()
		_, fstr := filepath.Split(handler.Filename)
		fstr = filepath.Dir(path) + `\` + fstr + `_new`
		f, err := os.Create(fstr)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer f.Close()
		io.Copy(f, file)
		fmt.Fprintf(w, "Self update applied after reboot.")
		trace(path + ` --> ` + fstr)
		err = setUpdateSelfAfterReboot(path, fstr)
		if reboot {
			trace("Rebooting system...")
			rebootSystem()
		}
	})
}

func handleGetInternalVersion(m *svccontext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, internalVersion)
	})
}

func handleGetBuildExists(m *svccontext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// variables
		vars := mux.Vars(r)
		name := vars["name"]
		trace("{name}", name)
		// query params
		q := r.URL.Query()
		bld, ok := q["build"]
		if ok {
			trace("Build param: " + bld[0])
			base := `c:\ftp-deploy\` + name + `\x64\`
			files, err := ioutil.ReadDir(base + bld[0])
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}

			ping := false
			for _, f := range files {
				p := base + f.Name()
				trace(p)
				matched, _ := regexp.MatchString(`(bin|iEMV|iemv)-\[.+\]\.zip$`, p)
				if matched {
					ping = true
					break
				}
			}

			if ping {
				fmt.Fprintf(w, "TRUE")
			} else {
				fmt.Fprintf(w, "FALSE")
			}
		} else {
			http.Error(w, "Query parameter 'build' required.", 500)
			return
		}
	})
}

// This is quite dangerous since we can execute virtually any command, considering that this service
// is running as SYSTEM account in session 0.
func handlePostExec(m *svccontext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer r.Body.Close()
		var m map[string]interface{}
		err = json.Unmarshal(body, &m)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		cmd, ok := m["cmd"]
		if !ok {
			http.Error(w, "No 'cmd' option found.", 500)
			return
		}

		cmdStr := fmt.Sprintf("%s", cmd)
		args := strings.Split(cmdStr, " ")
		res, err := localExec(args)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		trace(`[` + cmdStr + `]` + "\n" + res)
		fmt.Fprintf(w, `[`+cmdStr+`]`+"\n"+res)
	})
}

func (m *svccontext) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	tickdef := 5 * time.Second // we default to 5 seconds for main timer
	proceed := false
	var busy int32

	// Start our main http interface
	go func() {
		mux := mux.NewRouter()
		// API version 1
		v1 := mux.PathPrefix("/api/v1").Subrouter()
		v1.Methods("GET").Path("/version").Handler(handleGetInternalVersion(m))
		v1.Methods("GET").Path("/{name}/builds/exists").Handler(handleGetBuildExists(m))
		v1.Methods("POST").Path("/update/self").Handler(handlePostUpdateSelf(m))
		v1.Methods("POST").Path("/update/runner").Handler(handlePostUpdateGitlabRunner(m))
		v1.Methods("POST").Path("/exec").Handler(handlePostExec(m))
		n := negroni.Classic()
		n.UseHandler(mux)
		trace("Launching http interface...")
		graceful.Run(":8080", 5*time.Second, n)
	}()

	// Get the timer delay from config file
	path, err := getModuleFileName()
	if err != nil {
		trace(err.Error())
	} else {
		dir, _ := filepath.Abs(filepath.Dir(path))
		conf, err := ioutil.ReadFile(dir + "\\run.conf")
		if err != nil {
			trace(err.Error())
		} else {
			re := regexp.MustCompile(`timer\s?=\s?\d+`)
			tmr := re.Find(conf)
			if tmr != nil {
				kvtmr := strings.Split(string(tmr), "=")
				val, err := strconv.Atoi(strings.TrimSpace(kvtmr[1]))
				if err != nil {
					trace(err.Error())
				} else {
					str := fmt.Sprintf("New timer tick: %d (seconds)", val)
					trace(str)
					tickdef = time.Duration(val) * time.Second
					proceed = true
				}
			}
		}
	}

	maintick := time.Tick(tickdef)
	slowtick := time.Tick(2 * time.Second)
	tick := maintick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-tick:
			busy = atomic.LoadInt32(&m.busy)
			if busy == 0 {
				if proceed {
					go func(m *svccontext) {
						handleMainExecute(m)
					}(m)
				}
			}
		case c := <-r:
			switch c.Cmd {
			case svc.Interrogate:
				changes <- c.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- c.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = maintick
			default:
				elog.Error(1, fmt.Sprintf("unexpected control request #%d", c))
			}
		}
	}
	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string, conf string, isDebug bool) {
	var err error
	if isDebug {
		elog = debug.New(name)
	} else {
		elog, err = eventlog.Open(name)
		if err != nil {
			return
		}
	}
	defer elog.Close()

	elog.Info(1, fmt.Sprintf("starting %s service", name))
	run := svc.Run
	if isDebug {
		run = debug.Run
	}
	err = run(name, &svccontext{conf: conf, busy: 0})
	if err != nil {
		elog.Error(1, fmt.Sprintf("%s service failed: %v", name, err))
		return
	}
	elog.Info(1, fmt.Sprintf("%s service stopped", name))
}
