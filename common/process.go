package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
)

const (
	PL_PID_FNAME         = ".pid"
	PL_UMASK_PID_LOCKDIR = 0755
	PL_UMASK_PID_FILE    = 0644
)

type ProcessExistError struct{}

func (*ProcessExistError) Error() string {
	return "Process exists"
}

var (
	ProcessExist = &ProcessExistError{}
)

type ProcessLocker interface {
	Lock() error
	Unlock() error
}

type DirProcessLocker struct {
	DirName string
}

func NewDirProcessLocker(dirname string) ProcessLocker {
	return &DirProcessLocker{
		DirName: dirname,
	}
}

func (dpl *DirProcessLocker) getPidFilePath() string {
	return filepath.Join(dpl.DirName, PL_PID_FNAME)
}

func (dpl *DirProcessLocker) Lock() error {
	fpid := dpl.getPidFilePath()
	if err := os.Mkdir(dpl.DirName, PL_UMASK_PID_LOCKDIR); err != nil {
		perr := err.(*os.PathError)
		if perr.Err != os.EEXIST {
			return err
		}
		// check whether if the process truely exists or not(might go down by unexpected errors or something else...)
		if pidbuf, rerr := ioutil.ReadFile(fpid); rerr != nil {
			return rerr
		} else if pid, serr := strconv.Atoi(string(pidbuf)); serr != nil {
			return serr
		} else if Pid(pid).IsExist() {
			// process exist
			return ProcessExist
		}
		// previous process might go down with garbage files so go ahead
	}
	pidbuf := strconv.Itoa(os.Getpid())
	// make pid file
	if err := ioutil.WriteFile(fpid, []byte(pidbuf), PL_UMASK_PID_FILE); err != nil {
		return err
	}
	return nil
}

func (dpl *DirProcessLocker) Unlock() error {
	return os.RemoveAll(dpl.DirName)
}

type Pid int

func (p Pid) IsExist() bool {
	procfs := "/proc/" + strconv.Itoa(int(p))
	if fi, err := os.Lstat(procfs); err != nil {
		return false
	} else {
		return fi.IsDir()
	}
	// unreached
	panic("unreached")
}
