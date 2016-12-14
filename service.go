package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"github.com/gorilla/mux"
	"github.com/tylerb/graceful"
	"github.com/urfave/negroni"
	"golang.org/x/sys/windows/svc"
	_ "golang.org/x/sys/windows/svc/debug"
	_ "golang.org/x/sys/windows/svc/eventlog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type httpContextValue struct {
	ipaddr string
}

type etw struct {
	mod  *syscall.LazyDLL
	proc *syscall.LazyProc
	init bool
}

func (e *etw) trace(v ...interface{}) {
	if !e.init {
		return
	}

	pc, _, _, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	fno := regexp.MustCompile(`^.*\.(.*)$`)
	fnName := fno.ReplaceAllString(fn.Name(), "$1")
	m := fmt.Sprint(v...)
	_, _, _ = e.proc.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("[" + fnName + "] " + m))))
}

func newEtw() *etw {
	path, _ := getModuleFileName()
	lib := filepath.Dir(path) + `\disptrace.dll`
	if _, err := os.Stat(lib); os.IsNotExist(err) {
		return nil
	}

	mod := syscall.NewLazyDLL(lib)
	proc := mod.NewProc("ETWTrace")
	return &etw{mod: mod, proc: proc, init: true}
}

// Service's main context structure.
type svcContext struct {
	*log.Logger                 // rotating logs (service level) using lumberjack
	*etw                        // embedded etw tracer
	busy        int32           // 0 = idle; 1 = busy
	mruns       map[string]bool // run state for cmd lines
}

type cmdRunner struct {
	console string
	err     error
}

func (r *cmdRunner) run(cmd *exec.Cmd) {
	var out bytes.Buffer
	cmd.Stdout = &out
	r.err = cmd.Run()
	if r.err == nil {
		r.console = out.String()
	}
}

// Up to 15 args only. I don't know how to make this dynamic.
func (c *svcContext) execute(args []string) (string, error) {
	cr := cmdRunner{}
	switch len(args) {
	case 1:
		cr.run(exec.Command(args[0]))
	case 2:
		cr.run(exec.Command(args[0], args[1]))
	case 3:
		cr.run(exec.Command(args[0], args[1], args[2]))
	case 4:
		cr.run(exec.Command(args[0], args[1], args[2], args[3]))
	case 5:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4]))
	case 6:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5]))
	case 7:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6]))
	case 8:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7]))
	case 9:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8]))
	case 10:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9]))
	case 11:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10]))
	case 12:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11]))
	case 13:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12]))
	case 14:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13]))
	case 15:
		cr.run(exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14]))
	}

	return cr.console, cr.err
}

func (c *svcContext) setUpdateSelfAfterReboot(old string, new string) error {
	var (
		sysproc                     = syscall.MustLoadDLL("kernel32.dll").MustFindProc("MoveFileExW")
		MOVEFILE_DELAY_UNTIL_REBOOT = 0x4
	)

	o, err := syscall.UTF16PtrFromString(old)
	if err != nil {
		c.trace(err)
	}

	n, err := syscall.UTF16PtrFromString(new)
	if err != nil {
		c.trace(err)
	}

	// Register file replacements.
	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(o)), 0, uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))
	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(n)), uintptr(unsafe.Pointer(o)), uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))
	_, _, _ = sysproc.Call(uintptr(unsafe.Pointer(n)), 0, uintptr(MOVEFILE_DELAY_UNTIL_REBOOT))
	return nil
}

func handleHttpGetInternalVersion(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"version":"` + internalVersion + `"}`))
	})
}

func handleHttpGetExec(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			interactive bool = false
			wait        bool = true
			waitms      int  = 5000
		)

		q := r.URL.Query()
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer r.Body.Close()
		cmd := fmt.Sprintf("%s", body)
		qi, ok := q["interactive"]
		if ok {
			if qi[0] == "true" {
				interactive = true
				// The 'wait' and 'waitms' args are only valid when 'interactive' is true.
				if val, ok := q["wait"]; ok {
					if val[0] == "false" {
						wait = false
					}
				}

				if val, ok := q["waitms"]; ok {
					ms, err := strconv.Atoi(val[0])
					if err == nil {
						waitms = ms
					}
				}
			}
		}

		v := httpContextValue{ipaddr: r.RemoteAddr}
		ctx := context.WithValue(context.Background(), "data", v)
		doExec(ctx, c, w, cmd, interactive, wait, waitms)
	})
}

// This is quite dangerous since we can execute virtually any command, considering that this service
// is running as SYSTEM account in session 0.
func doExec(ctx context.Context, c *svcContext, w http.ResponseWriter, cmd string, interactive, wait bool, waitms int) {
	ip := ctx.Value("data").(httpContextValue).ipaddr + ` | `
	c.trace(ip, cmd)
	args := strings.Split(cmd, " ")
	if interactive {
		c.trace(ip, "cmd: ", args[0])
		c.trace(ip, "args (joined): ", strings.Join(args[1:], " "))
		r, err := runInteractive(args[0], strings.Join(args[1:], " "), wait, waitms)
		c.trace(ip, "return: ", r, ", err: ", err)
		w.Write([]byte(`{"cmd":"` + cmd + `","return":"` + fmt.Sprintf("%s", r) + `","error":"` + err.Error() + `"}`))
		return
	}

	res, err := c.execute(args)
	if err != nil {
		c.trace(ip, err)
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte(`{"cmd":"` + cmd + `","result":"` + res + `"}`))
}

func handleHttpGetFileStat(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			data string                        // data string
			ip   string = r.RemoteAddr + ` | ` // for logging
		)

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer r.Body.Close()
		files := fmt.Sprintf("%s", body)
		c.trace(ip, files)
		fl := strings.Split(files, ",")
		var fss = map[string]string{}
		for _, f := range fl {
			c.trace(ip, f)
			stats, err := os.Stat(f)
			if err != nil {
				data += err.Error()
			} else {
				data += "name:" + stats.Name() + ","
				data += "size:" + fmt.Sprintf("%v", stats.Size()) + ","
				data += "mode:" + fmt.Sprintf("%v", stats.Mode()) + ","
				data += "modtime:" + fmt.Sprintf("%v", stats.ModTime()) + ","
				data += "isdir:" + fmt.Sprintf("%v", stats.IsDir())
			}

			fss[f] = data
			data = ""
		}

		payload, err := json.Marshal(fss)
		if err != nil {
			http.Error(w, err.Error(), 500)
		} else {
			w.Write(payload)
		}
	})
}

func handleHttpGetReadFile(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr + ` | ` // for logging
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer r.Body.Close()
		file := fmt.Sprintf("%s", body)
		c.trace(ip, file)
		data, err := ioutil.ReadFile(file)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		// Just send the raw contents as reply.
		w.Write(data)
	})
}

// Update self binary. This, by default, reboots the system. To cancel, use 'reboot=false' param.
func handleHttpPostUpdateSelf(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     string = r.RemoteAddr + ` | ` // for logging
			reboot bool   = true
		)

		q := r.URL.Query()
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
		c.trace(ip, str)
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
		c.trace(ip, path+` --> `+fstr)
		// Send reply first before triggering reboot (if needed).
		w.Write([]byte(`{"result":"Self update applied.","reboot":"` + fmt.Sprintf("%v", reboot) + `"}`))
		err = c.setUpdateSelfAfterReboot(path, fstr)
		if reboot {
			c.trace(ip, "Rebooting system...")
			rebootSystem()
		}
	})
}

func handleHttpPostUpdateGitlabRunner(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     string = r.RemoteAddr + ` | ` // for logging
			runner string = `c:\runner\gitlab-ci-multi-runner-windows-amd64.exe`
			retry  int    = 10
		)

		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer file.Close()
		str := fmt.Sprintf("Handler.Header: %v", handler.Header)
		c.trace(ip, str)
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

		// Restart service regardless of update result status.
		defer func() {
			for i := 0; i < retry; i++ {
				c.trace(ip, "attempt (start): ", i)
				cmd := exec.Command(runner, "start")
				out, err := cmd.Output()
				if err == nil {
					sout := fmt.Sprintf("out: %s", out)
					c.trace(sout)
					break
				}

				c.trace(err)
				if i >= retry-1 {
					http.Error(w, "start: "+err.Error(), 500)
					return
				}
			}
		}()

		// Don't do anything if runner is active.
		if isRunnerActive() {
			c.trace(ip, "Runner is active. Skip update.")
			w.Write([]byte(`{"result":"GitLab runner active. Skip update."}`))
			return
		}

		// Stop the runner service.
		c.trace(ip, runner+` --> `+fstr)
		for i := 0; i < retry; i++ {
			c.trace(ip, "attempt (stop): ", i)
			cmd := exec.Command(runner, "stop")
			out, err := cmd.Output()
			if err == nil {
				sout := fmt.Sprintf("out: %s", out)
				c.trace(sout)
				break
			}

			c.trace(err)
			if i >= retry-1 {
				http.Error(w, "stop: "+err.Error(), 500)
				return
			}
		}

		// Replace the runner exe.
		for i := 0; i < retry; i++ {
			c.trace(ip, "attempt (copy): ", i)
			cmd := exec.Command(
				"c:\\windows\\system32\\cmd.exe",
				"/c",
				"copy",
				"/Y",
				fstr,
				filepath.Dir(runner)+`\`)
			out, err := cmd.Output()
			if err == nil {
				sout := fmt.Sprintf("out: %s", out)
				c.trace(sout)
				break
			}

			c.trace(err)
			if i >= retry-1 {
				http.Error(w, "copy: "+err.Error(), 500)
				return
			}
		}

		w.Write([]byte(`{"result":"GitLab runner updated."}`))
	})
}

func handleHttpPostUpdateConf(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr + ` | ` // for logging
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer file.Close()
		atomic.StoreInt32(&c.busy, 1)
		defer atomic.StoreInt32(&c.busy, 0)
		str := fmt.Sprintf("Handler.Header: %v", handler.Header)
		c.trace(ip, str)
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
		w.Write([]byte(`{"result":"Config file updated."}`))
	})
}

// Upload any file to some location.
func handleHttpPostUpload(c *svcContext) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr + ` | ` // for logging
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer file.Close()
		str := fmt.Sprintf("Handler.Header: %v", handler.Header)
		c.trace(ip, str)
		path := r.FormValue("path")
		c.trace("path: " + path)
		_, fstr := filepath.Split(handler.Filename)
		if path == "root" {
			fp, _ := getModuleFileName()
			fstr = filepath.Dir(fp) + `\` + fstr
		} else {
			fstr = path + `\` + fstr
		}

		f, err := os.Create(fstr)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		defer f.Close()
		io.Copy(f, file)
		// Send full path of file as reply.
		w.Write([]byte(`{"file":"` + fstr + `"}`))
	})
}

// Returns true if the command line is scheduled (should run). The second value is the total target minutes when
// the schedule option takes the '*/frequency' form. If zero, that means, the scheduled time is specific.
func isCmdLineScheduled(c *svcContext, line string) (bool, uint64) {
	star5 := 0 // special case
	var target uint64 = 0
	mults := [5]uint64{
		1,     // minutes in a minute
		60,    // minutes in an hour
		1440,  // minutes in a day
		43800, // minutes in a month
		0,     // skip (always exact weekday)
	}

	evals := [5]bool{false, false, false, false, false}
	tms := [5]int{
		time.Now().Minute(),
		time.Now().Hour(),
		time.Now().Day(),
		int(time.Now().Month()),
		int(time.Now().Weekday()),
	}

	c.trace("reftime: ", tms)
	r := fmt.Sprintf("^(%d|\\*|\\*/\\d+)\\s(%d|\\*|\\*/\\d+)\\s(%d|\\*|\\*/\\d+)\\s(%d|\\*|\\*/\\d+)\\s(%d|\\*|\\*/\\d+)",
		tms[0],
		tms[1],
		tms[2],
		tms[3],
		tms[4])

	c.trace("regexp: ", r)
	items := strings.Split(line, " ")
	match, _ := regexp.MatchString(r, line)
	if match {
		for idx, item := range items {
			// Only the first 5 items are needed.
			if idx > 4 {
				break
			}

			if item == "*" {
				evals[idx] = true
				star5 += 1
				continue
			}

			day := false
			if idx == 2 {
				// Day of month (priority over 'day of week').
				if v, err := strconv.Atoi(item); err == nil {
					day = true
					if v == tms[idx] {
						evals[idx] = true
						evals[4] = true
						continue
					}
				}
			}

			if idx == 4 {
				if !day {
					// Day of week.
					if v, err := strconv.Atoi(item); err == nil {
						if v == tms[idx] {
							evals[idx] = true
							evals[2] = true
							continue
						}
					}
				}
			}

			vals := strings.Split(item, "/")
			if len(vals) == 2 && idx < 4 /* day-of-week not included */ {
				if val, err := strconv.ParseUint(vals[1], 10, 64); err == nil {
					target += val * mults[idx]
					evals[idx] = true
					continue
				}
			}

			if v, err := strconv.Atoi(item); err == nil {
				if v == tms[idx] {
					evals[idx] = true
					continue
				}
			}
		}

		valid := true
		for _, eval := range evals {
			if !eval {
				valid = false
				break
			}
		}

		// When all inputs are '*'s, we set target to '1'.
		if star5 == 5 && target == 0 {
			target = 1
		}

		c.trace("evals: ", evals)
		c.trace("target: ", target)
		return valid, target
	}

	return false, 0
}

func handleMainExecute(c *svcContext, count uint64) error {
	atomic.StoreInt32(&c.busy, 1)
	defer atomic.StoreInt32(&c.busy, 0)

	path, err := getModuleFileName()
	if err != nil {
		c.trace(err)
		return err
	}

	dir, _ := filepath.Abs(filepath.Dir(path))
	lines, err := readLines(dir + "\\run.conf")
	if err != nil {
		c.trace(err)
		return err
	}

	activeLinesExact := map[string]bool{}
	var start, end []int
	for _, str := range lines {
		s := strings.TrimSpace(str)
		var s2 []string
		// Skip blank lines...
		if len(str) == 0 {
			continue
		}

		// and comments.
		if s[0] == '#' {
			continue
		}

		items := strings.Split(s, " ")
		c.trace(items)
		for i, e := range items {
			if len(e) == 0 {
				continue
			}

			if e[0] == '"' {
				start = append(start, i)
			}

			if e[len(e)-1] == '"' {
				end = append(end, i)
			}
		}

		// Extract double-quoted arguments.
		c.trace("Double-quoted arguments indeces:")
		tr := fmt.Sprintf("  start:%v, end:%v", start, end)
		c.trace(tr)
		for i, e := range start {
			s2 = append(s2, strings.Join(items[e:end[i]+1], " "))
		}

		// Reconstruct arguments list.
		var items2 []string
		skip := false
		j := 0
		for _, e := range items {
			if len(e) == 0 {
				continue
			}

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

		// Should be at least sched params + a single cmd.
		if len(items2) < 6 {
			continue
		}

		c.trace("Arguments list:")
		items2 = append(items2[:0], items2[5:]...)
		for _, e := range items2 {
			c.trace("  " + e)
		}

		// Run the command line.
		sched, target := isCmdLineScheduled(c, s)
		if target > 0 {
			c.trace("count: ", count)
			if math.Mod(float64(count), float64(target)) != 0 {
				sched = false
			}
		}

		if sched {
			exec := true
			if v, found := c.mruns[s]; found {
				if v == true {
					c.trace("Exact sched: should exec once (already executed).")
					exec = false
				}
			}

			if exec {
				c.trace("Execute: ", items2)
				c.Println("Execute", items2)
				_, err := c.execute(items2)
				if err != nil {
					c.trace(err)
					c.Println(err)
				}
			}

			if target == 0 {
				// We store this line since for the 'exact time' type of schedule, we need to execute
				// only once per every minute tick. For the 'every x time' type, we don't mind.
				//
				// Example, if the sched is * 1 * * *, that means once every hour. Since our tick is
				// per minute, this will normally execute once per min at 1:00am (total of 60 execs).
				// We don't want that to happen.
				activeLinesExact[s] = true
				c.mruns[s] = true
			}
		}

		s2 = nil
		start = nil
		end = nil
		c.trace("\n")
	}

	// Cleanup mruns.
	var delkeys []string
	for k, v := range c.mruns {
		_, active := activeLinesExact[k]
		if !active && v == true {
			delkeys = append(delkeys, k)
		}
	}

	if len(delkeys) > 0 {
		for _, k := range delkeys {
			delete(c.mruns, k)
		}
	}

	for k, v := range c.mruns {
		c.trace("key: ", k, ", val: ", v)
	}

	c.trace("----------\n")
	return nil
}

// Our service's main worker function.
func (c *svcContext) Execute(args []string, r <-chan svc.ChangeRequest, changes chan<- svc.Status) (ssec bool, errno uint32) {
	c.trace("Starting service: ", svcName)
	const cmdsAccepted = svc.AcceptStop | svc.AcceptShutdown | svc.AcceptPauseAndContinue
	changes <- svc.Status{State: svc.StartPending}
	c.mruns = map[string]bool{}
	tickdef := 1 * time.Minute

	var (
		cntr uint64 = 0
		busy int32
	)

	// Start our main http interface.
	go func() {
		mux := mux.NewRouter()
		v1 := mux.PathPrefix("/api/v1").Subrouter()
		v1.Methods("GET").Path("/version").Handler(handleHttpGetInternalVersion(c))
		v1.Methods("GET").Path("/exec").Handler(handleHttpGetExec(c))
		v1.Methods("GET").Path("/filestat").Handler(handleHttpGetFileStat(c))
		v1.Methods("GET").Path("/readfile").Handler(handleHttpGetReadFile(c))
		v1.Methods("POST").Path("/update/self").Handler(handleHttpPostUpdateSelf(c))
		v1.Methods("POST").Path("/update/runner").Handler(handleHttpPostUpdateGitlabRunner(c))
		v1.Methods("POST").Path("/update/conf").Handler(handleHttpPostUpdateConf(c))
		v1.Methods("POST").Path("/upload").Handler(handleHttpPostUpload(c))
		n := negroni.Classic()
		n.UseHandler(mux)
		c.trace("Launching http interface.")
		graceful.Run(":8080", 5*time.Second, n)
	}()

	maintick := time.Tick(tickdef)
	slowtick := time.Tick(2 * time.Second)
	tick := maintick
	changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
loop:
	for {
		select {
		case <-tick:
			cntr = cntr + 1
			if cntr == math.MaxUint64 {
				cntr = 1
			}

			busy = atomic.LoadInt32(&c.busy)
			if busy == 0 {
				go func(ctx *svcContext, count uint64) {
					handleMainExecute(ctx, count)
				}(c, cntr)
			}
		case crq := <-r:
			switch crq.Cmd {
			case svc.Interrogate:
				changes <- crq.CurrentStatus
				// Testing deadlock from https://code.google.com/p/winsvc/issues/detail?id=4
				time.Sleep(100 * time.Millisecond)
				changes <- crq.CurrentStatus
			case svc.Stop, svc.Shutdown:
				break loop
			case svc.Pause:
				changes <- svc.Status{State: svc.Paused, Accepts: cmdsAccepted}
				tick = slowtick
			case svc.Continue:
				changes <- svc.Status{State: svc.Running, Accepts: cmdsAccepted}
				tick = maintick
			default:
				c.trace("Unexpected control request #", crq)
			}
		}
	}

	changes <- svc.Status{State: svc.StopPending}
	return
}

func runService(name string) {
	// Create our main service context with etw tracer and rotating logger.
	path, _ := getModuleFileName()
	rlf := &lumberjack.Logger{
		Dir:        filepath.Dir(path),
		NameFormat: "holly.log",
		MaxSize:    500,
		MaxBackups: 3,
		MaxAge:     30,
	}

	ctx := svcContext{
		Logger: log.New(rlf, "HOLLY: ", log.Ldate|log.Ltime|log.Lshortfile),
		etw:    newEtw(),
		busy:   0,
	}

	defer rlf.Close()
	run := svc.Run
	err := run(name, &ctx)
	if err != nil {
		ctx.trace("Service failed: ", err)
		return
	}

	ctx.trace("Service stopped: ", name)
}
