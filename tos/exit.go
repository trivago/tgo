package tos

import "syscall"

const (
	// ExitSuccess should be used if no error occured
	ExitSuccess = 0
	// ExitError should be used in general failure cases
	ExitError = 1
	// ExitNotAllowed should be used if a permission problem occured
	ExitNotAllowed = 2
	// ExitCannotExecute should be used if e.g. a subprocess failed to execute
	ExitCannotExecute = 126
	// ExitNotFound should be used if e.g. a subprocess executable was not found
	ExitNotFound = 127
	// ExitInvalidArgument should be used if argument parsing failed
	ExitInvalidArgument = 128
	// ExitSignal + n should be used if the signal with the number n terminated the process
	ExitSignal = 128
	// ExitSignalInt is shorthand for ExitSignal + syscall.SIGINT
	ExitSignalHup = ExitSignal + syscall.SIGHUP
	// ExitSignalInt is shorthand for ExitSignal + syscall.SIGINT
	ExitSignalInt = ExitSignal + syscall.SIGINT
	// ExitSignalKill is shorthand for ExitSignal + syscall.SIGKILL
	ExitSignalKill = ExitSignal + syscall.SIGKILL
	// ExitSignalTerm is shorthand for ExitSignal + syscall.SIGTERM
	ExitSignalTerm = ExitSignal + syscall.SIGTERM
	// ExitSignalPipe is shorthand for ExitSignal + syscall.SIGPIPE
	ExitSignalPipe = ExitSignal + syscall.SIGPIPE
	// ExitSignalAlarm is shorthand for ExitSignal + syscall.SIGALRM
	ExitSignalAlarm = ExitSignal + syscall.SIGALRM
	// ExitSignalTrap is shorthand for ExitSignal + syscall.SIGTRAP
	ExitSignalTrap = ExitSignal + syscall.SIGTRAP
)
