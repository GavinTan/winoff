package main

import (
	"syscall"
	"unsafe"

	win "github.com/CodyGuo/win"
)

//win32api
//https://docs.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-messagebox

const (
	// MessageBox flags
	HCBT_ACTIVATE  = 5
	WH_CBT         = 5
	IDYES          = 6
	IDNO           = 7
	MB_YESNOCANCEL = 0x00000003
	MB_ICONWARNING = 0x00000030
	MB_DEFBUTTON3  = 0x00000200

	// click button
	SHUTDOWN = 6
	REBOOT   = 7
	CANCEL   = 2
)

var (
	user32                 = syscall.NewLazyDLL("user32.dll")
	kernel32               = syscall.NewLazyDLL("kernel32.dll")
	winMessageBoxW         = user32.NewProc("MessageBoxW")
	winSetWindowsHookEx    = user32.NewProc("SetWindowsHookExW")
	winUnhookWindowsHookEx = user32.NewProc("UnhookWindowsHookEx")
	winGetCurrentThreadId  = kernel32.NewProc("GetCurrentThreadId")
	winSetDlgItemText      = user32.NewProc("SetDlgItemTextW")
)

type (
	DWORD     uint32
	WPARAM    uintptr
	LPARAM    uintptr
	LRESULT   uintptr
	HANDLE    uintptr
	HINSTANCE HANDLE
	HHOOK     HANDLE
	HWND      HANDLE
)

type HOOKPROC func(int, WPARAM, LPARAM) LRESULT

func MessageBox(hwnd uintptr, caption string, title string, flags uint) int {
	ret, _, _ := winMessageBoxW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(caption))),
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(title))),
		uintptr(flags),
	)
	return int(ret)
}

func SetWindowsHookEx(idHook int, lpfn HOOKPROC, hMod HINSTANCE, dwThreadId DWORD) HHOOK {
	ret, _, _ := winSetWindowsHookEx.Call(
		uintptr(idHook),
		uintptr(syscall.NewCallback(lpfn)),
		uintptr(hMod),
		uintptr(dwThreadId),
	)
	return HHOOK(ret)
}

func UnhookWindowsHookEx(hhk HHOOK) bool {
	ret, _, _ := winUnhookWindowsHookEx.Call(
		uintptr(hhk),
	)
	return ret != 0
}

func GetCurrentThreadId() (id DWORD) {
	r0, _, _ := syscall.SyscallN(winGetCurrentThreadId.Addr(), 0, 0, 0, 0)
	id = DWORD(r0)
	return
}

func GetPrivileges() {
	var hToken win.HANDLE
	var tkp win.TOKEN_PRIVILEGES

	win.OpenProcessToken(win.GetCurrentProcess(), win.TOKEN_ADJUST_PRIVILEGES|win.TOKEN_QUERY, &hToken)
	win.LookupPrivilegeValueA(nil, win.StringToBytePtr(win.SE_SHUTDOWN_NAME), &tkp.Privileges[0].Luid)
	tkp.PrivilegeCount = 1
	tkp.Privileges[0].Attributes = win.SE_PRIVILEGE_ENABLED
	win.AdjustTokenPrivileges(hToken, false, &tkp, 0, nil, nil)
}

func main() {
	h := SetWindowsHookEx(WH_CBT,
		(HOOKPROC)(func(nCode int, wparam WPARAM, lparam LPARAM) LRESULT {
			if nCode == HCBT_ACTIVATE {
				shutdownBtnName := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("关机")))
				rebootBtnName := uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("重启")))
				winSetDlgItemText.Call(uintptr(wparam), IDYES, shutdownBtnName)
				winSetDlgItemText.Call(uintptr(wparam), IDNO, rebootBtnName)
			}
			return 0
		}), 0, GetCurrentThreadId())

	mbox := MessageBox(0x00000000, "请保存当前资料!!!", "警告", MB_YESNOCANCEL|MB_ICONWARNING|MB_DEFBUTTON3)
	UnhookWindowsHookEx(h)

	switch mbox {
	case SHUTDOWN:
		GetPrivileges()
		win.ExitWindowsEx(win.EWX_SHUTDOWN, 0)
	case REBOOT:
		GetPrivileges()
		win.ExitWindowsEx(win.EWX_REBOOT, 0)
	}

}
