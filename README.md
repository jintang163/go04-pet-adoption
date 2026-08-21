# 宠物领养系统（go04-pet-adoption）

一个使用 **纯 Go 标准库** 从 0 到 1 构建的宠物领养管理系统。救助站发布待领养宠物，领养人提交申请，员工审核并安排家访/回访，全程状态可追溯。系统内置前端页面与文件级 JSON 数据持久化，可通过 Docker 独立运行。

---

## 一、项目简介

流浪动物救助站、社区领养点常见的痛点是：待领养档案散落在表格和朋友圈里，同一只宠物被多人同时“预定”、审核口径不统一、领走后无人回访、退养无法追溯。本系统把四条主链路产品化：

- **展示待领养宠物**：员工录入档案（物种、年龄、健康、性格、领养条件），发布后广场可见。
- **提交申请**：领养人填写住房、家庭成员、养宠经验等问卷后提交；同一宠物每人仅一条有效申请。
- **审核**：员工家访/面谈后录取、候补、拒绝；录取即锁定宠物（预留），其余申请进入候补。
- **回访记录**：交接后系统按 7/30/90 天自动排期回访，员工上门记录生活/健康/行为评分；异常可加访或启动退养。

角色：

- **领养人（adopter）**：注册登录、浏览广场、收藏、提交/撤回申请、查看回访、处理退养。
- **员工（staff）**：管理所属救助站的宠物档案与健康记录、审核申请、家访与回访、办理交接与退养。
- **管理员（admin）**：用户与救助站治理、强制下架/退养、调整信用分、全局统计。

系统使用 Go 1.22 + 标准库（`net/http`、`encoding/json`、`embed`、`sync` 等），**零第三方依赖**，可完全离线构建与运行。

---

## 二、功能特性

### 2.1 用户与权限

| 角色 | 能力 |
|------|------|
| 管理员 | 用户列表/创建/冻结/解冻，救助站管理，强制下架宠物，审核任意申请，调整信用分，查看全局统计 |
| 员工 | 发布与管理本站宠物、录入健康档案、审核申请、安排家访/回访、办理交接与退养 |
| 领养人 | 注册登录、浏览待领养、收藏、提交申请、查看审核与回访、申请退养 |
| 未登录访客 | 仅登录/注册页与健康检查 |

- 首次启动自动创建种子管理员（默认 `admin / admin123`）。
- 领养人可自助注册；员工账号由管理员创建（绑定救助站）。
- 会话：Bearer Token，带过期时间；登出、改密、冻结即失效。
- 口令：盐值 + 多轮迭代 SHA-256（演示级，生产应替换为 bcrypt/argon2）。
- 账号状态：`active` / `frozen`（冻结后不可申请、发宠物） / `banned`（无法登录）。

### 2.2 宠物生命周期（展示待领养）

状态流转：

```
[ 草稿 draft ]
      │ 发布 publish
      ▼
[ 已发布 published ] ──员工下架 / 医疗留置──► [ 不可领养 unavailable ]
      │                                              │ 恢复
      │ 申请被录取                                    ▼
      ▼                                         [ 已发布 published ]
[ 已预留 reserved ]
      │ 交接 handover
      ▼
[ 已领养 adopted ] ──退养 return──► [ 已发布 published ]
      │
      │ 死亡登记
      ▼
[ 已死亡 deceased ]
```

宠物字段要点：

- **物种 `Species`**：狗、猫、兔、鸟、其他。
- **体型 `Size`**：小型 / 中型 / 大型。
- **性别、出生估算、绝育、疫苗、特殊需求**。
- **性格标签**：亲人、怕生、适合有经验领养人等。
- **领养条件**：是否接受公寓、是否接受有孩家庭、是否接受已有宠物、最低领养人年龄。
- **所属救助站 `ShelterID`**：员工只能维护本站宠物；管理员可跨站。
- **封面与相册 URL**（演示用外链/占位，不存二进制）。

列表规则：

- 广场默认只展示 `published`。
- 领养人可按物种、体型、是否绝育、是否特殊需求、关键词过滤。
- 员工/管理员可按状态查看草稿、预留、已领养。

### 2.3 申请规则（核心）

> **定义**：同一领养人对同一宠物最多一条有效申请（`pending` / `under_review` / `waitlisted` / `approved`）。撤回或拒绝后可再次申请（若宠物仍为 published）。

1. 不能申请自己录入的宠物（员工以个人身份领养须走管理员代提或换账号）。
2. 宠物必须为 `published` 才接受新申请；`reserved` 只处理已有名单（候补）。
3. **并发上限**：同一领养人同时处于 `pending` / `under_review` / `approved` 的申请不超过 `MaxPendingApplications`（默认 3）。
4. **信用门槛**：信用等级 `restricted`（分数 ≤ 39）禁止新申请，需管理员解禁或加分。
5. **条件匹配**：住房类型、是否有孩、是否已有宠物、领养人年龄必须满足宠物档案中的硬性条件，否则校验失败。
6. **NeedHomeCheck**：默认开启。员工将申请标为 `under_review` 后，可安排家访（`Visit.Type=home_check`）；家访完成且评分达标才允许录取。
7. **录取**：将申请置为 `approved`，宠物置为 `reserved`；其余 `pending`/`under_review` 自动进入 `waitlisted` 并记录候补序号。
8. **拒绝**：填写原因；若拒绝的是已录取申请，宠物回到 `published`，并按候补 FIFO 尝试递补（递补时再次检查信用与条件）。
9. **撤回**：`pending`/`under_review`/`waitlisted` 可自行撤回；`approved` 后、交接前撤回视为违约，扣信用。
10. 员工可把候补递补为录取（宠物仍须为 reserved 且当前无其他 approved，或当前 approved 已撤销）。

申请状态：`pending` / `under_review` / `waitlisted` / `approved` / `rejected` / `withdrawn` / `completed` / `revoked`。

问卷字段：住房类型（公寓/住宅/自建）、居住面积、是否有孩、是否已有宠物、每日独处时长、养宠经验、联系电话、自我介绍。

### 2.4 审核与交接

1. **开始审核**：`pending` → `under_review`，可写面谈纪要。
2. **家访**：关联申请的 `home_check` 回访记录；生活/环境评分任一低于 3 分视为不通过，不得录取（管理员可强制）。
3. **录取后交接**：员工登记交接单（时间、领取人证件备注、物品清单），申请变为 `completed`，宠物变为 `adopted`，系统按策略生成回访排期。
4. **撤销录取**：交接前可将 `approved` 置为 `revoked`，宠物解锁。

### 2.5 回访记录（核心）

> **定义**：交接完成后，系统为该领养关系自动创建 3 条 **随访（followup）** 排期：第 7、30、90 天。员工可追加 **加访（extra）**。家访（home_check）发生在审核阶段，不计入随访次数。

回访状态：`scheduled` / `completed` / `missed` / `cancelled`。

完成回访时填写：

- 生活环境分、健康分、行为分（1–5）
- 是否发现虐待/弃养风险 `RiskFlag`
- 文字记录、改进建议、是否需要加访

规则：

1. 仅 `adopted` 宠物的已完成申请可产生随访。
2. 到期未完成超过宽限（默认 3 天）可被员工标为 `missed`；连续两次 missed 扣信用并强制加访。
3. `RiskFlag=true`：通知管理员，可启动退养流程。
4. 领养人可查看与自己相关的回访，但不能改评分（可补充说明）。
5. 90 天随访全部完成且无风险，给领养人小幅加信用。

### 2.6 健康档案、收藏、问询、审计

- **健康档案**：疫苗、驱虫、绝育、体检、治疗记录；员工维护，领养广场详情只读展示。
- **收藏**：领养人可收藏 published 宠物；宠物下架后收藏仍可查看但提示不可申请。
- **问询**：对宠物留言，员工可回复；不当内容管理员可删除。
- **通知**：审核结果、候补递补、回访到期提醒（列表页惰性提示）、退养结果。
- **审计日志**：发布、录取、交接、回访、退养、强制下架写 `AuditLog`。
- **信用流水**：申请违约、连续缺访、退养、良好回访、管理员加减分。

### 2.7 退养

领养人可在交接后发起退养申请（说明原因）。员工/管理员审核：

- 通过：宠物回到 `published`（或因医疗原因 `unavailable`），原申请保持 `completed` 并标记 `ReturnedAt`；30 天内无正当理由退养扣信用。
- 拒绝：维持领养关系，安排加访。

### 2.8 统计

管理员看板：用户数、宠物数（按状态）、申请转化率、本月交接数、回访完成率、退养率。  
员工看板：本站待审申请、今日到期回访、草稿宠物。  
领养人主页：进行中的申请、即将到来的回访、收藏。

---

## 三、业务对象与持久化

数据全部保存在 `data/store.json`（路径由 `APP_DATA_PATH` 配置）。内存结构变更后通过钩子 **原子写盘**（临时文件 + rename），进程重启后恢复。

主要集合：

- Users、Shelters、Pets、Applications、Visits、HealthRecords
- Favorites、Inquiries、Notifications、AuditLogs、CreditLogs

并发：`MemoryStore` 使用 `sync.RWMutex`；跨实体操作（录取+锁宠物+候补、交接+生成回访、退养+信用）在同一把锁内完成，避免半更新。

---

## 四、API 一览

前缀 `/api`，除登录注册、健康检查外均需 `Authorization: Bearer <token>`。

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/healthz` | 健康检查 |
| POST | `/api/auth/register` | 领养人注册 |
| POST | `/api/auth/login` | 登录 |
| POST | `/api/auth/logout` | 登出 |
| GET | `/api/auth/me` | 当前用户 |
| PUT | `/api/me/profile` | 改资料 |
| PUT | `/api/me/password` | 改密 |
| GET | `/api/species` | 物种/体型/住房等枚举 |
| GET | `/api/shelters` | 救助站列表 |
| POST | `/api/shelters` | 创建救助站（管理员） |
| GET/POST | `/api/pets` | 列表 / 创建草稿 |
| GET/PUT | `/api/pets/{id}` | 详情 / 更新草稿 |
| POST | `/api/pets/{id}/publish` | 发布 |
| POST | `/api/pets/{id}/unpublish` | 下架为草稿或 unavailable |
| POST | `/api/pets/{id}/apply` | 提交申请 |
| GET | `/api/pets/{id}/applications` | 该宠申请名单 |
| GET | `/api/pets/{id}/health` | 健康档案 |
| POST | `/api/pets/{id}/health` | 新增健康记录 |
| POST | `/api/pets/{id}/favorite` | 收藏 |
| GET | `/api/pets/{id}/inquiries` | 问询 |
| POST | `/api/pets/{id}/inquiries` | 提问 |
| POST | `/api/inquiries/{id}/reply` | 回复问询 |
| GET | `/api/applications/{id}` | 申请详情 |
| POST | `/api/applications/{id}/review` | 开始审核 |
| POST | `/api/applications/{id}/approve` | 录取 |
| POST | `/api/applications/{id}/reject` | 拒绝 |
| POST | `/api/applications/{id}/withdraw` | 撤回 |
| POST | `/api/applications/{id}/handover` | 交接 |
| POST | `/api/applications/{id}/return` | 退养 |
| GET | `/api/me/applications` | 我的申请 |
| GET | `/api/me/visits` | 我的回访 |
| GET | `/api/visits` | 回访列表（员工） |
| POST | `/api/visits` | 安排家访/加访 |
| POST | `/api/visits/{id}/complete` | 完成回访 |
| POST | `/api/visits/{id}/miss` | 标记缺访 |
| GET | `/api/me/notifications` | 通知 |
| GET | `/api/stats` | 统计（管理员） |
| GET | `/api/users` | 用户列表（管理员） |
| POST | `/api/users/{id}/freeze` | 冻结 |

前端页面（`embed`）：`/login`、`/app`（待领养广场）、`/pets/{id}`、`/me`、`/staff`、`/admin`。

---

## 五、技术架构

```
main.go
  ├─ config          环境变量
  ├─ store           MemoryStore + FileStore 原子持久化
  ├─ auth            口令哈希 / 会话
  ├─ policy          信用、排期、住房匹配等策略常量
  ├─ service         宠物 / 申请 / 回访 / 健康 / 通知
  ├─ handler         HTTP JSON + 页面
  ├─ middleware      鉴权、角色、CORS、日志、Recover
  └─ web/assets      内置 HTML/CSS/JS
```

约束：

- `go.mod` 声明 `go 1.22`，不使用 Go 1.23+ API。
- 零第三方模块，质检镜像 `golang:1.22` 内 `go mod download` 与 `go build ./...` 可离线完成。
- 运行镜像见项目根 `Dockerfile` + `docker-compose.yml`（端口 8080，挂载 `./data`）。

---

## 六、默认账号与演示数据

| 账号 | 口令 | 角色 |
|------|------|------|
| admin | admin123 | 管理员 |
| staff | staff123 | 员工（阳光宠物救助站） |
| alice | alice123 | 领养人 |
| bob | bob123 | 领养人 |

`APP_SEED_DEMO=true` 时写入上述账号、一个救助站，以及若干待领养宠物（含一只已发布的中华田园猫、一只金毛草稿等）。

环境变量：`APP_ADDR`、`APP_DATA_PATH`、`APP_SESSION_TTL`、`APP_ADMIN_USERNAME`、`APP_ADMIN_PASSWORD`、`APP_SEED_ADMIN`、`APP_SEED_DEMO`、`APP_MAX_PENDING_APPLICATIONS`。

---

## 七、本地与 Docker 运行

见 `BENZHI_README.md`。简述：

```bash
go run .
# 或
bash ./go-run.sh
```

浏览器打开 http://localhost:8080/login 。

---

## 八、核心规则速查

| 场景 | 结果 |
|------|------|
| 宠物非 published | 不可新申请 |
| 信用 restricted | 禁止申请 |
| 同时进行中申请 ≥ 3 | 409 冲突 |
| 住房/有孩/已有宠物不满足档案条件 | 400 校验失败 |
| 录取申请 | 宠物 reserved，其余申请 waitlisted |
| 拒绝已录取申请 | 宠物解锁，候补 FIFO 递补 |
| 交接 | 宠物 adopted，生成 7/30/90 天回访 |
| 连续两次缺访 | 扣信用并强制加访 |
| 回访标记风险 | 通知管理员，可启动退养 |
| 30 天内无故退养 | 扣信用 |
