<div align="center">
  <img src="https://raw.githubusercontent.com/braless/clipboard-sync/main/images/clipboard-icon.png" style="display: inline-block; vertical-align: middle;">
  <h1 style="display: inline-block; vertical-align: middle;">clipboard-online</h1>
</div>

# clipboard-online · Windows / iOS 剪切板同步

一个运行在 **Windows** 上的剪切板共享服务，让你可以**在手机（iPhone / Android）上读写电脑的剪切板**。基于 Gin HTTP 服务 + 托盘图标 + Windows Toast 通知构建。

本项目在 [clipboard-online](https://github.com/YanxinTang/clipboard-online) 原版基础上做了大量增强：新增 `/message` 白名单接口、**短信验证码自动识别并复制**、**系统 Toast 通知（进入 Windows 通知中心）**、可配置的通知显示名、通知「标记已读」按钮、通知图标等。

---

## ✨ 功能特性

| 能力 | 说明 |
| --- | --- |
| 双向剪切板同步 | 手机获取 / 设置电脑剪切板，支持文本、文件（图片/任意文件） |
| `/message` 白名单接口 | 无需认证即可投递消息通知，适合与自动化脚本联动 |
| **验证码自动复制** | 文本疑似验证码时，自动把验证码写入剪切板并显示原文通知 |
| **系统 Toast 通知** | 通知进入 Windows 通知中心，可持久查看、可被「标记已读」清除 |
| **可配置通知名** | 通知中心的应用显示名由 `appName` 配置，默认「应用通知」 |
| **通知图标** | 通知来源图标 + 正文左侧图标均可显示 |
| 通知按钮 | 通知下方带「标记已读」按钮（点击后通知从通知中心消失） |
| 托盘菜单 | 自启 / 重启 / 退出 |
| 认证保护 | `/` 接口支持基于时间窗口的 authkey 鉴权 |

---

## 🖥 环境要求

- 操作系统：Windows（需要通知中心功能，建议 Windows 10 / 11）
- 构建：Go 1.18+（仅 Windows）
- 手机：iPhone（快捷指令）或 Android（HTTP Shortcuts / Tasker）

---

## 📥 下载与构建

### 直接下载

从 GitHub Releases 下载最新 `clipboard-online.exe`。

### 从源码构建（仅 Windows）

```bash
git clone git@github.com:braless/clipboard-sync.git
cd clipboard-online
go get github.com/akavel/rsrc

# PowerShell 构建
.\build.ps1

# 或 Bash
./build.sh
```

构建产物位于 `release/` 目录，即 `release/clipboard-online.exe`。

---

## 🚀 快速开始

1. 双击运行 `release/clipboard-online.exe`（托盘图标常驻）。
2. 首次运行会在 exe 所在目录生成 `config.json` 和 `log.txt`。
3. 按需修改 `config.json`（见下方配置说明）。
4. 在手机上配置电脑的 IP 与接口。

### iOS（iPhone / iPad）

使用快捷指令读取 / 写入电脑剪切板：

- **复制（读电脑剪切板）**
  - 快捷指令：[https://www.icloud.com/shortcuts/f463a1e431c94c60b8a5c65305eb819f](https://www.icloud.com/shortcuts/f463a1e431c94c60b8a5c65305eb819f)
  - ![](./images/copy.png)
- **粘贴（写电脑剪切板）**
  - 快捷指令：[https://www.icloud.com/shortcuts/90e7a2af70df4707a17dece8c263afc5](https://www.icloud.com/shortcuts/90e7a2af70df4707a17dece8c263afc5)
  - ![](./images/paste.png)

在快捷指令中填写电脑的 IP 地址、端口与 authkey。参考：[教程](https://www.kejiwanjia.com/jiaocheng/zheteng/83030.html)

### Android

- HTTP Shortcuts：[https://meta.appinn.net/t/topic/20322](https://meta.appinn.net/t/topic/20322)
- Tasker：[https://github.com/YanxinTang/clipboard-online/issues/26](https://github.com/YanxinTang/clipboard-online/issues/26)

感谢 [@xiozf](https://github.com/xiozf)、[@Befod](https://github.com/Befod) 及各位对 Android 方案的贡献。

---

## ⚙️ 配置说明（`config.json`）

首次运行自动生成，编辑后重启程序生效（托盘菜单「重启」或重新运行 exe）。

| 字段 | 类型 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `port` | string | `"5678"` | HTTP 服务监听端口 |
| `authkey` | string | `"clipboard"` | 认证密钥（`/` 接口鉴权用） |
| `authkeyExpiredTimeout` | int64 | `300` | authkey 时间窗口（秒） |
| `logLevel` | string | `"warning"` | 日志级别：`panic/fatal/error/warning/info/debug/trace` |
| `tempDir` | string | `"./temp"` | 临时文件目录 |
| `reserveHistory` | bool | `true` | 保留历史记录 |
| `appName` | string | `"应用通知"` | **通知中心显示的应用名称** |
| `notify.copy` | bool | `true` | 复制（读）剪切板时是否通知 |
| `notify.paste` | bool | `true` | 粘贴（写）剪切板时是否通知 |
| `notify.icon` | string | `""` | 通知左侧图标（PNG 绝对/相对路径）；留空则使用程序目录自动生成的 `icon.png` |

示例：

```json
{
  "port": "5678",
  "authkey": "clipboard",
  "authkeyExpiredTimeout": 300,
  "logLevel": "warning",
  "tempDir": "./temp",
  "reserveHistory": true,
  "appName": "应用通知",
  "notify": {
    "copy": true,
    "paste": true,
    "icon": ""
  }
}
```

---

## 🔌 API 文档

服务默认监听 `5678` 端口。

### 公共请求头

| 请求头 | 必需 | 说明 |
| --- | --- | --- |
| `X-API-Version` | `/` 接口必需 | 接口版本，当前为 `1` |
| `X-Client-Name` | 可选 | 设备名称（URL 编码），用于通知中显示来源 |
| `X-Auth` | `/` 接口可选 | `md5(authkey + 时间戳/300)`，用于鉴权 |

### 1. 获取 Windows 剪切板

- URL：`/`
- Method：`GET`

响应（JSON）：

```json
// 200，文本类型
{ "type": "text", "data": "服务器上的剪切板文本" }

// 200，文件类型
{
  "type": "file",
  "data": [
    { "name": "文件名", "content": "base64 字符串" }
  ]
}
```

### 2. 设置 Windows 剪切板

- URL：`/`
- Method：`POST`
- 请求头 `X-Content-Type`：`text` / `file` / `media`

请求体（JSON）：

```json
// 文本
{ "data": "要设置到剪切板的文本" }

// 文件
{
  "data": [
    { "name": "文件名", "base64": "base64 字符串" }
  ]
}
```

响应：设置成功返回 `200`，响应体为空。

### 3. 发送消息通知（白名单，`/message`）

> 无需 `X-API-Version`、无需 auth 认证，任何设备可直接调用，适合与脚本 / 自动化联动。

- URL：`/message`
- Method：`POST`
- Content-Type：`application/json`
- 请求体：`{ "text": "消息内容" }`

行为：

- 若文本**疑似短信验证码**（本身为 4~8 位纯数字，或包含「验证码 / 动态码 / 校验码 / 安全码 / 验证代码 / 短信验证码 / verification code / verify code / captcha」等关键词），程序会**自动把验证码复制到剪切板**，并弹出原文通知。
- 否则，仅弹出原文通知，**不**写入剪切板。

示例：

```bash
curl -X POST http://<电脑IP>:5678/message \
  -H "Content-Type: application/json" \
  -d '{"text":"您的验证码是 888888，请勿泄露"}'
```

---

## 🔔 通知说明

程序使用 Windows Toast（进入通知中心），并在托盘保留传统气泡作为降级方案。

- **通知中心持久**：Toast 通知进入系统通知中心，可随时查看记录。
- **应用显示名**：由 `config.json` 的 `appName` 决定，默认「应用通知」。
- **来源图标**：程序会在开始菜单创建带 `AppUserModelID` 的快捷方式，使通知中心正确显示来源图标与应用名。
- **左侧图标**：程序启动时在 exe 目录自动生成 `icon.png`，作为通知左侧图标；也可用 `notify.icon` 指定自定义 PNG。
- **标记已读按钮**：通知下方带「标记已读」按钮，点击后通知从通知中心消失（无其他副作用）。

---

## 🖱 托盘菜单

托盘图标右键菜单包含：

- **自启**：设置/取消开机自动运行
- **重启**：重启服务
- **退出**：退出程序

---

## ❓ 常见问题

**Q：手机连不上电脑？**
确保电脑与手机在同一局域网，防火墙放行 `port` 对应的端口；iOS 快捷指令中填写的 IP 为电脑局域网 IP。

**Q：验证码没有自动复制？**
确认文本包含「验证码」等关键词或本身是 4~8 位纯数字；`/message` 接口无需认证即可触发。

**Q：通知中心不显示图标 / 名称异常？**
程序启动时需成功创建开始菜单快捷方式（含 `AppUserModelID`）；检查 `appName` 配置与 exe 所在目录的 `icon.png`。

**Q：配置文件修改后不生效？**
修改 `config.json` 后需重启程序（托盘菜单「重启」）。

---

## 原始项目

本项目基于 [https://github.com/YanxinTang/clipboard-online](https://github.com/YanxinTang/clipboard-online) 增强而来，感谢原作者。
