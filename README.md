# goswitch - Go 语言数据库迁移工具

轻量级、高性能的异构数据库迁移工具，支持 MySQL、PostgreSQL、Oracle 和 MongoDB 之间的任意组合全量迁移。

## 功能特性

- ✅ 表结构自动迁移（字段类型、主键、注释）
- ✅ MongoDB 集合自动推断 schema
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

# MySQL -> MongoDB
cp configs/config.mongodb.example.yml config.yml
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

MongoDB 配置示例（MySQL → MongoDB）：

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  port: 3306
  database: "source_db"
  username: "root"
  password: "123456"

target:
  type: MONGODB
  host: "127.0.0.1"
  port: 27017
  database: "target_db"
  username: ""                # MongoDB 无认证时留空
  password: ""
  drop_target: true
  batch_size: 5000
  parallel: 4
```

### 2. 执行迁移

```bash
./goswitch run -c config.yml

# 输出到日志文件（同时获得日志和失败表的报告）
./goswitch run -c config.yml -l migration.log
```

## 配置说明

### 支持的数据库

| 数据库 | 类型标识 | 默认端口 |
|--------|----------|----------|
| MySQL | `MYSQL` | 3306 |
| PostgreSQL | `POSTGRESQL` | 5432 |
| Oracle | `ORACLE` | 1521 |
| MongoDB | `MONGODB` | 27017 |

### 源端配置 (source)

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| type | ✅ | 数据库类型 | MYSQL / POSTGRESQL / ORACLE / MONGODB |
| host | ✅ | 主机地址 | 127.0.0.1 |
| port | ❌ | 端口号（自动识别） | 3306 / 5432 / 1521 / 27017 |
| database | ✅ | 数据库名 | source_db |
| username | ✅* | 用户名（MongoDB 可选） | root |
| password | ❌ | 密码 | 123456 |
| fetch_size | ❌ | 批次大小 | 10000 |
| includes | ❌ | 包含的表（逗号分隔） | users,orders |
| excludes | ❌ | 排除的表（逗号分隔） | logs,tmp |
| table_mapper | ❌ | 表名正则映射 | 见示例 |

> *MongoDB 支持无认证模式，username 和 password 可以留空。

### 目标端配置 (target)

| 参数 | 必填 | 说明 | 示例 |
|------|------|------|------|
| type | ✅ | 数据库类型 | MYSQL / POSTGRESQL / ORACLE / MONGODB |
| host | ✅ | 主机地址 | 127.0.0.1 |
| port | ❌ | 端口号（自动识别） | 3306 / 5432 / 1521 / 27017 |
| database | ✅ | 数据库名 | target_db |
| username | ✅* | 用户名（MongoDB 可选） | root |
| password | ❌ | 密码 | 123456 |
| drop_target | ❌ | 是否先删后建 | true |
| table_name_case | ❌ | 表名大小写 | NONE/UPPER/LOWER |
| batch_size | ❌ | 写入批次大小 | 10000 |
| parallel | ❌ | 并行迁移表数量 | 1 |
| commit_interval | ❌ | 事务提交间隔 | 10 |
| pipeline | ❌ | 流水线模式 | false |

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

## MongoDB 迁移说明

### 概念映射

| 关系型数据库 | MongoDB | 说明 |
|---|---|---|
| database | database | 相同概念 |
| table | collection | 集合即表 |
| column | field | 自动采样文档推断 |
| primary key | `_id` | MongoDB 固定主键 |
| row | document | 文档即行 |

### 数据类型映射

| MongoDB BSON 类型 | PostgreSQL | MySQL | Oracle |
|---|---|---|---|
| ObjectId | varchar(24) | varchar(24) | VARCHAR2(24) |
| String | text | text | CLOB |
| Int32 | integer | integer | NUMBER(10) |
| Int64 | bigint | bigint | NUMBER(19) |
| Double | double precision | double precision | BINARY_DOUBLE |
| Boolean | boolean | tinyint(1) | NUMBER(1) |
| Decimal128 | numeric | decimal(38,9) | NUMBER(38,9) |
| Document/Array | jsonb | json | CLOB |
| Null | text | text | VARCHAR2(4000) |

### 嵌套文档处理

MongoDB 文档中的嵌套对象和数组会被序列化为 JSON 字符串存储。

## 从源码编译

```bash
# 克隆项目
git clone <repo-url>
cd goswitch

# 下载依赖
go mod tidy

# 编译
go build -o goswitch.exe ./cmd/goswitch/

# 交叉编译 Linux 版本
$env:GOOS="linux"; $env:GOARCH="amd64"; go build -o goswitch ./cmd/goswitch/
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
     ┌────────────┬───────────┼───────────┬────────────┐
     ▼            ▼           ▼           ▼            ▼
 ┌────────┐  ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
 │ MySQL  │  │PostgreSQL│ │ Oracle  │ │ MongoDB │ │  ...    │
 │ Driver │  │  Driver  │ │ Driver  │ │ Driver  │ │  更多   │
 └────────┘  └─────────┘ └─────────┘ └─────────┘ └─────────┘
```

## 后续计划

- [ ] 增量同步（基于时间戳/自增ID）
- [ ] 变化量同步（CDC）
- [x] PostgreSQL 支持 ✅
- [x] Oracle 支持 ✅
- [x] MongoDB 支持 ✅
- [ ] 达梦支持
- [ ] Web 管理界面

## 许可证

MIT License
