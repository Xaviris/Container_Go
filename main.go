package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// docker         run image <cmd> <params>
// go run main.go run       <cmd> <params>

func main() {
	fmt.Println(os.Args[1])
	switch os.Args[1] {
	case "run":
		run()
	case "child":
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

	// set namespace hostname
	syscall.Sethostname([]byte("container"))

	// change root directory to new ubuntu fs
	syscall.Chroot("/home/xavierruiz/Documents/Github/Container_Go/ubuntu-chroot")
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

func must(err error) {
	if err != nil {
		panic(err)
	}
}
