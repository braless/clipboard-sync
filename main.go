// Copyright 2011 The Walk Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/YanxinTang/clipboard-online/action"
	"github.com/YanxinTang/clipboard-online/utils"
	"github.com/gin-gonic/gin"
	"github.com/lxn/walk"
	"github.com/sirupsen/logrus"
)

var app *Application

var execPath string
var execFullPath string
var config *Config
var mode string = "debug"
var version string = ""
var log = logrus.New()

// AppUserModelID 使通知进入 Windows 通知中心的进程标识
const appUserModelID = "ClipboardOnline.App"

// notifyDisplayName 返回通知中心中显示的应用名称，取 config.appName 配置，
// 未配置时回退为 "clipboard-online"。
func notifyDisplayName() string {
	if config.AppName != "" {
		return config.AppName
	}
	return "clipboard-online"
}

// notifyIconImage 自定义通知左侧图标（nil 时使用默认图标）
var notifyIconImage walk.Image

func init() {
	execFullPath = os.Args[0]
	execPath = filepath.Dir(execFullPath)

	var err error
	configFilePath := filepath.Join(execPath, ConfigFile)
	config, err = loadConfig(configFilePath)
	if err != nil {
		log.WithError(err).Warn("failed to load config, fallback to default")
		config = &DefaultConfig
	}
	log.SetLevel(config.LogLevel)

	if mode == "debug" {
		log.SetFormatter(&logrus.TextFormatter{ForceColors: true})
	} else {
		logFilePath := filepath.Join(execPath, LogFile)
		f, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.WithError(err).Fatal("failed to open log file")
		}
		log.SetOutput(f)
		gin.SetMode(gin.ReleaseMode)
	}
}

func main() {
	var err error

	app, err = NewApplication(config)
	if err != nil {
		log.WithError(err).Fatal("failed to create applicaton")
	}
	defer app.BeforeExit()

	if err := utils.CreateDirectory(app.GetTempFilePath("")); err != nil {
		log.WithError(err).Fatal("failed to create temp directory")
	}

	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		log.WithError(err).Fatal("failed to get icon")
	}

	if err := app.ni.SetIcon(icon); err != nil {
		log.WithError(err).Fatal("failed to set icon")
	}

	if err := app.ni.SetToolTip(notifyDisplayName() + " :" + config.Port); err != nil {
		log.WithError(err).Fatal("failed to set tooltip")
	}

	// 设置 AppUserModelID，让通知进入系统通知中心
	if err := utils.SetProcessAppUserModelID(appUserModelID); err != nil {
		log.WithError(err).Warn("failed to set AppUserModelID")
	}
	// 注册 AUMID 的显示名与来源图标，避免通知中心显示裸 AUMID 名并丢失图标
	if err := utils.RegisterAppUserModelID(appUserModelID, notifyDisplayName()); err != nil {
		log.WithError(err).Warn("failed to register AppUserModelID display info")
	}
	// 在开始菜单创建快捷方式（含 AppUserModelID），为 toast 通知提供来源图标与持久化支持
	if err := utils.CreateStartMenuShortcut("clipboard-online.lnk", appUserModelID); err != nil {
		log.WithError(err).Warn("failed to create start menu shortcut")
	}
	// 确保程序目录存在通知左侧图标，并让 config 指向它（toast 显示对应图标）
	ensureNotificationIcon()
	notifyIconImage = loadNotifyIcon()

	autoRunAction, err := action.NewAutoRunAction()
	if err != nil {
		log.WithError(err).Fatal("failed to create AutoRunAction")
	}
	restartAction, err := action.NewRestartAction()
	if err != nil {
		log.WithError(err).Fatal("failed to create RestartAction")
	}
	exitAction, err := action.NewExitAction()
	if err != nil {
		log.WithError(err).Fatal("failed to create ExitAction")
	}
	if err := app.AddActions(autoRunAction, restartAction, exitAction); err != nil {
		log.WithError(err).Fatal("failed to add action")
	}

	if err := app.ni.SetVisible(true); err != nil {
		log.WithError(err).Fatal("failed to set notify visible")
	}

	log.Debug("start http server")
	app.RunHTTPServer()
	log.Debug("start app")
	app.Run()
}

// ensureNotificationIcon 确保程序目录存在通知左侧图标（icon.png），
// 并让 config.Notify.Icon 指向它，使 toast 通知左侧显示对应图标。
func ensureNotificationIcon() {
	iconPath := filepath.Join(execPath, "icon.png")
	if !utils.IsExistFile(iconPath) {
		if err := utils.WriteNotificationIcon(iconPath); err != nil {
			log.WithError(err).Warn("failed to generate notification icon")
			return
		}
	}
	if config.Notify.Icon == "" {
		config.Notify.Icon = "icon.png"
	}
}

// loadNotifyIcon 加载 config.json 中 notify.icon 指定的通知左侧图标。
// 未配置或加载失败时返回 nil，通知将使用默认图标。
func loadNotifyIcon() walk.Image {
	iconPath := config.Notify.Icon
	if iconPath == "" {
		return nil
	}
	if !filepath.IsAbs(iconPath) {
		iconPath = filepath.Join(execPath, iconPath)
	}

	var img walk.Image
	var err error
	switch strings.ToLower(filepath.Ext(iconPath)) {
	case ".ico":
		img, err = walk.NewIconFromFile(iconPath)
	default:
		img, err = walk.NewImageFromFile(iconPath)
	}
	if err != nil {
		log.WithField("iconPath", iconPath).Warn("failed to load notify icon, fallback to default")
		return nil
	}
	log.WithField("iconPath", iconPath).Info("loaded notify icon")
	return img
}
