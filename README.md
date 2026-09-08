# goswitch - Go 语言数据库迁移工具

轻量级、高性能的数据库迁移工具，支持 MySQL 和 PostgreSQL 数据库的全量迁移。

## 功能特性

- ✅ MySQL → MySQL 全量迁移
- ✅ PostgreSQL → PostgreSQL 全量迁移
- ✅ Oracle → Oracle 全量迁移
- ✅ MySQL → PostgreSQL 跨数据库迁移
- ✅ PostgreSQL → MySQL 跨数据库迁移
- ✅ MySQL → Oracle 跨数据库迁移
- ✅ Oracle → MySQL 跨数据库迁移
- ✅ PostgreSQL → Oracle 跨数据库迁移
- ✅ Oracle → PostgreSQL 跨数据库迁移
- ✅ 表结构自动迁移（字段类型、主键、注释）
- ✅ 表名正则映射（支持添加前缀、后缀等）
- ✅ 表名大小写转换
- ✅ 批量数据读写，高效稳定
- ✅ 支持优雅中断（Ctrl+C）

## 文档

- [详细使用说明](docs/USAGE.md) - 完整的配置说明、使用示例、常见问题
- [代码阅读指南](docs/CODE_GUIDE.md) - 代码结构、阅读顺序、核心流程
- [测试指南](docs/TESTING.md) - 测试环境准备、测试用例、性能测试

## 快速开始

### 1. 准备配置文件

复制示例配置：

```bash
# MySQL -> MySQL
cp configs/config.example.yml config.yml

# PostgreSQL -> PostgreSQL
cp configs/config.postgresql2postgresql.yml config.yml

# Oracle -> Oracle (或其他目标)
cp configs/config.oracle.example.yml config.yml

# MySQL -> PostgreSQL
cp configs/config.mysql2postgresql.yml config.yml

# PostgreSQL -> MySQL
cp configs/config.postgresql2mysql.yml config.yml
```

修改配置（以 MySQL -> PostgreSQL 为例）：

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  port: 3306
  database: "source_db"
  username: "root"
  password: "123456"
  fetch_size: 10000
  includes: ""                # 留空迁移所有表，逗号分隔指定表名
  excludes: ""                # 排除的表
  table_mapper:
    - from_pattern: "^"
      to_value: "t_"          # 给所有表添加 t_ 前缀

target:
  type: POSTGRESQL            # PostgreSQL 数据库
  host: "127.0.0.1"
  port: 5432                  # PostgreSQL 默认端口
  database: "target_db"
  username: "postgres"
  password: "123456"
  drop_target: true           # 先删后建
  table_name_case: NONE       # NONE/UPPER/LOWER
  batch_size: 10000
```

### 2. 执行迁移

```bash
./goswitch run -c config.yml
```

## 配置说明

### 源端配置 (source)

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| type | ✅ | 数据库类型 | MYSQL |
| host | ✅ | 主机地址 | 127.0.0.1 |
| port | ❌ | 端口号 | 3306 |
| database | ✅ | 数据库名 | source_db |
| username | ✅ | 用户名 | root |
| password | ❌ | 密码 | 123456 |
| fetch_size | ❌ | 批次大小 | 10000 |
| includes | ❌ | 包含的表（逗号分隔） | users,orders |
| excludes | ❌ | 排除的表（逗号分隔） | logs,tmp |
| table_mapper | ❌ | 表名正则映射 | 见示例 |

### 目标端配置 (target)

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| type | ✅ | 数据库类型 | MYSQL |
| host | ✅ | 主机地址 | 127.0.0.1 |
| port | ❌ | 端口号 | 3306 |
| database | ✅ | 数据库名 | target_db |
| username | ✅ | 用户名 | root |
| password | ❌ | 密码 | 123456 |
| drop_target | ❌ | 是否先删后建 | true |
| table_name_case | ❌ | 表名大小写 | NONE/UPPER/LOWER |
| batch_size | ❌ | 写入批次大小 | 10000 |

### 表名映射示例

添加前缀：
```yaml
table_mapper:
  - from_pattern: "^"
    to_value: "t_"
```

添加后缀：
```yaml
table_mapper:
  - from_pattern: "$"
    to_value: "_bak"
```

替换字符：
```yaml
table_mapper:
  - from_pattern: "^old_"
    to_value: "new_"
```

## 从源码编译

```bash
# 克隆项目
git clone <repo-url>
cd goswitch

# 编译
go build -o goswitch.exe ./cmd/goswitch/

# 或使用构建脚本
make build
```

## 技术架构

```
┌─────────────────────────────────────────────────────────────┐
│                      CLI (Cobra)                            │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   MigrationService                          │
│              (迁移主服务 - 协调整个迁移流程)                  │
└─────────────────────────────────────────────────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
   ┌──────────┐       ┌──────────┐       ┌──────────┐
   │ Metadata │       │  Reader  │       │  Writer  │
   │ Provider │       │ Provider │       │ Provider │
   └──────────┘       └──────────┘       └──────────┘
          │                   │                   │
          └───────────────────┼───────────────────┘
                              │
          ┌───────────────────┼───────────────────┐
          ▼                   ▼                   ▼
   ┌──────────┐       ┌──────────┐       ┌──────────┐
   │  MySQL   │       │PostgreSQL│       │  ...     │
   │  Driver  │       │  Driver  │       │  更多    │
   └──────────┘       └──────────┘       └──────────┘
```

## 后续计划

- [ ] 增量同步（基于时间戳/自增ID）
- [ ] 变化量同步（CDC）
- [x] PostgreSQL 支持 ✅
- [x] Oracle 支持 ✅
- [ ] 达梦支持
- [ ] Web 管理界面

## 许可证

MIT License
