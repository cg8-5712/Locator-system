# Locator

Locator 是一个面向 GPS + 4G 定位器的 Web 定位平台，目标是实现一套商业定位平台的简化版，而不是单纯的地图展示 Demo。

第一阶段将覆盖用户登录、设备管理、实时定位、历史轨迹、电子围栏、告警系统、MQTT 设备接入和地图展示，整体形态参考 Traccar、途强类 GPS 平台和物联网定位 SaaS。

这个项目还有第二个目标：把它作为你的 Go 学习主线。我后续会在实现后端的同时，持续解释 Go 语言基础、代码阅读方法、常见错误、生命周期和 `context`。

## 技术栈

### 后端

- Go 1.25
- Gin
- GORM
- SQLite / PostgreSQL
- Redis
- EMQX

### 前端

- React
- TypeScript
- Vite
- Tailwind CSS
- shadcn/ui
- Leaflet

### 部署

- Docker Compose
- Nginx

## 核心功能

- 用户登录与 JWT 鉴权
- 用户管理
- 设备管理
- MQTT 设备接入
- 实时定位
- 历史轨迹查询与回放
- 电子围栏
- 告警系统
- 地图可视化

## 项目目标

第一阶段重点是打通完整业务闭环：

1. 设备通过 MQTT 上报定位数据
2. 后端解析并写入 PostgreSQL
3. 围栏与告警规则同步执行
4. 实时定位通过 WebSocket 推送给前端
5. 前端在地图上展示实时位置、历史轨迹和围栏

这意味着项目重点是平台能力，不是静态地图页面。

## Go 学习目标

通过这个项目，逐步达到以下能力：

1. 能读懂常见 Go 项目结构和后端代码流转。
2. 能识别常见 Go 错误，包括空指针、错误未处理、切片误用、goroutine 泄漏和 `context` 使用错误。
3. 能独立写简单 Go 代码，包括结构体、方法、接口、HTTP handler、基础数据库读写和简单并发。
4. 能深入理解一个 Go 服务的生命周期，以及 `context.Context` 在请求链路、超时控制和优雅退出中的作用。

详细学习路线见 [GO_LEARNING.md](GO_LEARNING.md)。

## 建议目录结构

```text
Locator/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── api/
│   ├── service/
│   ├── repository/
│   ├── model/
│   ├── mqtt/
│   ├── websocket/
│   ├── fence/
│   ├── alarm/
│   ├── auth/
│   ├── task/
│   └── config/
├── pkg/
│   ├── jwt/
│   ├── logger/
│   └── geo/
├── migrations/
├── web/
│   ├── src/
│   │   ├── app/
│   │   ├── pages/
│   │   ├── components/
│   │   ├── features/
│   │   ├── hooks/
│   │   ├── lib/
│   │   ├── services/
│   │   ├── stores/
│   │   ├── types/
│   │   └── styles/
│   ├── public/
│   ├── index.html
│   ├── package.json
│   └── vite.config.ts
├── configs/
├── docker/
├── docker-compose.yml
├── CODEX.md
├── GO_LEARNING.md
└── README.md
```

## 系统架构

### 后端职责

- `api`：HTTP 路由、鉴权中间件、请求校验、统一响应
- `service`：业务编排
- `repository`：数据库访问
- `mqtt`：EMQX 连接、订阅与消息处理
- `websocket`：实时推送
- `fence`：电子围栏算法
- `alarm`：告警规则与落库
- `auth`：JWT 与权限控制
- `task`：定时任务，如离线检测

### 前端职责

- `pages`：页面入口，如登录页、设备页、地图页、告警页
- `features`：按业务拆分，如 `auth`、`devices`、`tracks`、`fences`、`alarms`
- `components`：通用组件
- `services`：HTTP 客户端、WebSocket 客户端、接口封装
- `stores`：登录态、设备筛选、地图状态
- `lib`：公共工具函数

## 数据库设计

当前后端默认支持两种数据库模式：

- 本地开发默认使用 SQLite，零依赖启动，数据库文件默认是 `locator.db`
- 生产或联调环境可切换到 PostgreSQL，通过 `DB_DRIVER=postgres` 和 `DB_DSN=...` 配置
- 服务启动时会使用 GORM 自动迁移基础表结构

### `users`

```sql
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username TEXT UNIQUE,
    password_hash TEXT,
    role VARCHAR(16) NOT NULL DEFAULT 'user',
    created_at TIMESTAMP
);
```

### `devices`

```sql
CREATE TABLE devices (
    id BIGSERIAL PRIMARY KEY,
    imei VARCHAR(32) UNIQUE,
    name VARCHAR(64),
    status INT,
    battery INT,
    last_online TIMESTAMP,
    created_at TIMESTAMP
);
```

### `gps_records`

这是核心大表，后续建议按设备和时间维度优化索引，必要时做分区。

```sql
CREATE TABLE gps_records (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT,
    latitude DOUBLE PRECISION,
    longitude DOUBLE PRECISION,
    speed REAL,
    altitude REAL,
    gps_time TIMESTAMP,
    created_at TIMESTAMP
);
```

建议至少建立：

```sql
CREATE INDEX idx_gps_records_device_time
ON gps_records (device_id, gps_time DESC);
```

### `fences`

```sql
CREATE TABLE fences (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT,
    name VARCHAR(64),
    polygon JSONB,
    created_at TIMESTAMP
);
```

`polygon` 示例：

```json
[
  [39.90, 116.30],
  [39.91, 116.31],
  [39.92, 116.32]
]
```

### `alarms`

```sql
CREATE TABLE alarms (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT,
    type VARCHAR(32),
    content TEXT,
    created_at TIMESTAMP
);
```

## MQTT 设计

### Broker

- EMQX

### Topic 规范

- 定位数据：`device/{imei}/gps`
- 状态数据：`device/{imei}/status`
- 设备告警：`device/{imei}/alarm`
- 服务器指令：`device/{imei}/cmd`

### GPS 消息示例

```json
{
  "lat": 39.90123,
  "lng": 116.31234,
  "speed": 42.5,
  "battery": 86,
  "timestamp": 1750000000
}
```

### 订阅主题

```text
device/+/gps
device/+/status
device/+/alarm
```

### 消息处理流程

```text
MQTT
  ↓
解析 JSON
  ↓
写入 gps_records
  ↓
更新 devices.last_online / battery
  ↓
电子围栏检测
  ↓
告警规则检测
  ↓
WebSocket 推送前端
```

## WebSocket 设计

前端连接入口：

```text
/ws
```

实时位置消息示例：

```json
{
  "device": "86888888",
  "lat": 39.90,
  "lng": 116.31
}
```

建议后续扩展为带事件类型的统一结构，例如：

```json
{
  "type": "location",
  "data": {
    "device": "86888888",
    "lat": 39.90,
    "lng": 116.31,
    "speed": 42.5,
    "time": "2026-06-15T10:00:00Z"
  }
}
```

## 电子围栏

围栏模块建议单独放在 `internal/fence`。

核心接口可设计为：

```go
func CheckFence(lat float64, lng float64, polygon [][]float64) bool
```

算法使用：

- Point In Polygon
- 射线法

处理流程：

```text
收到定位
  ↓
查询设备围栏
  ↓
判断点是否在围栏内
  ↓
若越界则生成 out_of_fence 告警
```

## 告警系统

第一阶段支持以下告警类型：

- `offline`
- `out_of_fence`
- `low_battery`
- `sos`
- `overspeed`

### 超速告警

```go
if speed > 80 {
    CreateAlarm(...)
}
```

### 离线告警

通过定时任务每分钟检测一次：

- 若设备 `last_online` 超过 5 分钟
- 则生成 `offline` 告警

告警系统第一版需要注意去重，避免设备持续离线时每分钟刷一条重复告警。

## 地图设计

推荐使用 Leaflet，原因如下：

- 免费
- 轻量
- 开源
- 接入简单

地图页至少支持：

- 设备实时位置
- 历史轨迹
- 电子围栏展示
- 告警点展示

## 历史轨迹接口

```http
GET /api/device/:id/tracks
```

查询参数：

- `start_time`
- `end_time`

返回示例：

```json
[
  {
    "lat": 39.9,
    "lng": 116.3,
    "time": "2026-06-15T10:00:00Z"
  }
]
```

前端地图使用 `Polyline` 绘制轨迹，轨迹回放可在此基础上做时间进度控制。

## 认证与权限

第一版角色只保留：

- `admin`
- `user`

登录接口：

```http
POST /api/login
```

返回示例：

```json
{
  "token": "xxxxx"
}
```

建议后续统一调整为：

```text
POST /api/auth/login
```

以便接口按业务域拆分。

## 建议 API 模块

- `POST /api/auth/login`
- `GET /api/users`
- `POST /api/users`
- `GET /api/devices`
- `POST /api/devices`
- `GET /api/devices/:id`
- `PUT /api/devices/:id`
- `GET /api/devices/:id/tracks`
- `GET /api/fences`
- `POST /api/fences`
- `GET /api/alarms`

## 开发顺序

### 第 1 周

- 搭建 Go 服务骨架
- 接入 Gin
- 接入 PostgreSQL
- 实现 JWT
- 完成用户登录
- 完成用户管理

### 第 2 周

- 完成设备管理
- 接入 MQTT
- 保存 GPS 数据

### 第 3 周

- 搭建 React 前端基础框架
- 完成地图页
- 接入 WebSocket
- 实现实时定位

### 第 4 周

- 完成历史轨迹查询
- 完成轨迹回放

### 第 5 周

- 完成电子围栏

### 第 6 周

- 完成告警系统

### 第 7 周

- 完成 Docker Compose 部署
- 配置 Nginx
- 进行压力测试

## 项目驱动的 Go 学习节奏

项目每推进一步，对应补足一组 Go 能力：

1. 服务骨架阶段：学习包结构、`main` 函数、依赖初始化、错误处理。
2. HTTP 阶段：学习 handler、结构体、方法、JSON 绑定、接口分层。
3. 数据库阶段：学习 repository 模式、指针和值、错误包装、超时控制。
4. MQTT 和 WebSocket 阶段：学习 goroutine、channel 基础、连接生命周期、资源释放。
5. 定时任务阶段：学习服务生命周期、优雅退出、`context.Context` 传播。
6. 围栏和告警阶段：学习如何把业务逻辑和框架代码拆开，提升代码阅读和排错能力。

## Docker Compose 规划

计划包含以下服务：

- `postgres`
- `redis`
- `emqx`
- `backend`
- `frontend`
- `nginx`

后续可在 `docker-compose.yml` 中统一管理本地开发与测试环境。

## 开发约束

- 不要把项目做成假数据地图页面
- 不要在 HTTP handler 中直接写复杂业务逻辑
- 不要一开始引入微服务拆分
- 不要先做围栏和告警，必须按开发顺序推进
- 所有时间字段建议统一使用 UTC 存储
- 核心链路必须保留日志和错误追踪
- 后续每次写 Go 代码，都要同步说明关键语法点和常见坑

## 下一步建议

如果继续往下做代码，建议按这个顺序开始初始化仓库：

1. 搭建 Go 后端骨架与配置系统
2. 初始化 PostgreSQL、Redis、EMQX 的 Docker Compose
3. 创建用户、设备、GPS、围栏、告警的 migration
4. 初始化 React + Vite + Tailwind + shadcn/ui 前端
5. 先打通登录、设备管理、MQTT 入库、地图实时点位四条主链路
