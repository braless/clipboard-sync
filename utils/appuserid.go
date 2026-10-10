package utils

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	shell32DLL            = windows.NewLazySystemDLL("shell32.dll")
	procSetAppUserModelID = shell32DLL.NewProc("SetCurrentProcessExplicitAppUserModelID")
)

// SetProcessAppUserModelID 为当前进程显式设置 AppUserModelID。
// 设置后，应用的托盘气泡通知会作为可追溯来源进入 Windows 通知中心（Action Center），
// 否则传统 balloon 通知只弹气泡、不会留存在通知中心。
func SetProcessAppUserModelID(appID string) error {
	p, err := syscall.UTF16PtrFromString(appID)
	if err != nil {
		return err
	}
	r1, _, e1 := procSetAppUserModelID.Call(uintptr(unsafe.Pointer(p)))
	if r1 != 0 {
		if e1 != syscall.Errno(0) {
			return e1
		}
		return syscall.Errno(r1)
	}
	return nil
}
