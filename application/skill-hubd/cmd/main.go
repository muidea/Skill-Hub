package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/muidea/skill-hub/internal/initiators/routeregistry"
	_ "github.com/muidea/skill-hub/internal/modules/application/daemonapi"
	_ "github.com/muidea/skill-hub/internal/modules/blocks/project_state"
	_ "github.com/muidea/skill-hub/internal/modules/blocks/repository"
	skillhubd "github.com/muidea/skill-hub/internal/services/skill-hubd"
)

func main() {
	host := flag.String("host", "127.0.0.1", "监听地址")
	port := flag.Int("port", 5525, "监听端口")
	secretKey := flag.String("secret-key", "", "远端推送密钥，未配置时禁止远端推送")
	showVersion := flag.Bool("version", false, "显示版本信息")
	flag.Parse()
	if *showVersion {
		fmt.Println(skillhubd.VersionString())
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := skillhubd.Run(ctx, skillhubd.Config{
		Host:      *host,
		Port:      *port,
		SecretKey: *secretKey,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "skill-hubd: %v\n", err)
		os.Exit(1)
	}
}
