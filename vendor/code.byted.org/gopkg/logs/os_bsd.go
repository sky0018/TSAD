// +build darwin dragonfly freebsd netbsd openbsd

package logs

import "syscall"

const ioctlReadTermios = syscall.TIOCGETA
