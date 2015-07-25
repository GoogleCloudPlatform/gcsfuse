package bazilfuse

import "syscall"

const (
	ENOATTR = Errno(syscall.ENOATTR)
)

const (
	errNoXattr = ENOATTR
)

func init() {
	errnoNames[errNoXattr] = "ENOATTR"
}
