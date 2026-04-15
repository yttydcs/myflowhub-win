// 本文件提供 `myflowhub-mcp` 的命令行入口，并把参数接入本地 MCP 运行时。

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/yttydcs/myflowhub-win/internal/mcp"
	"github.com/yttydcs/myflowhub-win/internal/mcpapp"
)

type cliConfig struct {
	endpoint      string
	configDir     string
	deviceID      string
	displayName   string
	defaultTarget uint
	timeout       time.Duration
	allowWrite    bool
	versionOnly   bool
}

func main() {
	// main 负责组装 CLI 参数、runtime 和 stdio server，并托管整个 MCP 进程生命周期。
	cfg := parseFlags()
	if cfg.versionOnly {
		fmt.Fprintln(os.Stdout, buildVersion())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	runtime, err := mcpapp.New(mcpapp.Config{
		Context:       ctx,
		ConfigDir:     cfg.configDir,
		Endpoint:      cfg.endpoint,
		DeviceID:      cfg.deviceID,
		DisplayName:   cfg.displayName,
		DefaultTarget: uint32(cfg.defaultTarget),
		AllowWrite:    cfg.allowWrite,
		Timeout:       cfg.timeout,
		LogWriter:     os.Stderr,
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "myflowhub-mcp init failed: %v\n", err)
		os.Exit(1)
	}
	defer runtime.Close()

	server, err := mcp.NewServer(mcp.ServerConfig{
		Name:         "myflowhub-mcp",
		Version:      buildVersion(),
		Instructions: "Use connect first, then register or login, then call management or varstore tools. stdout is reserved for MCP JSON-RPC; operational logs are sent to stderr.",
		Reader:       os.Stdin,
		Writer:       os.Stdout,
		Tools:        mcp.NewTools(runtime),
	})
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "myflowhub-mcp server init failed: %v\n", err)
		os.Exit(1)
	}

	if err := server.Serve(ctx); err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, io.EOF) {
		_, _ = fmt.Fprintf(os.Stderr, "myflowhub-mcp serve failed: %v\n", err)
		os.Exit(1)
	}
}

func parseFlags() cliConfig {
	// parseFlags 只解析启动默认值，不在这里做业务校验，保持 CLI 入口职责单一。
	cfg := cliConfig{}
	flag.StringVar(&cfg.endpoint, "endpoint", "", "default hub endpoint")
	flag.StringVar(&cfg.configDir, "config-dir", "", "isolated MCP config directory")
	flag.StringVar(&cfg.deviceID, "device-id", "", "default device ID")
	flag.StringVar(&cfg.displayName, "display-name", "", "default display name for register/login")
	flag.UintVar(&cfg.defaultTarget, "default-target", 0, "default target node ID")
	flag.DurationVar(&cfg.timeout, "timeout", 8*time.Second, "request timeout")
	flag.BoolVar(&cfg.allowWrite, "allow-write", false, "allow write tools such as varstore_set and varstore_revoke")
	flag.BoolVar(&cfg.versionOnly, "version", false, "print version and exit")
	flag.Parse()
	return cfg
}

func buildVersion() string {
	// buildVersion 优先读取构建信息里的模块版本，拿不到时回退到开发态标记。
	if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
		if version := bi.Main.Version; version != "" && version != "(devel)" {
			return version
		}
	}
	return "dev"
}
