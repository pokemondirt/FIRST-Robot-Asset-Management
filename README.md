# FIRST 机器人物资出入库系统 / FIRST Robotics Inventory

FRC + FTC 共用 · 扫码枪自助出入库

## 技术栈

| 部分 | 技术 |
|------|------|
| 后端 | **Go**（`backend-go/`，SQLite） |
| 前端 | **Vue 3** + Vite + vue-i18n（`frontend/`） |

> 旧版 Python（FastAPI）后端已移除，仅保留 Go + Vue。

## 环境要求

- Windows 10+
- [Go 1.22+](https://go.dev/dl/)
- [Node.js 18+](https://nodejs.org/)（开发前端或构建静态页）

## 快速启动（开发）

在项目根目录双击 `start.cmd`，或：

```powershell
.\start.cmd
```

浏览器会打开 **http://127.0.0.1:5173/#/checkout**。

- 后端 API：`http://127.0.0.1:8000`
- 前端开发服：`5173`（通过 Vite 代理访问 `/api`）

### 为什么入口是 .cmd 而不是 .ps1

Windows 默认执行策略（`RemoteSigned`）会拒绝运行未签名的 `.ps1`；个别目录还会让 PowerShell
直接报 `AuthorizationManager check failed`，即使策略本身允许该文件。`.cmd` 启动器内部用
`-ExecutionPolicy Bypass` 调用 PowerShell，两种情况都能正常启动，也不需要修改系统设置。

如果确实想直接跑 `.ps1`：

```powershell
powershell -ExecutionPolicy Bypass -File .\start.ps1
```

可用启动器：

| 启动器 | 作用 |
|--------|------|
| `start.cmd` | 开发模式：Go 后端 + Vite 开发服 |
| `start-one-port.cmd` | 构建前端并以单端口 8000 运行 |
| `start-go.cmd` | 仅启动 Go 后端 |
| `build-frontend.cmd` | 构建前端到 `frontend/dist` |
| `build-release.cmd` | 打跨平台发布包 |

仅启动前端开发服：`powershell -ExecutionPolicy Bypass -File .\start-frontend.ps1`

## 单端口部署（仓库电脑）

```powershell
.\build-frontend.cmd
.\start-go.cmd
```

浏览器：**http://127.0.0.1:8000/#/checkout**

## 一键安装包（目标电脑无需 Go / Node.js）

发布包是自包含的：后端是**静态编译的单个可执行文件**（SQLite 驱动 `modernc.org/sqlite`
是纯 Go 实现，不需要 CGO、不需要任何 DLL），前端是构建好的静态文件，由同一个进程托管。

```powershell
.\build-release.ps1                            # 默认 windows/amd64 + windows/arm64
.\build-release.ps1 -Targets windows/amd64     # 只出一个目标
.\build-release.ps1 -Targets windows/amd64,linux/amd64,darwin/arm64
```

产物在 `scripts/release/out/FIRST-Inventory-<os>-<arch>.zip`。把它拷到目标电脑：

1. 解压到任意目录（建议 `D:\FIRST-Robot-Inventory`）
2. 双击 `start.bat`（或先运行 `install-desktop.bat` 建桌面快捷方式）
3. 首次启动 Windows 防火墙会询问，勾选「专用网络」并允许

**前端已经编进 exe 里**，包里没有 `dist` 目录：`inventory-server.exe` 一个文件就能跑，
双击它和在 Go 里 `go build` 出来的是同一个东西。`build-frontend.cmd` 会把 `frontend/dist`
拷到 `backend-go/webembed/dist` 再编进去。

数据库在解压目录的 `data\inventory.db`（首次运行自动创建），**升级时只替换 exe、保留 `data\` 即可**。

> 本机 Go 工具链是 32 位的（`GOARCH=386`），所以脚本里始终显式指定目标架构；
> 否则会不小心发出一个 32 位版本。

> 开发时如果想改完前端立刻生效、不重新编译 exe：把前端跑在 Vite（`start.cmd`）或先
> `build-frontend.cmd` 再 `start-go.cmd`。服务器会**优先使用磁盘上的 `frontend/dist`**，
> 找不到才用 exe 里内置的那份。

## 局域网共用

服务**有意监听所有网卡**（`:8000`），同一局域网的其他电脑直接用浏览器访问即可，
不需要在每台电脑上安装任何东西。启动时日志会打印可直接分享的地址：

```
Server starting on http://127.0.0.1:8000
  On this network: http://192.168.1.23:8000/#/checkout
```

注意：**系统没有登录功能**，能访问到该地址的人都能修改库存。请只在可信的仓库内网使用，
不要做端口转发或暴露到公网。跨站脚本调用已被拦截（见 `guardMiddleware`），
但这只防浏览器被第三方网页利用，不防局域网内的直接访问。

## 扫码与输入法

扫码枪在系统里等同于键盘。如果 Windows 当前是**中文输入法**，扫进来的字符会进入拼音组字，
轻则查不到条码，重则把组字内容当成条码提交。

页面做了两层防护：

1. **自动丢弃**：扫码框一旦检测到输入法组字（`compositionstart`）或非 ASCII 字符，
   会立即清空输入并红字提示「检测到中文输入法，这次输入已丢弃」，**不会把错误内容发给服务器**。
2. **提示常驻**：扫码框下方有一行小字，提醒扫码前确认输入法是英文（`Shift` 切换）。

如果你希望**真的自动切换**——注意：网页无法修改系统输入法（浏览器没有这个 API），
所以额外提供了一个可选的 Windows 助手：

```powershell
scripts\ime\start-ime-helper.cmd
```

它只在「出入库」页的扫码框获得焦点时把输入法切到英文，光标离开就停止干预；`Ctrl+C` 停止。
原理是页面把"扫码框聚焦"通过窗口标题里的一个零宽字符发布出来，助手轮询后调用
`ImmSetConversionStatus`。个别浏览器/输入法组合下可能不生效，此时页面自身的防护仍然有效。

不想跑助手的话，也可以在 Windows 里做一次性设置：**设置 → 时间和语言 → 输入 →
高级键盘设置 → 勾选「允许我为每个应用窗口使用不同的输入法」**，然后在浏览器里切一次英文，
浏览器就会记住英文状态（其它程序仍是中文）。

## 发布到 GitHub Releases

推送一个 tag 即可由 GitHub Actions 自动构建全部平台的压缩包并挂到 Release 上：

```powershell
git tag v1.0.0
git push origin v1.0.0
```

工作流见 [`.github/workflows/release.yml`](.github/workflows/release.yml)；在 Actions 页面手动
触发（workflow_dispatch）则只产出可下载的构建产物，不建 Release。
每次 push / PR 会跑 [`.github/workflows/ci.yml`](.github/workflows/ci.yml)：`gofmt`、`go vet`、
`go test` 与前端构建。

## 功能

- 入库 / 出库（扫码枪直扫即生效，归还按入库处理）
- 展板：今日入库、库存预警（`#/board`）
- 物资档案、分类管理、操作人
- CSV 导入 / 导出
- 数据库备份
- 中英界面

## 数据

- 数据库：`backend-go/data/inventory.db`
- 备份：`backend-go/data/backups/`（设置页可触发）

## 项目结构

```
backend-go/     Go HTTP API，webembed/ 里内嵌构建好的前端
frontend/       Vue 3 界面
start.ps1       开发：Go + Vite
start-go.ps1    仅 Go 后端
build-frontend.ps1  构建前端并写入 backend-go/webembed/dist
build-release.ps1   交叉编译出单文件免安装发布包（scripts/release/out/）
scripts/release/    发布包模板（start.bat / open-browser.ps1 / 说明）
scripts/ime/        可选：扫码时自动切英文输入法
.github/workflows/  CI 与自动发布
LICENSE             MIT
CHANGELOG.md        版本变更记录
```

## 测试

```powershell
cd backend-go
gofmt -l .          # 应无输出
go vet ./...
go test ./...       # 回归测试，含接口行为与边界用例
```

前端没有单元测试，构建验证即 `npm run build`。

## API

- `GET http://127.0.0.1:8000/api/health` → `{"status":"ok"}`
