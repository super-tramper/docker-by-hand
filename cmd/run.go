// +build linux

package cmd

import (
	"docker/cgroups"
	"docker/cgroups/subsystems"
	"docker/container"
	log "github.com/sirupsen/logrus"
	"os"
	"strings"
)

// Run 运行用户命令
func Run(tty bool, comArray []string, volume string, res *subsystems.ResourceConfig) {
	// 对创建出来的进程进行初始化
	parent, writePipe := container.NewParentProcess(tty, volume)
	if parent == nil {
		log.Errorf("New parent process error")
		return
	}
	if err := parent.Start(); err != nil {
		log.Error(err)
	}

	cgroupManager := cgroups.NewCgroupManager("mydocker-cgroup")
	defer cgroupManager.Destroy()
	cgroupManager.Set(res)
	cgroupManager.Apply(parent.Process.Pid)

	// 发送用户指令
	sendInitCommand(comArray, writePipe)
	if tty {
		parent.Wait()
	}
	//mntURL := "/root/mnt/"
	//rootURL := "/root/"
	//container.DeleteWorkSpace(rootURL, mntURL, volume)
	//os.Exit(0)
}

func sendInitCommand(comArray []string, writePipe *os.File) {
	command := strings.Join(comArray, " ")
	log.Infof("command all is %s", command)
	writePipe.WriteString(command)
	writePipe.Close()
}
