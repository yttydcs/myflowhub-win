package localhub

import (
	"errors"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"syscall"
)

func pickPort(host string, desired int) (port int, changed bool, err error) {
	if desired < 0 || desired > 65535 {
		return 0, false, errors.New("port must be 0..65535")
	}

	tryListen := func(p int) (int, error) {
		addr := net.JoinHostPort(strings.TrimSpace(host), strconv.Itoa(p))
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return 0, err
		}
		got := ln.Addr().(*net.TCPAddr).Port
		_ = ln.Close()
		return got, nil
	}

	if desired == 0 {
		p, err := tryListen(0)
		return p, false, err
	}

	p, err := tryListen(desired)
	if err == nil {
		return p, false, nil
	}

	p, err2 := tryListen(0)
	if err2 != nil {
		return 0, false, fmt.Errorf("desired port unavailable (%v), and auto-pick failed (%v)", err, err2)
	}
	return p, true, nil
}

func configureDetached(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	if runtime.GOOS == "windows" {
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	}
}
