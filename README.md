# FIRST 机器人物资出入库系统 / FIRST Robotics Inventory

FRC + FTC 共用 · 扫码枪自助 · 璞趣 PQ00 标签（40×30 mm）

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
cd "f:\项目资料\First机器人出入库管理系统"
.\start.ps1
```

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
```

## API

- `GET http://127.0.0.1:8000/api/health` → `{"status":"ok"}`
