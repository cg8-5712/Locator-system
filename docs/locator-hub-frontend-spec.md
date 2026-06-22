# Locator Hub 前端 UI/UX 与实现规格说明书

## 1. 文档目的

本文档用于将当前项目从“通用定位平台”收敛为“人员定位与安全管理系统”的前端实现规范，目标不是展示一套概念图，而是为后续使用 React + TypeScript + Tailwind CSS + Leaflet 开发 Web 端提供可直接执行的产品、交互与工程说明。

本文档覆盖三类内容：

1. 产品与交互设计：页面结构、角色权限、视觉状态、关键流程。
2. 前端工程设计：目录分层、状态管理、地图层设计、组件拆分、数据流。
3. 前后端契约：当前后端已支持的能力、前端可直接接入的接口、为了“分享链接”必须新增的接口。

## 2. 产品定位

### 2.1 从车定位切换到人定位

系统定位从“车队监控”切换为“人员定位与安全管理”，前端表达必须同步变化：

- 地图主角从车辆变为人员。
- 强化状态感知而非速度感知。
- 强化安全态势而非运输效率。
- 强化隐私保护和临时分享，而不是后台内部查看。

### 2.2 核心使用场景

1. 管理员在地图总览页查看全员分布、在线情况、低电量、SOS、围栏越界。
2. 班组长或调度员查看某一个人员的实时位置、轨迹、围栏和当前设备状态。
3. 管理员为外部临时人员生成“实时位置分享链接”。
4. 外部访客通过分享链接进入精简地图页，仅查看被授权人员的位置。
5. 发生 SOS 时，系统全局高亮、推送、聚焦和告警提醒。

## 3. 用户角色与访问边界

### 3.1 内部角色

- `admin`
  - 查看全部人员
  - 下发设备命令
  - 管理围栏
  - 创建分享链接
  - 查看告警中心
- `user`
  - 查看授权范围内人员
  - 查看地图、轨迹、围栏、实时状态
  - 视业务规则决定是否可创建分享链接

### 3.2 外部访客角色

- 无后台账号
- 只能通过分享链接访问
- 只能看到单个被分享对象
- 不显示后台操作按钮
- 不显示系统导航、命令下发、围栏编辑、告警列表等内部能力

## 4. 信息架构与路由设计

建议前端采用如下路由结构：

```text
/login
/app/map
/app/devices/:deviceSN
/app/alarms
/app/devices/:deviceSN/history
/app/devices/:deviceSN/fences
/app/share-management
/s/:shareCode
/s/:shareCode/verify
```

说明：

- `/app/map` 是内部主工作台，默认进入该页。
- `/app/devices/:deviceSN` 是单人详情页，也可作为右侧详情抽屉的独立页版本。
- `/app/alarms` 用于集中查看离线、越界、低电量、SOS。
- `/s/:shareCode` 是公开访客页面。
- `/s/:shareCode/verify` 用于密码校验页，校验成功后进入公开地图页。

## 5. 视觉系统与品牌方向

### 5.1 整体视觉方向

前端不应做成默认后台模板风格，建议采用“城市安全调度台”风格：

- 底色：偏浅灰蓝和雾白，便于长时间盯地图。
- 主强调色：冷青蓝，表示定位、路线、在线、围栏。
- 危险色：高饱和红橙，仅用于 SOS、越界、超时离线。
- 辅助色：苔绿表示正常，琥珀表示低电量或风险预警。
- 大量使用圆角、半透明面板、模糊阴影，但避免炫技。

### 5.2 字体建议

避免直接使用默认 `Arial` 或通用系统栈，建议：

- 中文主字体：`Noto Sans SC`
- 英文与数字辅助字体：`IBM Plex Sans` 或 `Space Grotesk`

理由：

- 中文信息密度高时更稳定。
- 时间、坐标、电量、状态值在仪表型界面中需要更清晰的数字表现。

### 5.3 建议的 CSS 变量

```css
:root {
  --bg-app: #eef3f6;
  --bg-panel: rgba(255, 255, 255, 0.82);
  --bg-panel-strong: #ffffff;
  --text-primary: #10212b;
  --text-secondary: #546570;
  --line-soft: rgba(16, 33, 43, 0.08);
  --brand: #1f88c9;
  --brand-soft: #d9edf8;
  --safe: #2f9e68;
  --warn: #d48a1f;
  --danger: #d94747;
  --danger-deep: #6f1010;
  --offline: #7c8b94;
}
```

## 6. 地图对象与状态表达规范

### 6.1 人员 Marker 设计

地图上的点位不能继续使用“小车图标”，建议改为“人员身份圆章”：

- 内层：姓名首字母或头像。
- 外圈：表示安全状态。
- 边缘光晕：表示新鲜度与活动强度。
- 选中态：加粗边框和轻微缩放。

### 6.2 Marker 状态映射

建议统一如下状态映射：

| 状态 | 视觉 | 含义 |
| --- | --- | --- |
| 在线且稳定 | 绿色外圈 | 设备在线，定位正常 |
| 步行中 | 蓝绿色外圈 | 有位置移动，速度较低 |
| 跑步/快速移动 | 亮蓝外圈 | 运动速度明显 |
| 静止 | 琥珀外圈 | 长时间未移动 |
| 低电量 | 黄色脉冲点 | 设备可用但风险上升 |
| 离线 | 灰色虚线外圈 | 最后在线超过阈值 |
| SOS | 红黑闪烁外圈 | 紧急状态，需要全局关注 |

### 6.3 定位精度圈

人员定位比车辆定位更容易被质疑“点位准不准”，因此建议默认支持 Accuracy Circle：

- 半透明淡蓝圆圈，中心对齐当前定位点。
- 半径基于设备上报精度，如果后端暂时没有精度字段，则按默认 5m / 10m 级别展示。
- 当定位处于 `unable` 或 `offline`，不绘制精度圈。

### 6.4 静止与无定位的图层规则

结合当前 MQTT 协议：

- `F:` 产生真实位置点，可用于轨迹绘制。
- `S:` 仅更新上一真实点的 `still_seconds`，地图上不新增新点。
- `Z:0` 只表示在线但无有效定位，地图上不应伪造坐标。

这部分前端必须严格遵守后端语义，避免把保活消息误渲染成新轨迹点。

## 7. 核心页面设计

## 7.1 内部主工作台 `/app/map`

这是内部用户最常用页面，建议采用三栏布局：

```text
左侧：人员列表与筛选
中间：主地图
底部或右侧：选中人员详情
顶部：团队、在线数、告警入口、分享入口、当前时间
```

### 7.1.1 顶部栏

建议包含：

- 系统名 `Locator Hub`
- 当前团队/区域切换
- 在线人数统计
- 告警计数
- WebSocket 连接状态
- 当前用户菜单

### 7.1.2 左侧人员列表

每个列表卡片建议包含：

- 姓名
- 岗位或标签，如“外勤”“巡检”“出差”
- 电量
- 当前状态，如“步行中”“静止 1 小时”“离线 8 分钟”
- 最新更新时间
- 告警徽标，如 `SOS`、`低电量`、`围栏外`

筛选维度建议支持：

- 姓名搜索
- 在线状态
- SOS 优先
- 低电量优先
- 是否在围栏内

### 7.1.3 地图区域

地图中建议支持：

- 多人员实时点位
- 当前选中人员居中聚焦
- 围栏叠加
- 告警点位与轨迹
- 轻量工具条

地图工具条建议包含：

- 回到全部人员视野
- 跟随当前选中人员
- 显示/隐藏围栏
- 显示/隐藏轨迹
- 切换浅色或暗色底图

### 7.1.4 详情面板

选中某个人员后，详情面板展示：

- 姓名
- 角色或岗位
- 电量
- GPS 状态
- 在线状态
- 最后在线时间
- 最近可信定位时间
- 当前位置描述
- 当前速度或静止时长
- IMEI / ICCID
- 当前设备配置摘要

操作区按钮建议：

- `[历史轨迹]`
- `[查看围栏]`
- `[分享实时位置]`
- `[更多]`

`[更多]` 中可放：

- 获取状态
- 获取配置
- 远程下发配置

## 7.2 历史轨迹页 `/app/devices/:deviceSN/history`

该页核心任务是回放与判读，不是展示全部设备。

建议布局：

- 左上：人员信息摘要
- 右上：时间范围选择
- 中间：地图 + 轨迹
- 底部：轨迹时间轴和关键点列表

关键功能：

- 今日、近 6 小时、近 24 小时快捷筛选
- 自定义时间范围
- 轨迹回放播放
- 静止段高亮
- 围栏穿越点标注
- 告警点并轨显示

轨迹绘制建议：

- 真实定位点使用后端返回的 `tracks`
- `still_seconds > 0` 的点绘制为“停留节点”
- 连线颜色可从浅蓝渐变到深蓝，表达时间方向

## 7.3 告警中心 `/app/alarms`

面向调度员和管理员，集中查看：

- `sos`
- `offline`
- `out_of_fence`
- `low_battery`

建议结构：

- 顶部筛选条
- 左侧告警时间线
- 右侧地图聚焦

列表项显示：

- 告警类型
- 人员姓名 / `device_sn`
- 告警内容
- 发生时间
- 快捷动作，如“定位到地图”“打开详情”

## 7.4 公开访客页 `/s/:shareCode`

公开页必须极简，不允许出现任何后台能力。

建议只保留：

- 品牌条
- 剩余有效时间
- 被分享人的名字
- 地图
- 电量
- 实时状态
- 最近更新时间
- 简要地址或区域描述

不应展示：

- 设备命令
- 围栏编辑
- 告警列表
- IMEI / ICCID
- 团队其他人员
- 内部导航

## 8. 分享功能交互规范

## 8.1 分享弹窗目标

分享弹窗不是单纯“复制链接”，而是一个带安全策略的授权面板。

建议字段：

- 分享模式
  - 仅实时位置
  - 允许查看今日轨迹
- 是否启用访问密码
- 密码内容
- 有效期
- 访问次数限制

### 8.1.1 默认值建议

- 默认启用访问密码
- 默认生成 6 位数字密码
- 默认有效期 1 小时
- 默认访问次数 5 次
- 默认分享模式为“仅实时位置”

## 8.2 分享弹窗成功态

生成成功后，应显示：

- 完整链接
- 密码
- 剩余有效时间
- 剩余访问次数
- `复制链接`
- `复制全部信息`

复制全部信息文案建议：

```text
位置分享链接：{url}
访问密码：{password}
有效期至：{expireAt}
剩余访问次数：{remaining}
```

## 8.3 访问次数与防刷 UX

访问次数限制要避免“用户刷新一下就扣一次”的糟糕体验。

建议规则：

1. 第一次成功通过密码校验并建立有效访问会话时，才扣减 1 次。
2. 同一浏览器在有效会话内反复刷新，不重复扣减。
3. 会话可由后端返回 `viewer_session_token` 或使用 HttpOnly Cookie。
4. 访问次数耗尽后，返回专用失效页。

### 8.3.1 次数耗尽页文案

建议文案：

```text
该分享链接的访问次数已达上限。
如需继续查看，请联系分享人重新生成链接。
```

## 8.4 密码校验页

如果启用了密码，首次进入分享页时不直接展示地图，而是先进入验证页：

- 标题：位置分享已受保护
- 输入框：6 位密码
- 错误提示：密码错误，请重试
- 二级信息：分享链接剩余有效时间

## 9. SOS 紧急状态 UX

## 9.1 全局触发条件

当后端通过 WebSocket 推送 `alarm` 事件且 `type = sos` 时，前端执行以下行为：

1. 顶部弹出全局紧急 Banner。
2. 播放一次可被用户关闭的警报音。
3. 地图平滑聚焦到该人员位置。
4. 对应 Marker 切换为红黑闪烁态。
5. 左侧列表将此人置顶。

## 9.2 全局 Banner 设计

Banner 建议固定在顶部，不被局部滚动遮挡。

内容包含：

- `SOS 紧急求助`
- 人员姓名
- 触发时间
- 快捷动作按钮

动作按钮建议：

- `[查看位置]`
- `[打开详情]`
- `[静音]`

## 9.3 声音与动效原则

SOS 允许更强的视觉动效，但必须克制：

- 声音默认播放一次，不循环轰炸。
- Marker 闪烁频率不要过快，避免视觉疲劳。
- 地图平移采用平滑动画，不瞬移。

## 10. OSM 与 Leaflet 交互细节

## 10.1 底图策略

建议至少支持两套底图：

- 默认浅色 OSM
- 暗色 OSM

适用场景：

- 白天调度：浅色
- 夜班值守或大屏：暗色

## 10.2 平滑轨迹建议

人员步行的原始 GPS 点更容易产生锯齿状轨迹，建议：

- 后端仍保留原始轨迹数据
- 前端显示层可对轨迹做轻量平滑
- 平滑只用于视觉展示，不回写后端

推荐策略：

1. 默认显示原始轨迹。
2. 勾选“平滑轨迹”后，对折线点做 Kalman Filter 或简化滤波。
3. 在 UI 上明确标注“已启用显示平滑，仅用于视觉优化”。

## 10.3 跟随模式

在单人查看时可开启“跟随人员”：

- WebSocket 收到该人新位置后自动 `panTo`
- 用户手动拖动地图后，自动退出跟随模式
- 顶部显示“已暂停跟随”提示，支持一键恢复

## 10.4 地址展示策略

公开访客页常常更希望看到“这人在哪附近”，而不是纯坐标。

建议：

- 若后端暂时没有逆地理编码接口，先显示经纬度 + 团队区域名。
- 后续新增地理编码后，优先显示“北京市朝阳区建国路附近”这类弱精确文本。

## 11. React + TypeScript + Tailwind 前端工程方案

## 11.1 推荐技术组合

前端基础栈建议：

- React 19
- TypeScript
- Vite
- Tailwind CSS
- React Router
- TanStack Query
- Zustand
- Leaflet
- shadcn/ui

说明：

- `TanStack Query` 负责服务端状态。
- `Zustand` 负责 UI 状态和地图交互态。
- `shadcn/ui` 只提供基础交互组件，如 Dialog、Drawer、Dropdown，不负责产品风格。

## 11.2 推荐目录结构

```text
web/src/
  app/
    router.tsx
    providers.tsx
  pages/
    login/
    map/
    alarms/
    device-history/
    public-share/
  features/
    auth/
    devices/
    alarms/
    fences/
    share/
    realtime/
    map-view/
  components/
    shell/
    map/
    status/
    share/
    feedback/
  services/
    http/
    websocket/
  stores/
    auth-store.ts
    map-store.ts
    ui-store.ts
  hooks/
    use-auth.ts
    use-realtime.ts
    use-device-list.ts
    use-share-session.ts
  lib/
    format.ts
    geo.ts
    battery.ts
    status.ts
    time.ts
  types/
    api.ts
    device.ts
    alarm.ts
    share.ts
```

## 11.3 页面级组件拆分

### 地图页组件建议

```text
MapPage
  AppShell
  TeamTopbar
  DeviceSidebar
  LiveMapCanvas
  DeviceDetailPanel
  GlobalAlarmBanner
  ShareLocationDialog
```

### 公开页组件建议

```text
PublicSharePage
  PublicShareHeader
  PasswordGate
  PublicLiveMap
  ShareMetaCard
  ShareInvalidState
```

## 11.4 状态管理边界

建议按下列边界管理状态：

### TanStack Query

用于缓存：

- 当前登录用户
- 设备列表
- 设备详情
- 历史轨迹
- 围栏列表
- 告警列表
- 分享链接详情

### Zustand

用于 UI 与地图状态：

- 当前选中人员
- 当前地图中心与缩放级别
- 是否显示围栏
- 是否显示轨迹
- 是否跟随选中人员
- 左侧筛选器
- 分享弹窗开关
- SOS Banner 当前状态

### WebSocket 临时流

用于接收：

- `location`
- `device_status`
- `alarm`

WebSocket 更新进来后：

1. 优先更新 Zustand 中的实时地图态。
2. 再选择性同步到 Query Cache。
3. 避免每一条位置消息都全量刷新整页查询。

## 12. 与当前后端的接口契约

## 12.1 当前已可直接接入的内部接口

当前后端已经具备以下能力：

- `POST /api/auth/login`
- `GET /api/devices`
- `GET /api/devices/:device_sn`
- `GET /api/devices/:device_sn/tracks`
- `GET /api/devices/:device_sn/fences`
- `GET /api/alarms`
- `POST /api/devices/:device_sn/commands`
- `GET /api/mqtt/status`
- `GET /api/mqtt/messages`
- `GET /ws`

所有受保护接口默认需要 JWT。

### 12.1.1 登录响应

```json
{
  "success": true,
  "data": {
    "token": "jwt-token",
    "expires_at": "2026-06-22T10:00:00Z",
    "user": {
      "id": 1,
      "username": "admin",
      "role": "admin"
    }
  }
}
```

### 12.1.2 设备列表/详情关键字段

当前设备数据建议前端至少消费以下字段：

- `device_sn`
- `imei`
- `iccid`
- `name`
- `topic_prefix`
- `gps_state`
- `status`
- `battery`
- `status_payload`
- `config_payload`
- `status_updated_at`
- `config_updated_at`
- `last_fix_at`
- `last_online`

其中：

- `status_payload` 是设备上报的完整状态 JSON。
- `config_payload` 是设备当前生效配置 JSON。
- `last_fix_at` 已根据 `fix_age_ms` 做过反推，可直接用于显示“最近可信定位时间”。

### 12.1.3 轨迹接口

轨迹接口返回的点结构为：

```json
{
  "lat": 39.9,
  "lng": 116.3,
  "time": "2026-06-15T10:00:00Z",
  "still_seconds": 600
}
```

前端要注意：

- `still_seconds = 0` 表示普通移动点
- `still_seconds > 0` 表示停留点

### 12.1.4 WebSocket 事件

当前实时事件统一结构为：

```json
{
  "type": "location | device_status | alarm",
  "data": {}
}
```

前端需要处理的三个事件：

- `location`
  - 更新地图 Marker 和轨迹尾点
- `device_status`
  - 更新电量、在线状态、GPS 状态、`status_payload`、`config_payload`
- `alarm`
  - 更新告警列表，并在 `type = sos` 时触发全局紧急流程

## 12.2 当前后端不足以支撑的能力

分享功能目前后端尚未实现，前端不能只靠静态路由硬做，需要新增接口。

### 12.2.1 建议新增分享数据模型

建议新增 `location_shares` 表，字段至少包括：

- `id`
- `device_id`
- `share_code`
- `share_mode`
- `password_hash`
- `require_password`
- `expires_at`
- `max_visits`
- `visit_count`
- `created_by`
- `created_at`
- `revoked_at`

### 12.2.2 建议新增内部分享接口

```text
POST   /api/devices/:device_sn/shares
GET    /api/devices/:device_sn/shares
DELETE /api/shares/:share_id
```

`POST /api/devices/:device_sn/shares` 请求建议：

```json
{
  "mode": "live_only",
  "require_password": true,
  "password": "888999",
  "expires_at": "2026-06-23T00:00:00Z",
  "max_visits": 5
}
```

响应建议：

```json
{
  "success": true,
  "data": {
    "share_id": "shr_001",
    "share_code": "a7f9g2e",
    "url": "https://maps.locatorhub.com/s/a7f9g2e",
    "password": "888999",
    "expires_at": "2026-06-23T00:00:00Z",
    "max_visits": 5,
    "remaining_visits": 5,
    "mode": "live_only"
  }
}
```

### 12.2.3 建议新增公开访问接口

```text
GET  /api/public/shares/:share_code
POST /api/public/shares/:share_code/verify
GET  /api/public/shares/:share_code/location
GET  /api/public/shares/:share_code/track
GET  /api/public/shares/:share_code/ws-token
```

职责建议：

- `GET /api/public/shares/:share_code`
  - 返回分享是否存在、是否过期、是否需要密码
- `POST /api/public/shares/:share_code/verify`
  - 校验密码并建立访问会话
- `GET /api/public/shares/:share_code/location`
  - 返回公开态需要的单人位置摘要
- `GET /api/public/shares/:share_code/track`
  - 当模式允许今日轨迹时返回轨迹
- `GET /api/public/shares/:share_code/ws-token`
  - 返回只读临时 WS 凭证

### 12.2.4 建议新增分享态 WebSocket

公开访客页建议不要复用后台全量 `/ws` 订阅，而是通过受限凭证订阅单人实时位置：

- 只能收到被分享对象的数据
- 不推送团队其他人的位置
- 不推送后台告警总线

## 13. TypeScript 类型建议

建议前端先把核心领域类型单独定义，避免 UI 到处写匿名结构。

```ts
export type DeviceStatus = 0 | 1 | 2

export type GPSState =
  | "not_started"
  | "offline"
  | "searching"
  | "located"
  | "unable"

export interface DeviceSummary {
  device_sn: string
  imei: string
  iccid: string
  name: string
  topic_prefix: string
  gps_state: GPSState
  status: DeviceStatus
  battery: number
  status_payload?: Record<string, unknown>
  config_payload?: Record<string, unknown>
  status_updated_at?: string
  config_updated_at?: string
  last_fix_at?: string
  last_online?: string
  created_at: string
}

export interface TrackPoint {
  lat: number
  lng: number
  time: string
  still_seconds: number
}

export interface AlarmSummary {
  device_sn: string
  type: string
  content: string
  created_at: string
}
```

## 14. Tailwind 与组件风格建议

## 14.1 组件风格原则

- 面板使用半透明白底 + 轻模糊
- 图层按钮尺寸统一，圆角统一
- 状态色必须统一封装，不能页面里各自判断
- 所有危险操作和紧急状态必须有明确层级差异

## 14.2 建议抽出的基础组件

- `StatusBadge`
- `BatteryBadge`
- `SignalPill`
- `LiveDot`
- `GlassPanel`
- `MapToolbar`
- `DetailSection`
- `EmergencyBanner`
- `ShareInfoCard`

### `StatusBadge` 输入建议

- `gps_state`
- `status`
- `battery`
- `alarmType?`

由组件统一输出文案：

- 在线
- 搜索中
- 已定位
- 无定位
- 离线
- 低电量
- SOS

## 15. 实现优先级建议

为了降低风险，前端建议分三阶段推进。

### 第一阶段：内部地图工作台

先完成：

- 登录
- 地图总览页
- 左侧人员列表
- 单人详情面板
- WebSocket 实时联动
- 历史轨迹页
- 围栏展示

这一阶段完全可以基于当前后端开工。

### 第二阶段：告警与 SOS

完成：

- 告警列表页
- 全局 SOS Banner
- 地图自动聚焦
- 声音提醒
- 低电量高亮

这一阶段同样可以基于当前后端继续推进。

### 第三阶段：分享系统

在后端补齐分享接口后再做：

- 分享弹窗
- 公开密码校验页
- 公开实时地图页
- 访问次数与设备会话保护

## 16. 结论

这套前端不应当被当成“后台管理 + 一张地图”的普通项目，而应按“安全调度台 + 单人隐私分享”的双模式产品来设计。

落地上建议遵守三个原则：

1. 内部工作台优先强调实时态势、告警和人员状态判断。
2. 公开分享页优先强调隐私、克制和单目标查看。
3. React 工程上严格拆分地图层、实时层、业务层，避免所有逻辑塞进一个页面组件。

如果后续按本说明书继续实现，建议先初始化 `web/` 工程并优先完成内部地图工作台，不要先做分享页，因为当前后端还没有分享数据模型和公开接口。
