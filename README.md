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

```powershell
cd <项目目录>
.\start.ps1
```

> 若提示"未对文件进行数字签名"，说明本机执行策略较严，改用
> `powershell -ExecutionPolicy Bypass -File .\start.ps1`（其余脚本同理）。

浏览器打开：**http://127.0.0.1:5173/#/checkout**

- 后端 API：`http://127.0.0.1:8000`
- 前端开发服：`5173`（通过 Vite 代理访问 `/api`）

仅启动 Go 后端：

```powershell
.\start-go.ps1
```

仅启动前端开发服：

```powershell
.\start-frontend.ps1
```

## 单端口部署（仓库电脑）

```powershell
.\build-frontend.ps1
.\start-go.ps1
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

数据库在解压目录的 `data\inventory.db`，**升级时只替换程序文件、保留 `data\` 即可**。

> 本机 Go 工具链是 32 位的（`GOARCH=386`），所以脚本里始终显式指定目标架构；
> 否则会不小心发出一个 32 位版本。

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

- 入库 / 出库 / 归还 / 查询（扫码枪）
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
backend-go/     Go HTTP API + 可选托管 frontend/dist
frontend/       Vue 3 界面
start.ps1       开发：Go + Vite
start-go.ps1    仅 Go 后端
build-frontend.ps1
build-release.ps1   交叉编译出免安装发布包（scripts/release/out/）
scripts/release/    发布包模板（start.bat / open-browser.ps1 / 说明）
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
