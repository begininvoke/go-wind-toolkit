# gowind-cli 命令行工具

`gowind-cli` 将 go-wind-uiapp 桌面应用的全部能力以**非交互**命令行方式暴露，供开发者与 AI Agent 调用。

- 结果输出到 **stdout**（JSON），日志输出到 **stderr**
- 默认输出缩进 JSON（人可读）；`--json` 输出单行紧凑 JSON（机器友好，适合 AI/脚本解析）
- 退出码：`0` 成功 / `1` 失败 / `2` 用法错误
- 零交互：所有输入通过 flag 或环境变量提供，缺参数直接报错

构建：

```bash
cd gowind-uiapp && go build -o gowind-cli ./cmd/gowind-cli
```

---

## 命令总览

| 命令 | 说明 |
|------|------|
| `gowind-cli project inspect` | 探测项目信息（模块路径、服务列表） |
| `gowind-cli db test/tables/columns` | 数据库连接测试与元数据 |
| `gowind-cli backend grpc/rest` | 后端 Kratos 微服务代码生成 |
| `gowind-cli frontend gen` | 前端 CRUD 代码生成（三框架） |
| `gowind-cli ai presets/test/ddl/partition/review` | AI 助手 |
| `gowind-cli config types/services/export` | 远程配置中心导出 |
| `gowind-cli scaffold project/service` | 项目/服务脚手架 |
| `gowind-cli dev buf/tidy/ent/wire` | 开发工具 |

---

## project — 项目探测

```bash
gowind-cli project inspect --path D:/GoProject/go-wind-admin/backend
```

返回项目根目录、Go 模块路径、`app/` 下服务列表、是否有 `api/` 目录。

## db — 数据库

连接参数二选一：`--dsn`（支持环境变量 `GOWIND_DSN`）或离散参数（`--type --host --port --user --password --database`）。

```bash
# 连接测试
gowind-cli db test --dsn "mysql://user:pass@tcp(localhost:3306)/demo"

# 列出全部表
gowind-cli db tables --type postgresql --host localhost --port 5432 --user demo --password demo --database demo

# 列出某表的列
gowind-cli db columns --dsn "..." --table sys_user
```

> Oracle 连接暂不可用（上游驱动名错配），会明确报错。

## backend — 后端代码生成

数据源二选一：`--ddl <文件>`（本地 DDL，无需连库）或 `--dsn`。表到服务的映射用 `--mapping` JSON 文件或 `--tables` 简写。

```bash
# 表映射文件 tables.json:
# [{"table":"user","service":"identity"},
#  {"table":"role","service":"permission","protoPackage":"permission.service.v1"}]

# 生成 gRPC 全栈（proto+ent+service+server+装配+config），并自动执行
# go mod tidy -> buf generate -> ent generate（wire 仅对旧式 wire 服务执行，手写装配服务跳过）
gowind-cli backend grpc \
  --ddl schema.sql \
  --mapping tables.json \
  --orm ent --strategy per-table \
  --out /path/to/project

# 简写映射
gowind-cli backend grpc --dsn "$GOWIND_DSN" --tables user:identity,role:permission

# 只要生成产物、后处理自己控制
gowind-cli backend grpc --ddl schema.sql --mapping tables.json --skip-postprocess

# REST 网关（不生成 ORM/data，无后处理）
gowind-cli backend rest --ddl schema.sql --mapping tables.json \
  --service-name admin-portal --out /path/to/project
```

产物位于 `<out>/app/<服务名>/service/`，与 GUI 的 gRPC/REST 生成完全同源。

## frontend — 前端 CRUD 代码生成

```bash
# 预览将生成的文件清单（不写盘）
gowind-cli frontend gen \
  --openapi http://localhost:8000/q/openapi.yaml \
  --framework vue-vben \
  --module-path-map '{"role":"permission/role","api-audit-log":"log/api_audit_log"}' \
  --dry-run

# 写入前端项目（out 为前端 src 目录）
gowind-cli frontend gen \
  --openapi openapi.yaml \
  --framework vue-vben \
  --out D:/GoProject/go-wind-admin/frontend/admin/vue-vben/apps/admin/src

# 生成全部内容 JSON 到 stdout（供 AI 直接消费）
gowind-cli frontend gen --openapi openapi.yaml --framework react --stdout
```

| 参数 | 说明 |
|------|------|
| `--openapi` | OpenAPI 3.0 YAML 文件路径或 http(s) URL（必填） |
| `--framework` | `vue-element` / `vue-vben` / `react`（必填） |
| `--tags` | 要生成的服务 tag（缺省全部） |
| `--types` | `composable,page,drawer,router,locale` 子集（缺省全部；react 的 composable 自动映射为 hooks） |
| `--module-path-map` | 文件名 -> 模块路径 JSON（或 `@file.json`），决定 `views/app/<module>/...` 布局 |
| `--router-modules` | 路由分组 JSON（或 `@file`），缺省 `auto` 按 basePath 自动检测 |
| `--out` | 前端项目 **src 目录**（与 `--dry-run`/`--stdout` 三选一） |

**写盘语义**：普通文件创建或覆盖（manifest 标注 `created`/`overwritten`）；vue-vben 的国际化是**合并式片段**——目标 `locales/langs/{lang}/page.json`/`menu.json` 已存在时按键合并写回（`merged`，键已存在则原位替换），不存在时新建独立片段文件（`merged-new`）。

生成器类型前缀映射（`permissionservicev1_` 等来自领域 proto 包）维护在 `gowind/pkg/frontendgen/openapi.go` 的 `serviceTypePrefixes`，黄金样本测试保证与 GUI/TS 版行为一致。

## ai — AI 助手

凭证可用 flag 或环境变量：`GOWIND_AI_PROVIDER` / `GOWIND_AI_BASE_URL` / `GOWIND_AI_API_KEY` / `GOWIND_AI_MODEL`。

```bash
gowind-cli ai presets                          # 服务商预设
gowind-cli ai test                             # 连通性测试
gowind-cli ai ddl --requirements req.md        # 需求文档 -> MySQL DDL
gowind-cli ai partition --ddl schema.sql       # DDL -> 微服务划分建议
gowind-cli ai review --files internal/user/service.go,internal/role/service.go
```

典型的 AI 全链路组合：

```bash
gowind-cli ai ddl --requirements req.md --json | jq -r .content > schema.sql
gowind-cli ai partition --ddl schema.sql --json > partitions.json
# 依据 partitions.json 构建 mapping.json 后:
gowind-cli backend grpc --ddl schema.sql --mapping mapping.json
```

## config — 远程配置导出

```bash
gowind-cli config types
gowind-cli config services --path /path/to/project
gowind-cli config export --type consul --endpoint http://localhost:8500 \
  --project demo --path /path/to/project --dry-run
gowind-cli config export --type nacos --endpoint http://localhost:8848 \
  --project demo --group DEFAULT_GROUP --service admin --path /path/to/project
```

Etcd 暂未实现（如实报错）。

## scaffold — 脚手架

```bash
# 从模板创建项目（github 优先，失败回退 gitee）
gowind-cli scaffold project --name demo --module github.com/yourname/demo --parent-dir .

# 现有项目中添加服务
gowind-cli scaffold service --name order --servers grpc,rest --db-clients ent --path /path/to/project
```

## dev — 开发工具

```bash
gowind-cli dev buf                     # api 目录 buf generate
gowind-cli dev tidy                    # go mod tidy
gowind-cli dev ent                     # 全服务 ent generate
gowind-cli dev ent identity            # 单服务
gowind-cli dev wire identity           # wire（仅旧式 wire 服务；手写装配服务自动跳过，全服务时不带参数）
```

---

## AI 调用约定

1. 优先 `--json` + stdout 解析；不要解析 stderr（那是日志）。
2. 退出码非 0 时，stdout（`--json`）会输出 `{"error":"..."}`。
3. 长流程（backend grpc 的后处理链）日志在 stderr 实时输出，最终结果一次性输出到 stdout。
4. 涉及凭证（数据库密码、AI API Key）建议用环境变量传入。
