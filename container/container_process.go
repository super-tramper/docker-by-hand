// +build linux

package container

import (
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"syscall"
)

func NewParentProcess(tty bool) (*exec.Cmd, *os.File) {
	readPipe, writePipe, err := NewPipe()
	if err != nil {
		log.Errorf("New pipe error %v", err)
		return nil, nil
	}
	cmd := exec.Command("/proc/self/exe", "init")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS |
			syscall.CLONE_NEWNET | syscall.CLONE_NEWIPC,
	}
	if tty {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	// 带着文件句柄创建子进程
	cmd.ExtraFiles = []*os.File{readPipe}
	mntURL := "/root/mnt/"
	rootURL := "/root/"
	NewWorkSpace(rootURL, mntURL)
	cmd.Dir = mntURL
	return cmd, writePipe
}

func NewPipe() (*os.File, *os.File, error) {
	// 创建匿名管道，返回读写两端
	read, write, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	return read, write, nil
}

// NewWorkSpace Create a AUFS filesystem as container root workspace
func NewWorkSpace(rootURL string, mntURL string) {
	CreateReadOnlyLayer(rootURL)
	CreateWriteLayer(rootURL)
	CreateMountPoint(rootURL, mntURL)
}

// CreateReadOnlyLayer 将busybox.tar解压到busybox目录下，作为容器的只读层
func CreateReadOnlyLayer(rootURL string) {
	busyboxURL := rootURL + "busybox/"
	busyboxTarURL := rootURL + "busybox.tar"
	exist, err := PathExists(busyboxURL)
	if err != nil {
		log.Infof("Fail to judge whether dir %s exists. %v", busyboxURL, err)
	}
	if exist == false {
		if err := os.Mkdir(busyboxURL, 0777); err != nil {
			log.Errorf("Mkdir dir %s error. %v", busyboxURL, err)
		}
		if _, err := exec.Command("tar", "-xvf", busyboxTarURL, "-C", busyboxURL).CombinedOutput(); err != nil {
			log.Errorf("Untar dir %s error %v", busyboxURL, err)
		}
	}
}

// CreateWriteLayer 创建一个名为writeLayer的文件夹作为容器唯一的可写层
func CreateWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.Mkdir(writeURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", writeURL, err)
	}
}

func CreateMountPoint(rootURL string, mntURL string) {
	// 创建mnt文件夹作为挂载点
	if err := os.Mkdir(mntURL, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntURL, err)
	}
	var tempDir = "tmp"
	if err := os.Mkdir(rootURL+tempDir, 0777); err != nil {
		log.Errorf("Mkdir dir %s error. %v", mntURL+tempDir, err)
	}
	// 把writeLayer目录和busybox目录mount到mnt目录下
	//dirs := "dirs=" + rootURL + "writeLayer:" + rootURL + "busybox"
	dirs := "lowerdir=" + rootURL + "busybox" + ",upperdir=" + rootURL + "writeLayer" + ",workdir=" + rootURL + tempDir
	//mount -t aufs -o dirs=/root/test2:/root/test1 none /root/mnt/
	//mount -t overlay overlay -o lowerdir=/home/kali/lower,upperdir=/home/kali/upper,workdir=/home/kali/workdir /mnt/overlay_test
	//mount -t overlay overlay -o lowerdir=/home/kali/Desktop/test1,upperdir=/home/kali/Desktop/test2,workdir=/home/kali/Desktop/test3 /home/kali/Desktop/test4
	//mount -t overlay overlay -o lowerdir=/home/uos/test1,upperdir=/home/uos/test2,workdir=/home/uos/test3 /home/uos/test4
	// mount -t overlay overlay -o lowerdir=/home/kali/Desktop/test1,upperdir=/home/kali/Desktop/test2,workdir=/home/kali/Desktop/test3 /home/kali/Desktop/test4
	// mount -t overlay overlay -o lowerdir=/root/test1,upperdir=/root/test2,workdir=/root/test3 /root/test4
	// mount -t overlay overlay -o lowerdir=/root/busybox,upperdir=/root/writeLayer,workdir=/root/tmp /root/mnt/
	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", dirs, mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("%v", err)
	}
}

// DeleteWorkSpace Delete the AUFS filesystem while container exit
func DeleteWorkSpace(rootURL string, mntURL string) {
	DeleteMountPoint(rootURL, mntURL)
	DeleteWriteLayer(rootURL)
}

func DeleteMountPoint(rootURL string, mntURL string) {
	cmd := exec.Command("umount", mntURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		log.Errorf("DeleteMountPoint1 %v", err)
	}
	if err := os.RemoveAll(mntURL); err != nil {
		log.Errorf("DeleteMountPoint2 Remove dir %s error %v", mntURL, err)
	}
}

func DeleteWriteLayer(rootURL string) {
	writeURL := rootURL + "writeLayer/"
	if err := os.RemoveAll(writeURL); err != nil {
		log.Errorf("Remove dir %s error %v", writeURL, err)
	}
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
