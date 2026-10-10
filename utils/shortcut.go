package utils

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// sysProcAttrHideWindow 隐藏子进程窗口（powershell 无控制台闪窗）
var sysProcAttrHideWindow = &syscall.SysProcAttr{HideWindow: true}

// shortcutScript 通过 PowerShell 创建开始菜单快捷方式，并写入
// System.AppUserModel.ID 属性（AppUserModelID）。在独立进程运行，
// 避免与主进程内 walk 已初始化的 COM 冲突。
const shortcutScript = `param($linkPath, $targetExe, $appId)
$ErrorActionPreference = "Stop"
$ws = New-Object -ComObject WScript.Shell
$sc = $ws.CreateShortcut($linkPath)
$sc.TargetPath = $targetExe
$sc.IconLocation = "$targetExe,0"
$sc.WorkingDirectory = (Split-Path $targetExe)
$sc.Save()
Add-Type -TypeDefinition @"
using System;
using System.Runtime.InteropServices;
[StructLayout(LayoutKind.Sequential)]
public struct PROPERTYKEY { public Guid fmtid; public uint pid; }
[StructLayout(LayoutKind.Sequential)]
public struct PROPVARIANT { public ushort vt; public ushort w1; public ushort w2; public ushort w3; public IntPtr p; }
[UnmanagedFunctionPointer(CallingConvention.StdCall)]
public delegate int SetValueDelegate(IntPtr self, ref PROPERTYKEY key, ref PROPVARIANT pv);
[UnmanagedFunctionPointer(CallingConvention.StdCall)]
public delegate int CommitDelegate(IntPtr self);
public static class S {
  [DllImport("shell32.dll", CharSet=CharSet.Unicode)]
  public static extern int SHGetPropertyStoreFromParsingName(string pszPath, IntPtr punk, uint flags, ref Guid riid, out IntPtr ppv);
}
"@
$iid = [Guid]"886D8EEB-8CF2-4446-8D02-CDBA1DBDCF99"
$ppv = [IntPtr]::Zero
$hr = [S]::SHGetPropertyStoreFromParsingName($linkPath, [IntPtr]::Zero, 2, [ref]$iid, [ref]$ppv)
if ($hr -lt 0) { throw "SHGetPropertyStoreFromParsingName failed: $hr" }
$vtbl = [Runtime.InteropServices.Marshal]::ReadIntPtr($ppv)
$setValuePtr = [Runtime.InteropServices.Marshal]::ReadIntPtr($vtbl, (6 * [IntPtr]::Size))
$commitPtr  = [Runtime.InteropServices.Marshal]::ReadIntPtr($vtbl, (7 * [IntPtr]::Size))
$setValue = [Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer($setValuePtr, [SetValueDelegate])
$commit  = [Runtime.InteropServices.Marshal]::GetDelegateForFunctionPointer($commitPtr, [CommitDelegate])
$fmtid = [Guid]"9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3"
$pkey = New-Object PROPERTYKEY
$pkey.fmtid = $fmtid
$pkey.pid = 5
$pv = New-Object PROPVARIANT
$pv.vt = 31
$pv.p = [Runtime.InteropServices.Marshal]::StringToCoTaskMemUni($appId)
$hr2 = $setValue.Invoke($ppv, [ref]$pkey, [ref]$pv)
if ($hr2 -lt 0) { throw "SetValue failed: $hr2" }
$hr3 = $commit.Invoke($ppv)
if ($hr3 -lt 0) { throw "Commit failed: $hr3" }
`

// CreateStartMenuShortcut 在开始菜单创建指向当前 exe 的快捷方式，
// 写入图标与 AppUserModelID（System.AppUserModel.ID），
// 使 Windows 通知中心能识别并显示正确的来源图标与应用名。
func CreateStartMenuShortcut(linkName, appID string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeAbs, err := filepath.Abs(exe)
	if err != nil {
		return err
	}

	startMenu := filepath.Join(os.Getenv("APPDATA"), `Microsoft\Windows\Start Menu\Programs`)
	linkPath := filepath.Join(startMenu, linkName)

	// 将脚本写入临时文件后通过 powershell 执行
	f, err := os.CreateTemp("", "co_shortcut_*.ps1")
	if err != nil {
		return err
	}
	scriptPath := f.Name()
	if _, err := f.WriteString(shortcutScript); err != nil {
		f.Close()
		os.Remove(scriptPath)
		return err
	}
	f.Close()
	defer os.Remove(scriptPath)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-linkPath", linkPath,
		"-targetExe", exeAbs,
		"-appId", appID)
	cmd.SysProcAttr = sysProcAttrHideWindow
	out, err := cmd.CombinedOutput()
	if err != nil {
		if len(out) > 0 {
			return &shortcutError{op: "powershell shortcut script", msg: string(out)}
		}
		return err
	}
	return nil
}

type shortcutError struct {
	op  string
	msg string
}

func (e *shortcutError) Error() string {
	return e.op + ": " + e.msg
}
