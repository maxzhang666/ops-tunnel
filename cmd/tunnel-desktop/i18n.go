package main

import (
	"strings"

	"github.com/maxzhang666/ops-tunnel/internal/config"
)

var lang *string

// InitI18n binds the translation system to the config language field.
func InitI18n(cfg *config.Config) {
	lang = &cfg.General.Language
}

// T returns the translated string for the given key based on the current language.
func T(key string) string {
	msgs := messagesEN
	if lang != nil && strings.HasPrefix(*lang, "zh") {
		msgs = messagesZhCN
	}
	if v, ok := msgs[key]; ok {
		return v
	}
	if v, ok := messagesEN[key]; ok {
		return v
	}
	return key
}

var messagesEN = map[string]string{
	"close.title":    "Close OpsTunnel",
	"close.message":  "What would you like to do?",
	"close.minimize": "Minimize to Tray",
	"close.quit":     "Quit",
	"close.cancel":   "Cancel",
	"quit.title":     "Tunnels Running",
	"quit.message":   "%d tunnel(s) are still running. Quit anyway?",
	"tray.tooltip":   "OpsTunnel - SSH Tunnel Manager",
	"tray.startAll":  "Start All",
	"tray.stopAll":   "Stop All",
	"tray.start":     "Start",
	"tray.stop":      "Stop",
	"tray.restart":   "Restart",
	"tray.show":      "Show Window",
	"tray.quit":      "Quit",
	"tray.copy":      "Copy %s",
}

var messagesZhCN = map[string]string{
	"close.title":    "关闭 OpsTunnel",
	"close.message":  "您想要做什么？",
	"close.minimize": "最小化到托盘",
	"close.quit":     "退出",
	"close.cancel":   "取消",
	"quit.title":     "隧道运行中",
	"quit.message":   "%d 个隧道仍在运行，确定退出吗？",
	"tray.tooltip":   "OpsTunnel - SSH 隧道管理器",
	"tray.startAll":  "全部启动",
	"tray.stopAll":   "全部停止",
	"tray.start":     "启动",
	"tray.stop":      "停止",
	"tray.restart":   "重启",
	"tray.show":      "显示窗口",
	"tray.quit":      "退出",
	"tray.copy":      "复制 %s",
}
