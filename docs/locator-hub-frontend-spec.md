# Locator Hub 前端 UI/UX 与实现说明

## 1. 文档目标

这份文档不是概念稿，而是面向当前仓库实际代码结构的前端实施说明。目标有三个：

1. 把系统从“通用车辆定位看板”明确收敛到“人员定位与安全管理系统”。
2. 说明当前 React + TypeScript + Tailwind CSS 前端已经落地的页面、路由、共享组件和数据流。
3. 明确哪些能力已经接通真实后端，哪些能力目前先用演示数据或前端预览完成验证。

当前前端已经按“双模式”组织：

- `demo`：用死数据做 UI、交互、状态流和演示验证。
- `live`：走真实后端 API 和 WebSocket。

这两套模式共用同一套页面和组件，不复制页面实现，只切换数据源。

## 2. 产品定位

### 2.1 从车到人

系统已经按“人防场景”重新定义前端表现，不再以车载定位 UI 为中心。

核心变化如下：

- 地图 Marker 由车标改为“人员首字母圆形身份点”。
- 状态维度从“是否在线”扩展到“GPS 状态、活动状态、电量、SOS 告警”。
- 详情面板强调“人员状态 + 安全状态 + 设备状态 + 当前配置”。
- 分享能力按“隐私优先”设计，但当前仅落地了前端交互预览，真实后端分享接口还未实现。

### 2.2 核心使用场景

1. 管理员在地图总览页查看团队分布、在线人数、低电量、SOS 和围栏相关风险。
2. 调度人员查看某个人员的实时位置、状态摘要、历史轨迹和设备配置。
3. 告警中心集中查看 `sos`、`low_battery`、`offline`、`out_of_fence`。
4. 外部访客通过分享链接进入一个极简的公开看板，只看到单人位置。

## 3. 技术栈与工程约束

### 3.1 当前前端栈

- React 19
- TypeScript
- Vite
- Tailwind CSS
- React Router
- TanStack Query
- Zustand
- Leaflet

### 3.2 包管理

前端统一使用 `yarn`，不是 `npm`。

常用命令：

```bash
cd web
yarn
yarn dev
yarn build
```

当前 `package.json` 已经改为通过本地 `node_modules` 显式调用 `tsc` 和 `vite`，避免 Windows PowerShell 下 `yarn build` 找不到可执行文件的问题。

## 4. 当前前端模式设计

### 4.1 设计原则

不做两套页面，不维护两份 UI。页面只维护一份，数据通过统一抽象切换。

当前采用的数据源抽象：

- `MapDataSource`
- `MapDataProvider`
- `demoDataSource`
- `liveDataSource`

共享页面只依赖这些统一接口：

- `useDevices`
- `useDeviceDetail`
- `useRealtimeFeed`
- `useAlarms`
- `useTrack`

### 4.2 模式差异

#### `demo`

用途：

- 死数据界面验收
- 演示用 SOS 流程
- 分享 UI 验证
- 轨迹页和告警页交互验证

特点：

- 数据来自 `mock-devices.ts`、`mock-alarms.ts`、`mock-tracks.ts`
- 地图会模拟实时移动
- 会定时注入一条 `sos` 告警消息

#### `live`

用途：

- 联调真实后端
- 接通设备列表、详情、告警、轨迹和 WebSocket

特点：

- 列表和详情来自 HTTP API
- 实时位置和告警来自 WebSocket
- 历史轨迹来自 `/api/devices/:device_sn/tracks`
- 告警来自 `/api/alarms`

## 5. 当前路由结构

### 5.1 已落地路由

```text
/login

/demo/map
/demo/alarms
/demo/devices/:deviceSN/history
/demo/share/:deviceSN

/app/map
/app/alarms
/app/devices/:deviceSN/history
```

### 5.2 路由说明

- `/login`
  - 内部登录页，只用于 live 模式登录入口。
- `/demo/map`
  - 死数据地图总览页。
- `/demo/alarms`
  - 死数据告警中心。
- `/demo/devices/:deviceSN/history`
  - 死数据历史轨迹页。
- `/demo/share/:deviceSN`
  - 演示版公开分享看板，只做前端视觉验证。
- `/app/map`
  - 真实后端地图总览页。
- `/app/alarms`
  - 真实后端告警中心。
- `/app/devices/:deviceSN/history`
  - 真实后端历史轨迹页。

## 6. 已实现页面说明

### 6.1 登录页

位置：

- `web/src/pages/login/login-page.tsx`

作用：

- 调用 `POST /api/auth/login`
- 保存 JWT 到 `authStore`
- 登录成功后跳转 `/app/map`

设计说明：

- 左侧是品牌与场景说明
- 右侧是登录表单
- 页面显式提供 `/demo/map` 入口，便于只验证前端时直接进入

### 6.2 地图总览页

位置：

- `web/src/pages/map/map-page.tsx`

布局：

- 顶部：`AppHeader`
- 左侧：`DeviceSidebar`
- 中间：`DeviceMap`
- 右侧：`DeviceDetailPanel`

已实现能力：

- 设备列表搜索
- 选中设备并跟随
- OSM 地图展示
- 实时点位渲染
- 人员圆形 Marker
- 精度圈显示
- 详情面板展示状态、设备信息、配置摘要
- 分享弹窗预览
- SOS 全局 Banner
- 收到 SOS 后自动选中、聚焦并提供“查看历史轨迹”入口

### 6.3 告警中心页

位置：

- `web/src/pages/alarms/alarms-page.tsx`

布局：

- 顶部：共享头部
- 左侧：告警列表和类型筛选
- 右侧：地图联动 + 当前焦点摘要

已实现能力：

- 告警类型筛选
- 告警列表查看
- 点击告警后地图聚焦对应人员
- 从告警页跳转到该人员历史轨迹页

### 6.4 历史轨迹页

位置：

- `web/src/pages/history/history-page.tsx`

布局：

- 顶部：共享头部
- 左侧：轨迹地图
- 右侧：摘要卡片 + 时间线列表

已实现能力：

- 近 1 小时 / 近 6 小时 / 近 24 小时时间范围切换
- Polyline 轨迹渲染
- 轨迹点点击联动
- 停留点识别
- 停留总时长汇总
- 相关告警数量展示

### 6.5 演示分享页

位置：

- `web/src/pages/share/demo-share-page.tsx`

说明：

- 这是一个前端演示页，不接真实后端分享会话。
- 只用于验证公开看板视觉、极简权限边界和地图呈现方式。

已实现能力：

- 单人地图看板
- 电量、状态、最近更新时间、位置、精度展示
- 底图明暗切换

未实现能力：

- 密码校验
- 链接次数扣减
- 有效期失效
- 公开 WebSocket 会话

## 7. 共享组件设计

### 7.1 当前关键组件

- `AppHeader`
- `DeviceSidebar`
- `DeviceMap`
- `DeviceDetailPanel`
- `EmergencyBanner`
- `ShareLocationModal`
- `TrackMap`
- `StatusBadge`
- `BatteryBadge`

### 7.2 组件职责

#### `AppHeader`

负责：

- 展示页面标题
- 展示模式标签
- 展示顶部统计指标
- 展示导航入口
- 在 live 模式提供退出登录按钮

#### `DeviceSidebar`

负责：

- 搜索人员
- 展示设备列表
- 展示电量、状态、最近在线时间

#### `DeviceMap`

负责：

- 渲染多设备 Marker
- 渲染精度圈
- 渲染选中状态
- 渲染 SOS 闪烁态
- 跟随选中人员

#### `DeviceDetailPanel`

负责：

- 展示当前选中人员的状态摘要
- 展示 IMEI、ICCID、模块固件、网络状态
- 展示当前配置摘要
- 提供历史轨迹入口
- 提供分享入口

#### `EmergencyBanner`

负责：

- 全局显示 SOS 告警
- 提供定位到地图
- 提供历史轨迹跳转
- 提供关闭

#### `ShareLocationModal`

负责：

- 分享模式选择
- 密码开关与密码输入
- 有效期设置
- 次数限制设置
- 生成分享结果预览
- 复制完整信息

说明：

- 当前这个弹窗在 `demo` 和 `live` 都能打开。
- `live` 里目前只是前端预览，不会真正创建分享记录。

## 8. 当前视觉与交互实现

### 8.1 Marker 设计

当前地图 Marker 已经从车辆图标切换为“人员首字母圆点”：

- 内层：深色圆形 + 姓名首字母
- 外圈：根据状态着色
- 脉冲：表示实时在线
- SOS：红色高风险闪烁态

### 8.2 状态颜色建议

当前实现与设计建议一致：

- 正常定位：绿色
- 搜索中：品牌蓝
- 无定位或低电量：橙色
- 离线：灰色
- SOS：红色高风险强调

### 8.3 顶部工作台风格

当前页面整体采用：

- 浅灰蓝背景
- 白色毛玻璃面板
- 蓝色为品牌主色
- 红色为风险高亮色

这套视觉方向适合安全调度和长时间盯盘，不像传统后台模板那样生硬。

## 9. SOS 交互设计

### 9.1 当前已实现

当收到 WebSocket `alarm` 且 `type === "sos"` 时：

1. 弹出全局 `EmergencyBanner`
2. 自动选中对应人员
3. 自动开启地图跟随
4. Marker 切换到 SOS 闪烁态
5. 支持跳转到该人员历史轨迹页

### 9.2 当前未实现但建议保留

- 可关闭的报警声设置
- 用户级静音状态
- 多个 SOS 同时发生时的队列或堆叠策略

## 10. 历史轨迹与精度圈策略

### 10.1 当前实现

轨迹页现在使用：

- `Polyline` 绘制路径
- `still_seconds > 0` 识别停留点
- 右侧时间线联动地图

### 10.2 后续建议

你提到的“人行轨迹锯齿”和“精度圈”是对的，建议这样分阶段做：

#### 第一阶段，当前状态

- 保持后端原始轨迹数据
- 前端渲染原始 Polyline
- 用 Accuracy Circle 表达“当前位置置信范围”

#### 第二阶段，增强版

- 在前端增加可切换的“轨迹平滑显示”
- 平滑只作用于显示层，不回写原始轨迹
- 可以用 Kalman Filter 或轻量滤波方案

这样做的好处是：

- 不破坏真实轨迹
- 又能让人眼看到更顺滑的路线
- 方便在 UI 上明确说明“已启用显示平滑”

## 11. 当前后端对接边界

### 11.1 已接通接口

当前前端已经使用这些接口：

- `POST /api/auth/login`
- `GET /api/devices`
- `GET /api/devices/:device_sn`
- `GET /api/devices/:device_sn/tracks`
- `GET /api/alarms`
- `GET /ws`

### 11.2 WebSocket 事件

当前前端已经按统一事件包处理：

```json
{
  "type": "location | device_status | alarm",
  "data": {}
}
```

已消费事件：

- `location`
  - 更新实时点位和最近在线时间
- `device_status`
  - 更新状态、电量、状态负载、配置负载
- `alarm`
  - 更新告警列表
  - 若为 `sos` 则触发全局紧急 Banner

### 11.3 设备详情前端已消耗字段

前端当前依赖这些设备字段：

- `device_sn`
- `name`
- `imei`
- `iccid`
- `gps_state`
- `status`
- `battery`
- `status_payload`
- `config_payload`
- `status_updated_at`
- `config_updated_at`
- `last_fix_at`
- `last_online`

## 12. 分享能力的现状与后续接口建议

### 12.1 当前现状

当前前端只做了“分享 UI 与公开看板演示”，没有真实后端分享能力。

也就是说：

- 可以验证分享弹窗布局和字段
- 可以验证公开页视觉
- 不能真正保证密码、次数、有效期

### 12.2 建议后端补齐的数据模型

建议新增 `location_shares`：

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

### 12.3 建议补齐的内部接口

```text
POST   /api/devices/:device_sn/shares
GET    /api/devices/:device_sn/shares
DELETE /api/shares/:share_id
```

### 12.4 建议补齐的公开接口

```text
GET  /api/public/shares/:share_code
POST /api/public/shares/:share_code/verify
GET  /api/public/shares/:share_code/location
GET  /api/public/shares/:share_code/track
GET  /api/public/shares/:share_code/ws-token
```

### 12.5 次数限制 UX 建议

你前面提的“同一设备刷新不重复扣次数”是合理的，建议后端实现规则如下：

1. 只有首次通过密码校验并建立访问会话时才扣减 1 次。
2. 同一浏览器在会话有效期内刷新页面，不重复扣次数。
3. 公开链接过期或次数耗尽时，返回明确失效状态页。

## 13. 当前工程结构建议

当前 `web/src` 建议继续保持如下组织方式：

```text
app/
  router.tsx
  providers.tsx

pages/
  login/
  map/
  alarms/
  history/
  share/

components/
  shell/
  map/
  status/

features/
  map-view/

hooks/
  use-auth.ts
  use-devices.ts
  use-alarms.ts
  use-track.ts
  use-realtime.ts

services/
  http/
  websocket/

stores/
  auth-store.ts
  map-store.ts

types/
  auth.ts
  device.ts
  alarm.ts
  realtime.ts
```

## 14. 当前实现总结

你要的“两个都做”现在已经按照下面的方式落地：

### 14.1 demo 模式

已实现：

- 地图总览
- 告警中心
- 历史轨迹
- 演示公开分享页
- 模拟实时位置
- 模拟 SOS 告警

用途：

- 先验证前端 UI、交互、状态流程

### 14.2 live 模式

已实现：

- 登录
- 地图总览
- 告警中心
- 历史轨迹
- 真实后端 API 数据
- 真实 WebSocket 实时联动

用途：

- 与当前 Go 后端和设备消息链路做联调

## 15. 下一步建议

按价值优先级，建议后续这样推进：

1. 先补真实分享后端接口，把当前分享弹窗从“预览”升级为“真创建”。
2. 再补电子围栏页面和围栏编辑入口。
3. 最后做轨迹平滑、告警声音设置、公开分享密码校验页等增强能力。

如果继续做前端，下一个最合适的目标就是：把“分享实时位置”从 UI 预览接成真实业务闭环。
