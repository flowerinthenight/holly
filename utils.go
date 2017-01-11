package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const (
	PS_ALL = iota
	PS_ANY
)

func eInfo(v ...interface{}) {
	m := fmt.Sprint(v...) // does not insert space in between items.
	el.Info(1, m)
}

func eError(v ...interface{}) {
	m := fmt.Sprint(v...) // does not insert space in between items.
	el.Error(1, m)
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

// Up to 15 args only.
func execute(args []string) ([]byte, error) {
	var (
		con []byte
		err error
	)

	switch len(args) {
	case 1:
		cmd := exec.Command(args[0])
		con, err = cmd.Output()
	case 2:
		cmd := exec.Command(args[0], args[1])
		con, err = cmd.Output()
	case 3:
		cmd := exec.Command(args[0], args[1], args[2])
		con, err = cmd.Output()
	case 4:
		cmd := exec.Command(args[0], args[1], args[2], args[3])
		con, err = cmd.Output()
	case 5:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4])
		con, err = cmd.Output()
	case 6:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5])
		con, err = cmd.Output()
	case 7:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6])
		con, err = cmd.Output()
	case 8:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7])
		con, err = cmd.Output()
	case 9:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8])
		con, err = cmd.Output()
	case 10:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9])
		con, err = cmd.Output()
	case 11:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10])
		con, err = cmd.Output()
	case 12:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11])
		con, err = cmd.Output()
	case 13:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12])
		con, err = cmd.Output()
	case 14:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13])
		con, err = cmd.Output()
	case 15:
		cmd := exec.Command(args[0], args[1], args[2], args[3], args[4], args[5], args[6], args[7], args[8], args[9], args[10], args[11], args[12], args[13], args[14])
		con, err = cmd.Output()
	}

	return con, err
}

// Run process as SYSTEM in the same session as winlogon.exe, not session 0.
func runInteractive(cmd string, args string, wait bool, waitms int) (uint32, error) {
	path, _ := getModuleFileName()
	lib := filepath.Dir(path) + `\libcore.dll`
	if _, err := os.Stat(lib); os.IsNotExist(err) {
		return uint32(syscall.ENOENT), fmt.Errorf("Cannot find libcore.dll.")
	}

	var (
		exitCode uint32
		runUser  = syscall.MustLoadDLL(lib).MustFindProc("StartSystemUserProcess")
	)

	shouldWait := 1
	if !wait {
		shouldWait = 0
	}

	_, _, err := runUser.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(cmd))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(args))),
		0,
		uintptr(unsafe.Pointer(&exitCode)),
		uintptr(shouldWait),
		uintptr(waitms))

	return exitCode, err
}

// The 'and' argument specifies the type of check for the list names; true is all names should
// be running, false if its only one (or any) of the names list.
func isProcessActive(check int, names ...string) bool {
	if len(names) == 0 {
		return false
	}

	res := true
	cmd := exec.Command("c:/windows/system32/tasklist.exe", "/fo", "csv", "/nh")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	switch check {
	case PS_ALL:
		s := strings.ToLower(fmt.Sprintf("%s", out))
		for _, name := range names {
			if !strings.Contains(s, strings.ToLower(name)) {
				return false
			}
		}
	case PS_ANY:
		s := strings.ToLower(fmt.Sprintf("%s", out))
		found := 0
		for _, name := range names {
			if strings.Contains(s, strings.ToLower(name)) {
				found++
			}
		}

		if found > 1 {
			return true
		} else {
			res = false
		}
	default:
		return false
	}

	return res
}

// Note that user has no option to cancel since this is from session 0. Default to 10 seconds.
func rebootSystem() error {
	cmd := exec.Command("shutdown", "/r", "/t", "10")
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

// The way to detect this is if git.exe and/or msbuild.exe is/are running.
func isRunnerActive() bool {
	return isProcessActive(PS_ANY, "git.exe", "msbuild.exe")
}
