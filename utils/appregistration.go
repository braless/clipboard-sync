package utils

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

// RegisterAppUserModelID 在注册表注册 AppUserModelID 的显示名与来源图标，
// 使 toast 通知在通知中心显示友好的应用名称和图标，而不是 AUMID 的裸转换名。
// 参考桌面应用 toast 通知的标准做法：
//   HKCU\Software\Classes\AppUserModelId\{AUMID} 下设置 DisplayName / IconUri。
func RegisterAppUserModelID(appID, displayName string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER,
		`Software\Classes\AppUserModelId\`+appID, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.SetStringValue("DisplayName", displayName); err != nil {
		return err
	}

	// 来源图标指向当前可执行文件（内含图标），或优先使用 notify.icon 配置
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exeAbs, err := filepath.Abs(exe)
	if err != nil {
		return err
	}

	// 来源图标优先使用 exe 同目录的 app.ico（Windows 需要图片文件路径而非 exe），否则回退 exe
	iconPath := filepath.Join(filepath.Dir(exeAbs), "app.ico")
	if _, statErr := os.Stat(iconPath); statErr != nil {
		iconPath = exeAbs
	}
	if err := key.SetStringValue("IconUri", iconPath); err != nil {
		return err
	}
	return nil
}
