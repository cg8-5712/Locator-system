# Locator 开发与协作约束

## 项目定位

Locator 不是一个地图演示项目，而是一个面向 GPS + 4G 定位器的商业定位平台简化版。

目标是先做出一套可持续扩展的定位平台基础能力，产品形态参考：

- Traccar
- 途强类 GPS 平台
- 咕咚物联类设备平台
- 爱车安类车辆定位平台

第一阶段只做核心闭环，不做大而全。

## 第一阶段目标

必须完成以下能力：

- 用户登录与 JWT 鉴权
- 用户管理与基础角色控制
- 设备管理
- MQTT 设备接入
- 实时定位
- 历史轨迹查询
- 电子围栏
- 告警系统
- 地图展示

第一阶段明确不做：

- 多租户计费
- 工单系统
- 复杂组织架构
- 复杂权限矩阵
- 设备指令编排平台
- BI 报表中心
- 移动端 App

## 技术栈

### 后端

- Go 1.25
- Gin
- GORM
- PostgreSQL
- Redis
- EMQX
- WebSocket

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

## 架构原则

1. 平台优先，不写成单页地图 Demo。
2. 设备接入、业务规则、实时推送必须分层。
3. GPS 数据写入与实时消息推送要解耦。
4. 历史轨迹、围栏、告警三类能力必须围绕 `gps_records` 统一建模。
5. 所有接口默认面向后续多用户、多设备扩展设计。
6. 第一阶段优先保证正确性和可维护性，再考虑高并发优化。

## 建议目录

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

## 后端模块边界

### `internal/api`

负责 HTTP 路由、请求参数校验、响应结构统一，不直接写业务逻辑。

### `internal/service`

负责业务编排，例如：

- 登录与用户鉴权
- 设备管理
- 轨迹查询
- 围栏管理
- 告警生成

### `internal/repository`

负责数据库读写，不掺杂 HTTP 或 MQTT 逻辑。

### `internal/mqtt`

负责连接 EMQX、订阅主题、解析设备上报消息、调用业务服务。

### `internal/websocket`

负责浏览器实时推送，包括：

- 设备位置更新
- 设备状态更新
- 告警推送

### `internal/fence`

负责电子围栏算法与越界判断。这里保持纯逻辑，不依赖 HTTP 或数据库细节。

### `internal/alarm`

负责告警类型、告警生成规则、告警去重与告警状态处理。

### `internal/auth`

负责 JWT、登录态、角色判断、中间件。

### `internal/task`

负责定时任务，例如离线检测、告警扫描、历史数据归档。

## 前端模块边界

前端不是简单页面堆砌，至少分为以下层次：

- `pages`：页面级视图
- `features`：按业务领域拆分，例如 `auth`、`devices`、`tracks`、`fences`、`alarms`
- `components`：可复用 UI 组件
- `services`：接口封装、WebSocket 客户端封装
- `stores`：登录态、当前设备、地图筛选条件等状态管理
- `lib`：工具函数、时间格式化、地图坐标转换、请求实例

前端必须保证：

- 桌面端优先，兼容平板宽度
- 地图页支持设备列表联动
- 实时定位与历史轨迹页面分离
- shadcn/ui 只作为基础组件，不把整个产品做成默认后台模板风格

## 数据模型要求

### 用户

角色第一版只支持：

- `admin`
- `user`

### 设备

设备唯一标识使用 `imei`，所有接入链路都围绕 `imei` 建立映射。

### GPS 记录

`gps_records` 是全系统最大表，后续实现时必须优先考虑：

- 按时间范围查询
- 按设备查询
- 组合索引
- 未来可分区

### 围栏

围栏第一阶段仅支持多边形围栏，`polygon` 使用 JSONB 存储顶点数组。

### 告警

第一阶段支持以下告警类型：

- `offline`
- `out_of_fence`
- `low_battery`
- `sos`
- `overspeed`

## MQTT 约定

Broker 使用 EMQX。

主题规范：

- 设备定位上报：`device/{imei}/gps`
- 设备状态上报：`device/{imei}/status`
- 设备告警上报：`device/{imei}/alarm`
- 服务器下发命令：`device/{imei}/cmd`

服务启动后默认订阅：

- `device/+/gps`
- `device/+/status`
- `device/+/alarm`

GPS 消息标准流程：

1. MQTT 收到消息
2. 解析 JSON
3. 根据 `imei` 找设备
4. 写入 `gps_records`
5. 更新设备最后在线时间与电量
6. 执行围栏检测
7. 执行告警规则
8. 推送 WebSocket 实时消息

## WebSocket 约定

浏览器连接入口：

- `/ws`

至少支持以下事件：

- 实时定位推送
- 设备状态推送
- 告警推送

消息格式保持轻量，前端地图实时更新不依赖二次轮询。

## API 设计约束

第一阶段至少包含以下接口域：

- `/api/auth`
- `/api/users`
- `/api/devices`
- `/api/tracks`
- `/api/fences`
- `/api/alarms`

要求：

- 统一返回结构
- 统一错误码
- 所有受保护接口走 JWT 中间件
- 列表接口默认支持分页
- 时间查询参数统一使用 ISO8601 或 RFC3339

## 地图能力要求

地图 SDK 使用 Leaflet。

必须支持：

- 设备实时位置展示
- 多设备点位展示
- 单设备历史轨迹折线
- 围栏绘制与展示
- 告警点定位

第一阶段不强制做：

- 热力图
- 聚合轨迹分析
- 路况图层
- 矢量切片服务

## Go 学习协作要求

这个项目同时是 Go 学习项目。后续在实现后端时，协作方式必须满足以下要求：

1. 每完成一个 Go 模块，要说明它对应的语言知识点。
2. 每次新增关键代码时，要解释代码在做什么、为什么这么写、常见错误在哪里。
3. 每次出现 Go 报错或逻辑 Bug，要顺带解释如何定位问题。
4. 优先让你建立读代码能力，再扩展到改代码和写简单代码。
5. 生命周期和 `context` 必须作为重点主线，不作为边角知识带过。

后续讲解范围至少覆盖：

- 包、文件、模块与导入关系
- 结构体、接口、方法
- 指针和值语义
- 错误处理
- 切片、映射、零值
- 并发基础
- HTTP 服务生命周期
- `context.Context` 的传递、取消、超时和边界

## Go 代码教学约束

后续编写 Go 代码时遵守以下规则：

- 尽量保持函数短小，方便阅读和讲解
- 明确区分 handler、service、repository 的责任
- 尽量返回具体错误，避免只返回模糊字符串
- 优先用标准库思维解释问题，再引入框架用法
- 在关键链路上说明 `context` 从哪里来、传到哪里、什么时候结束
- 讲清楚 goroutine 什么时候启动、什么时候退出、谁负责回收

## 生命周期与 Context 专项要求

后续实现中，所有以下场景都要显式讲解生命周期和 `context`：

- HTTP 请求进入到返回响应
- 数据库查询的超时和取消
- MQTT 客户端启动、订阅、重连和关闭
- WebSocket 连接建立、消息循环、断开
- 定时任务启动与优雅退出
- 整个服务的启动、运行、关闭

要重点讲清楚：

- 谁创建 context
- 谁派生 context
- 谁负责 cancel
- context 的超时边界是否合理
- 什么时候不应该把 context 存在结构体里

## 开发顺序

严格按以下顺序推进，避免过早进入复杂模块：

### 第一周

- Go 服务骨架
- Gin
- PostgreSQL
- JWT
- 用户登录
- 用户管理

### 第二周

- 设备管理
- MQTT 接入
- 保存 GPS 数据

### 第三周

- React 前端基础框架
- 地图页
- WebSocket
- 实时定位

### 第四周

- 历史轨迹查询
- 轨迹回放

### 第五周

- 电子围栏

### 第六周

- 告警系统

### 第七周

- Docker Compose 部署
- Nginx 反向代理
- 压力测试

## 完成定义

一个功能只有在满足以下条件后才算完成：

1. 接口可用
2. 前后端联调通过
3. 至少有基础异常处理
4. 至少有最小可验证测试或手工验证路径
5. 文档同步更新
6. 如果涉及 Go 核心知识点，要同步补充讲解

## 实现注意事项

- 不要在 handler 里直接写 GORM 查询
- 不要把围栏算法散落在多个模块
- 不要让 WebSocket 消息结构频繁变动
- 不要把地图页做成静态假数据展示
- 不要在第一阶段引入过多微服务拆分
- 不要一开始就做复杂权限系统
- 不要只给代码不解释 Go 知识点

## 质量底线

- 核心链路必须有日志
- MQTT 消息解析失败必须可追踪
- 告警生成必须避免明显重复刷屏
- 历史轨迹查询必须支持时间范围过滤
- 所有时间字段统一时区策略，后端建议存 UTC
- 重要配置必须从环境变量或配置文件读取
- 涉及并发或 context 的代码必须讲清退出路径
