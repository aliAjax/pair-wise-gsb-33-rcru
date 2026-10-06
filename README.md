# FloraWiki（植物养护知识百科平台）

为园艺爱好者提供全面的植物养护指南：品种库、养护文章、病虫害诊断、季节养护日历、我的花园、问答社区与养护小测验，支持图文展示与个人花园/提醒管理。

## Docker Compose 一键启动（推荐）

```bash
cp .env.example .env
docker compose up -d --build
```

启动后访问：

- 前端：http://localhost:8102
- 后端 API：http://localhost:3102
- 健康检查：http://localhost:3102/healthz

默认种子账号：`admin / admin123`（管理员）、`gardener / user123`（普通用户）。

关闭并清理数据：

```bash
docker compose down -v --remove-orphans
```

## 本地开发（备选）

后端（Go 1.22 + Gin + GORM）：

```bash
cd backend
go mod tidy
go run ./cmd/server
go build ./...
go test ./...
```

前端（Vue 3 + TypeScript + Element Plus + Vite）：

```bash
cd frontend
npm install
npm run dev     # 开发服务器，/api 代理到 http://localhost:3102
npm run build   # 生产构建
```

## 技术栈

| 分层 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Element Plus + Vite + Pinia + Vue Router |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 认证 | JWT（github.com/golang-jwt/jwt/v5）+ RBAC |
| 其他 | validator/v10、log/slog、Nginx |

## 项目目录结构

```
gb-61/
├── docker-compose.yml
├── .env.example
├── README.md
├── database/
│   └── init.sql                 # 建表 + 种子数据（首次启动自动执行）
├── backend/
│   ├── cmd/server/              # main.go + migrate/seed
│   └── internal/
│       ├── config/              # 环境变量解析
│       ├── model/               # 9 个实体，按实体分文件
│       ├── repository/          # 按实体分文件，哨兵错误
│       ├── service/             # 按实体分文件，构造器注入
│       ├── handler/             # 按实体分文件 + upload/home
│       ├── router/              # router.go + 按实体分文件
│       ├── middleware/          # auth/rbac/rate_limiter/error_handler/logger/cors
│       ├── dto/                 # 请求/响应结构体 + 统一响应包装
│       ├── constants/           # plant/article/favorite/error_codes/log_templates/messages
│       └── util/                # jwt/logger/formatters/app_error/file/season
└── frontend/
    ├── nginx.conf               # /api 反代 backend + SPA
    └── src/
        ├── api/                 # user/plant/article/pest/reminder/favorite/garden/question
        ├── stores/              # authStore/userStore/plantStore/articleStore/reminderStore
        ├── components/common/   # PlantCard/CareArticleCard/FavoriteButton/SearchFilter/...
        ├── hooks/               # useAuth/useFavorite/useReminderStats/useQuiz
        ├── pages/               # Home/PlantLibrary/PlantDetail/ArticleList/.../Login
        ├── router/              # index.ts + guards.ts
        ├── utils/               # request/dateFormat/season
        └── constants/           # plant/article/favorite/errorCodes
```

## 环境变量

| 变量 | 默认值 | 说明 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | gbplantwiki | Compose 项目名/容器前缀 |
| DB_NAME | gbplantwiki_db | MySQL 库名 |
| DB_USER | gbplantwiki_user | MySQL 用户 |
| DB_PASSWORD | gbplantwiki_pwd | MySQL 密码 |
| DB_ROOT_PASSWORD | gbplantwiki_root | MySQL root 密码 |
| JWT_SECRET | change_me_to_a_long_random_string | JWT 签名密钥（生产必改） |
| FRONTEND_PORT | 8102 | 前端端口 |
| BACKEND_PORT | 3102 | 后端端口 |
| DB_PORT | 3502 | 数据库端口 |

## Docker 部署说明

- 端口映射：前端 `8102:80`，后端 `${BACKEND_PORT:-3102}:8080`，数据库 `${DB_PORT:-3502}:3306`
- 数据卷：`db_data`（MySQL 数据）、`uploads`（上传图片）
- 依赖顺序：db healthcheck → backend `depends_on: db: service_healthy` → frontend `depends_on: backend`
- 常见问题：
  - 端口冲突：修改 `.env` 中对应端口后 `docker compose up -d`
  - 数据重置：`docker compose down -v` 后重新 `up`
  - 中文目录名：Compose 使用命名卷与容器名，不依赖目录路径，任意目录名可启动

## API 接口清单

> 后端统一前缀 `/api/v1`，响应统一为 `{ "code": 0, "message": "ok", "data": ... }`。标注「登录」的接口需携带 `Authorization: Bearer <JWT>`。

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| GET | /healthz | 公开 | 健康检查 |
| GET | /api/v1/home/overview | 公开 | 首页聚合：热门品种+最新文章+当季任务 |
| POST | /api/v1/users/register | 公开（限流） | 注册并返回 JWT |
| POST | /api/v1/users/login | 公开（限流） | 登录并返回 JWT |
| GET | /api/v1/users/me | 登录 | 获取当前用户资料 |
| PUT | /api/v1/users/me | 登录 | 更新当前用户资料 |
| GET | /api/v1/plants | 公开 | 品种分页列表/筛选 |
| GET | /api/v1/plants/:id | 公开 | 品种详情 |
| POST | /api/v1/plants | 管理员（限流） | 新增品种 |
| PUT | /api/v1/plants/:id | 管理员 | 更新品种 |
| DELETE | /api/v1/plants/:id | 管理员 | 删除品种 |
| GET | /api/v1/articles | 公开 | 养护文章分页列表/筛选 |
| GET | /api/v1/articles/:id | 公开 | 文章详情并自增阅读数 |
| POST | /api/v1/articles | 登录（限流） | 发布文章 |
| PUT | /api/v1/articles/:id | 登录 | 编辑自己的文章 |
| DELETE | /api/v1/articles/:id | 登录 | 删除自己的文章 |
| GET | /api/v1/pests | 公开 | 病虫害手册搜索 |
| GET | /api/v1/pests/:id | 公开 | 病虫害详情 |
| POST | /api/v1/pests | 管理员（限流） | 新增病虫害条目 |
| PUT | /api/v1/pests/:id | 管理员 | 更新病虫害条目 |
| DELETE | /api/v1/pests/:id | 管理员 | 删除病虫害条目 |
| GET | /api/v1/reminders | 登录 | 当前用户提醒列表（自动标记逾期，带品种来源/盆位状态） |
| GET | /api/v1/reminders/calendar | 登录 | 按月查询提醒 |
| GET | /api/v1/reminders/awaiting | 登录 | 移出植物后待确认的提醒 + 同品种可转入盆 |
| POST | /api/v1/reminders | 登录（限流） | 创建养护提醒（自动锚定 series_id） |
| PUT | /api/v1/reminders/:id | 登录 | 改任务/日期/频率，下一期失效重算 |
| PUT | /api/v1/reminders/:id/status | 登录 | 状态流转 pending/done（done 走幂等完成） |
| POST | /api/v1/reminders/:id/complete | 登录 | 幂等完成：仅生成一条下一期，重复完成返回 processed=false |
| DELETE | /api/v1/reminders/:id | 登录 | 删除提醒 |
| GET | /api/v1/favorites | 登录 | 收藏列表 |
| POST | /api/v1/favorites | 登录（限流） | 添加收藏 |
| DELETE | /api/v1/favorites/:targetType/:targetId | 登录 | 取消收藏 |
| GET | /api/v1/gardens | 登录 | 我的花园列表（带品种来源/状态） |
| POST | /api/v1/gardens | 登录（限流） | 加入我的花园（同品种可多盆） |
| PUT | /api/v1/gardens/:id/reminder | 登录 | 关联养护提醒（事务+同品种校验） |
| DELETE | /api/v1/gardens/:id | 登录 | 移出植物（事务：盆位置 removed + 未完成提醒置 awaiting_confirm，失败回滚恢复） |
| DELETE | /api/v1/gardens/:id/reminders/awaiting | 登录 | 取消该移出盆的待确认提醒 |
| POST | /api/v1/gardens/:id/reminders/transfer | 登录（限流） | 把待确认提醒转给同品种的另一盆 |
| GET | /api/v1/questions | 公开 | 问答列表 |
| GET | /api/v1/questions/:id | 公开 | 问题详情 |
| GET | /api/v1/questions/:id/answers | 公开 | 问题回答列表 |
| POST | /api/v1/questions | 登录（限流） | 发布问题 |
| POST | /api/v1/questions/:id/answers | 登录 | 回答问题 |
| PUT | /api/v1/questions/:id/adopt | 登录 | 采纳最佳回答（事务：清旧最佳+标最佳+关闭问题） |
| PUT | /api/v1/answers/:id/like | 登录 | 回答点赞 |
| POST | /api/v1/uploads | 登录（限流） | 上传图片 |

## 养护提醒系列（series）语义

「我的花园 × 养护提醒 × 植物品种」三者打通后的核心规则：

1. **一条频率一个下一期**：每条提醒属于一个系列（`series_id`，首期 `series_id = id`）。重复型提醒（daily/weekly/monthly/yearly）完成时只生成**一条**下一期；单次提醒不生成下一期。数据库用函数唯一索引
   `uk_reminders_series_open(series_id, CASE WHEN status IN ('pending','overdue','awaiting_confirm') THEN 1 END)`
   保证每个系列至多一个未完成占位。
2. **重复完成返回已处理**：`POST /reminders/:id/complete` 用条件 UPDATE 兜底多窗口/双击并发——只有一次请求匹配到行并生成下一期，其余返回 `processed=false`、`message="该提醒已处理，下一期已生成，请勿重复完成"`。
3. **改日期/频率后下一期失效重算**：`PUT /reminders/:id` 提升 `schedule_version`，删除旧的待执行下一期并按新锚点日期/频率重建；编辑中的就是当前占位时直接原地改写。
4. **植物移出后未完成提醒停在待确认**：`DELETE /gardens/:id` 在一个事务内把盆位置 `removed`、把该盆的 pending/overdue 提醒置为 `awaiting_confirm`（保留品种与系列关联）。随后可：
   - 取消：`DELETE /gardens/:id/reminders/awaiting`
   - 转给同品种另一盆：`POST /gardens/:id/reminders/transfer`（校验目标盆同品种、本人、active）
5. **失败可恢复**：移出、转移、绑定全部走数据库事务，任何一步失败整体回滚，盆位恢复 `active`、提醒关系恢复原状。
6. **界面显示来源与状态**：提醒列表/花园列表都拼装品种名（`plant_name`）、品种类型、盆位昵称（`garden_name`）、位置与盆位状态（`garden_status`），待确认提醒在花园页顶部聚合处理。

## 枚举出现位置清单

### PlantType（植物类型：flower/foliage/succulent/aquatic）

- 后端：`backend/internal/constants/plant.go`（定义）、`backend/internal/model/plant_species.go`（GORM 模型字段）、`backend/internal/service/plant_species_service.go`（校验/日志）、`backend/internal/util/formatters.go`（PlantTypeText）、`backend/internal/constants/log_templates.go`（日志模板）、`backend/internal/constants/error_codes.go`（错误码）、`database/init.sql`（种子数据）
- 前端：`frontend/src/constants/plant.ts`（定义）、`frontend/src/components/common/PlantCard.vue`（类型标签）、`frontend/src/pages/PlantLibrary.vue`（筛选器）、`frontend/src/pages/PlantDetail.vue`（详情）、`frontend/src/utils/season.ts`（plantTypeOptions）

### CareTopicTag（养护主题：fertilizing/pruning/repotting/pest_control/propagation）

- 后端：`backend/internal/constants/article.go`（定义）、`backend/internal/model/care_article.go`（模型）、`backend/internal/service/care_article_service.go`（校验）、`backend/internal/util/formatters.go`（CareTopicText）、`backend/internal/constants/log_templates.go`、`database/init.sql`
- 前端：`frontend/src/constants/article.ts`（定义）、`frontend/src/components/common/CareArticleCard.vue`（标签）、`frontend/src/pages/ArticleList.vue`（筛选）、`frontend/src/pages/ArticleDetail.vue`（详情）

### FavoriteTargetType（收藏目标：plant/article）

- 后端：`backend/internal/constants/favorite.go`（定义）、`backend/internal/model/favorite.go`（模型）、`backend/internal/service/favorite_service.go`（校验）、`backend/internal/constants/log_templates.go`、`database/init.sql`
- 前端：`frontend/src/constants/favorite.ts`（定义）、`frontend/src/components/common/FavoriteButton.vue`（交互）、`frontend/src/pages/Garden.vue` 与 `frontend/src/pages/Profile.vue`（收藏夹列表）

### CareReminder 状态机（pending/done/overdue/awaiting_confirm）与频率（daily/weekly/monthly/yearly）

- 后端：`backend/internal/model/care_reminder.go`（状态与频率常量、`IsOpen`）、`backend/internal/service/care_reminder_service.go`（幂等完成/下一期/改期重算/挂起/转移状态机）、`backend/internal/service/reminder_schedule.go`（`nextRemindDate`/`isValidFrequency`）、`backend/internal/repository/care_reminder_repository.go`（条件 UPDATE、系列查询）、`backend/internal/util/formatters.go`（ReminderStatusText/ReminderFrequencyText/GardenStatusText）、`backend/internal/constants/log_templates.go`（完成/重复/下一期/转移日志）、`backend/internal/constants/messages.go`（已处理/待确认文案）、`database/init.sql`（status 列、函数唯一索引）
- 前端：`frontend/src/constants/reminder.ts`（频率/状态文本与标签颜色）、`frontend/src/types/api.ts`（ReminderStatus 联合类型）、`frontend/src/components/common/ReminderList.vue`（按钮显隐、状态标签、改期）、`frontend/src/components/common/AwaitingReminders.vue`（待确认取消/转移）、`frontend/src/pages/Garden.vue` 与 `frontend/src/pages/SeasonCalendar.vue`（完成/改期交互）、`frontend/src/hooks/useReminderStats.ts`（awaiting 统计）

## 横切关注点

- 认证授权（JWT + RBAC）：`internal/middleware/auth.go`、`rbac.go`、`util/jwt.go`、`router/*.go`、前端 `router/index.ts` 守卫、`components/common/RoleGuard.vue`
- 全局错误处理：`internal/middleware/error_handler.go`、`util/app_error.go`、`constants/error_codes.go`、前端 `utils/request.ts` 拦截器
- 接口限流：`internal/middleware/rate_limiter.go`、登录/发布/上传接口启用
- 文件上传：`handler/upload_handler.go`、`service` 本地存储、`util/file.go`、Nginx `/uploads/` 代理

## License

MIT
