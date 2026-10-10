package main

import (
	"encoding/json"
	"io/ioutil"

	"github.com/YanxinTang/clipboard-online/utils"
	"github.com/sirupsen/logrus"
)

const ConfigFile = "config.json"
const LogFile = "log.txt"

// Config represents configuration for applicaton
type Config struct {
	Port                  string       `json:"port"`
	Authkey               string       `json:"authkey"`
	AuthkeyExpiredTimeout int64        `json:"authkeyExpiredTimeout"`
	LogLevel              logrus.Level `json:"logLevel"`
	TempDir               string       `json:"tempDir"`
	ReserveHistory        bool         `json:"reserveHistory"`
	AppName               string       `json:"appName"`
	Notify                ConfigNotify `json:"notify"`
}

type ConfigNotify struct {
	Copy  bool   `json:"copy"`
	Paste bool   `json:"paste"`
	Icon  string `json:"icon"`
}

// DefaultConfig is a default configuration for application
var DefaultConfig = Config{
	Port:                  "5678",
	Authkey:               "clipboard",
	AuthkeyExpiredTimeout: 300,
	LogLevel:              logrus.WarnLevel,
	TempDir:               "./temp",
	ReserveHistory:        true,
	AppName:               "应用通知",
	Notify: ConfigNotify{
		Copy:  true,
		Paste: true,
		Icon:  "",
	},
}

func loadConfig(path string) (*Config, error) {
	if utils.IsExistFile(path) {
		return loadConfigFromFile(path)
	}
	if err := createConfigFile(path); err != nil {
		return nil, err
	}
	return &DefaultConfig, nil
}

func loadConfigFromFile(path string) (*Config, error) {
	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(configBytes, &DefaultConfig); err != nil {
		return nil, err
	}
	return &DefaultConfig, nil
}

func createConfigFile(path string) error {
	defaultConfigJSON, err := json.MarshalIndent(DefaultConfig, "", "  ")
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(path, []byte(defaultConfigJSON), 0744); err != nil {
		return err
	}
	return nil
}
