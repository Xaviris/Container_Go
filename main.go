package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	// "strings"
	"syscall"
	// "github.com/fsnotify/fsnotify"
)

// go run main.go run <hostname> <cmd> <params>

var (
	hostname string
)

func main() {
	// set flags for command
	flag.StringVar(&hostname, "hostname", "container", "HostName for container")

	switch os.Args[1] {
	case "run":
		run()
	case "child":
		// remove the "run" argument and then parse flags
		os.Args = append(os.Args[:1], os.Args[2:]...)
		flag.Parse()
		child()
	default:
		panic("bad command")
	}
}

func run() {
	fmt.Printf("Running %v\n as %d\n", os.Args[2:], os.Getpid())

	// set command to be executed from arguments
	// re invoke process with command child args...
	cmd := exec.Command("/proc/self/exe", append([]string{"child"}, os.Args[2:]...)...)

	// view input, output, and errors
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// create new namespace
	// UTS = Unix Time System
	// name space for mount to avoid cluttering host mount
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags:   syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID | syscall.CLONE_NEWNS,
		Unshareflags: syscall.CLONE_NEWNS,
	}

	// run set command inside new namespace
	cmd.Run()

}

// have to create child process because can't set hostname before namespace is created with cmd.Run()
// and can't create it after because it has exited the Run command
// re invoking the process inside the namespace will modify hostname before exiting Run
func child() {
	fmt.Printf("Running %v\n as %d\n", os.Args[2:], os.Getpid())

	// create control groups function
	cg()

	// set namespace hostname
	syscall.Sethostname([]byte(hostname))

	// change root directory to new ubuntu fs

	exePath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to determine executable path: %v", err)
	}
	baseDir := filepath.Dir(exePath)
	chrootPath := filepath.Join(baseDir, "ubuntu-chroot")
	syscall.Chroot(chrootPath)
	syscall.Chdir("/")

	// mount proc directory as sub file system for kernel
	syscall.Mount("proc", "proc", "proc", 0, "")
	// set command to be executed from arguments
	cmd := exec.Command(os.Args[2], os.Args[3:]...)

	// view input, output, and errors
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// run set command
	cmd.Run()

	// unmount /proc when finished
	syscall.Unmount("/proc", 0)
}

// control group to limit memory that processes can use inside the container process
func cg() {
	// file path of control groups on host machine
	cgroups := "/sys/fs/cgroup/"
	// create a directory for container control groups
	err := os.Mkdir(filepath.Join(cgroups, "container"), 0755)

	// if err is not 0 and os.IsExist is false (check if directory exists, if false, panic)
	if err != nil && !os.IsExist(err) {
		panic(err)
	}
	// enable pids controller for cgroup
	cmd := exec.Command("echo '+pids' | sudo tee /sys/fs/cgroup/container/cgroup.subtree_control")
	cmd.Run()

	// Constrain number of processes within container to 20
	must(ioutil.WriteFile(filepath.Join(cgroups, "container/pids.max"), []byte("20"), 0700))
	// Gets current process of container and adds to control group processes to apply limits
	must(ioutil.WriteFile(filepath.Join(cgroups, "container/cgroup.procs"), []byte(strconv.Itoa(os.Getpid())), 0700))
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
