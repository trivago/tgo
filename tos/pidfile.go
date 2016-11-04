package tgo

import (
	"io/ioutil"
	"os"
	"strconv"
)

// WritePidFile writes this process' process id into a file.
// An existing file will be overwritten.
func WritePidFile(filename string) error {
	pid := os.GetPid()
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
		pid  int
		proc *os.Process
		err  error
	)

	if pid, err = getPid(app, ver); err != nil {
		return nil, err
	}

	return os.FindProcess(pid)
}
