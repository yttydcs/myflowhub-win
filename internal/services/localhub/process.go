// Context: implements the process helper logic used by the localhub backend service.

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

const (
	envHubSelfID           = "HUB_SELF_ID"
	envHubParentJoinPermit = "HUB_PARENT_JOIN_PERMIT"
)

type launchSpec struct {
	Addr string
	Args []string
	Env  []string
}

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

func normalizeLaunchRequest(req LaunchRequest) LaunchRequest {
	req.ParentJoinPermit = strings.TrimSpace(req.ParentJoinPermit)
	return req
}

func validateLaunchInputs(cfg Config, req LaunchRequest) error {
	parent := strings.TrimSpace(cfg.Parent)
	selfID := strings.TrimSpace(cfg.SelfID)
	req = normalizeLaunchRequest(req)

	if cfg.ParentEnable && parent == "" {
		return errors.New("parent address is required when parent link is enabled")
	}
	if cfg.ParentReconnectSec < 0 {
		return errors.New("parent reconnect seconds must be 0 or a positive number")
	}
	if req.ParentJoinPermit == "" {
		return nil
	}
	if !cfg.ParentEnable {
		return errors.New("enable parent link before using a parent join permit")
	}
	if parent == "" {
		return errors.New("parent address is required when parent join permit is provided")
	}
	if selfID == "" {
		return errors.New("self ID is required when parent join permit is provided")
	}
	return nil
}

func buildLaunchSpec(baseEnv []string, cfg Config, req LaunchRequest, addr string) (launchSpec, error) {
	req = normalizeLaunchRequest(req)
	if err := validateLaunchInputs(cfg, req); err != nil {
		return launchSpec{}, err
	}

	args := []string{"-addr", addr, "-node-id", strconv.Itoa(cfg.NodeID)}
	if cfg.ParentEnable {
		parent := strings.TrimSpace(cfg.Parent)
		args = append(args, "-parent", parent, "-parent-enable=true")
		if cfg.ParentReconnectSec != 0 {
			args = append(args, "-parent-reconnect", strconv.Itoa(cfg.ParentReconnectSec))
		}
	}

	if v := strings.TrimSpace(cfg.AuthDefaultRole); v != "" {
		args = append(args, "-auth-default-role", v)
	}
	if v := strings.TrimSpace(cfg.AuthDefaultPerms); v != "" {
		args = append(args, "-auth-default-perms", v)
	}
	if v := strings.TrimSpace(cfg.AuthNodeRoles); v != "" {
		args = append(args, "-auth-node-roles", v)
	}
	if v := strings.TrimSpace(cfg.AuthRolePerms); v != "" {
		args = append(args, "-auth-role-perms", v)
	}
	args = append(args, splitExtraArgs(cfg.ExtraArgs)...)

	env := append([]string(nil), baseEnv...)
	env = upsertEnv(env, envHubSelfID, strings.TrimSpace(cfg.SelfID))
	env = upsertEnv(env, envHubParentJoinPermit, req.ParentJoinPermit)

	return launchSpec{
		Addr: addr,
		Args: args,
		Env:  env,
	}, nil
}

func upsertEnv(env []string, key string, value string) []string {
	for i, entry := range env {
		name, _, found := strings.Cut(entry, "=")
		if !found {
			name = entry
		}
		if strings.EqualFold(name, key) {
			env[i] = key + "=" + value
			return env
		}
	}
	return append(env, key+"="+value)
}
