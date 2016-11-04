package tos

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
)

const (
	InvalidPID = -1
)

// WritePidFile writes this a process id into a file.
// An error will be returned if the file already exists.
func WritePidFile(pid int, filename string) error {
	pidString := strconv.Itoa(pid)
	pidFile, err := os.OpenFile(filename, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer pidFile.Close()
	_, err = pidFile.WriteString(pidString)
	return err
}

// WritePidFileForced writes this a process id into a file.
// An existing file will be overwritten.
func WritePidFileForced(pid int, filename string) error {
	pidString := strconv.Itoa(pid)
	return ioutil.WriteFile(filename, []byte(pidString), 0644)
}

// GetPidFromFile tries loads the content of a pid file.
// A pidfile is expected to contain only an integer with a valid process id.
func GetPidFromFile(filename string) (int, error) {
	var (
		pidString []byte
		pid       int
		err       error
	)

	if pidString, err = ioutil.ReadFile(filename); err != nil {
		return InvalidPID, fmt.Errorf("Could not read pidfile %s: %s", filename, err)
	}

	if pid, err = strconv.Atoi(string(pidString)); err != nil {
		return InvalidPID, fmt.Errorf("Could not read pid from pidfile %s: %s", filename, err)
	}

	return pid, nil
}

// GetProcFromFile utilizes GetPidFromFile to create a os.Process handle for
// the pid contained in the given pid file.
func GetProcFromFile(filename string) (*os.Process, error) {
	var (
		pid int
		err error
	)

	if pid, err = GetPidFromFile(filename); err != nil {
		return nil, err
	}

	return os.FindProcess(pid)
}
