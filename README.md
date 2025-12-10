# mysql-user-migrate

Go CLI 工具，用于将 MySQL 用户（用户名、密码/认证、Host、权限）从源实例迁移到一个或多个目标实例，支持显式包含/排除用户、干跑、覆盖策略以及迁移报告输出。

## 快速开始
- 环境：Go 1.21+，本地可访问的 MySQL。
- 安装依赖：`make deps`
- 直接运行：  
  `go run ./cmd/mysql-user-migrate --source "user:pass@tcp(src:3306)/" --target "name=stg=user:pass@tcp(stg:3306)/" --include app_user --dry-run --report report.json`
- 配置文件运行：  
  `go run ./cmd/mysql-user-migrate --config config.example.yaml`

## 主要特性
- 用户筛选：`--include user1,user2`，`--exclude root,test`，支持 user@host。
- 多目标：重复 `--target` 或在配置文件中提供 targets，一对多迁移可并发执行（`--concurrency`）。
- 模式：`--dry-run` 生成计划与报告不落库；默认执行模式；`--drop-missing`/`--force-overwrite` 控制覆盖策略。
- 报告：终端摘要 + `--report` 输出 JSON（含每个目标的结果）。
- 安全：不记录明文密码，默认不迁移 root，需显式包含。

## 配置文件 (YAML/JSON)
参见 `config.example.yaml`，常用字段：
- `source`: 源 DSN
- `targets`: 目标数组（`name`, `dsn`）
- `include` / `exclude`
- `dry_run`, `drop_missing`, `force_overwrite`, `report_path`, `concurrency`, `verbose`

## 常用命令
- `make deps` 安装依赖
- `make fmt` 运行 gofmt/goimports
- `make lint` 运行 golangci-lint（如已安装）
- `make test` 运行单元测试
- `make run ARGS="--source ... --target ..."` 运行 CLI

## 运行时环境变量
- `SOURCE_DSN`, `TARGET_DSN` 或 `TARGET_DSN_LIST`（逗号分隔）可作为 DSN 输入。
- 参考 `.env.example` 填写，本工具不读取真实密码到日志。

## 现状与后续
当前为迁移骨架：包含配置/CLI 解析、源用户读取、目标迁移与报告输出。后续可补充：完整的集成测试（Docker Compose 启动 MySQL）、更细粒度的权限 diff、错误恢复与重试。***
