package action

import (
	"log"
	"os"
	"os/exec"

	"github.com/lxn/walk"
)

func NewRestartAction() (*walk.Action, error) {
	action := walk.NewAction()
	if err := action.SetText("重启"); err != nil {
		return nil, err
	}

	action.Triggered().Attach(restartHandler)
	return action, nil
}

func restartHandler() {
	exe, err := os.Executable()
	if err != nil {
		log.Println("failed to get executable path:", err)
		return
	}

	// 先启动新实例，再退出当前进程
	cmd := exec.Command(exe)
	if err := cmd.Start(); err != nil {
		log.Println("failed to start new instance:", err)
		return
	}

	walk.App().Exit(0)
}
