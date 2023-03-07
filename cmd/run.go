// +build linux

package cmd

import (
	"docker/container"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

func Run(tty bool, comArray []string) {
	parent, writePipe := container.NewParentProcess(tty)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	// 发送用户指令
	sendInitCommand(comArray, writePipe)
	parent.Wait()
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	container.DeleteWorkSpace(rootURL, mntURL)
	os.Exit(0)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
