# goswitch 使用说明

## 目录

- [快速开始](#快速开始)
- [安装方法](#安装方法)
- [配置说明](#配置说明)
- [使用示例](#使用示例)
- [命令行参数](#命令行参数)
- [输出说明](#输出说明)
- [常见问题](#常见问题)

---

## 快速开始

### 1. 下载可执行文件

从项目根目录获取编译好的 `goswitch.exe`（Windows）或 `goswitch`（Linux/Mac）。

### 2. 创建配置文件

复制示例配置并修改：

```bash
cp configs/config.example.yml config.yml
```

### 3. 执行迁移

```bash
# Windows
goswitch.exe run -c config.yml

# Linux/Mac
./goswitch run -c config.yml

# 输出到日志文件
./goswitch run -c config.yml -l migration.log
```

---

## 安装方法

### 方式一：直接使用编译好的文件

项目根目录的 `goswitch.exe`（Windows）或 `goswitch`（Linux）是单文件可执行程序，无需安装任何依赖。

### 方式二：从源码编译

**前置要求**：
- Go 1.18 或更高版本
- Git（可选）

**编译步骤**：

```bash
# 1. 克隆项目（如果需要）
git clone <repo-url>
cd goswitch

# 2. 下载依赖
go mod tidy

# 3. 编译当前平台
go build -o goswitch.exe ./cmd/goswitch/

# 4. 交叉编译 Linux 版本
GOOS=linux GOARCH=amd64 go build -o goswitch ./cmd/goswitch/
```

---

## 配置说明

配置文件使用 YAML 格式，包含两个部分：`source`（源端）、`target`（目标端）。

### 支持的数据库

| 数据库 | 类型标识 | 默认端口 |
|--------|----------|----------|
| MySQL | `MYSQL` | 3306 |
| PostgreSQL | `POSTGRESQL` | 5432 |
| Oracle | `ORACLE` | 1521 |

支持任意组合的跨数据库迁移，如 MySQL → PostgreSQL、Oracle → MySQL 等。

### 完整配置示例

```yaml
# ============================================================
# 源端数据库配置
# ============================================================
source:
  # 数据库类型（必填）
  # 支持: MYSQL, POSTGRESQL, ORACLE
  type: MYSQL

  # 主机地址（必填）
  host: "127.0.0.1"

  # 端口号（可选）
  # MySQL: 3306, PostgreSQL: 5432, Oracle: 1521
  port: 3306

  # 数据库名（必填）
  database: "source_db"

  # 用户名（必填）
  username: "root"

  # 密码（可选）
  password: "123456"

  # 批次读取大小（可选，默认 10000）
  # 越大内存占用越多，但速度可能更快
  fetch_size: 10000

  # 包含的表（可选，逗号分隔）
  # 留空表示迁移所有表
  # 示例: "users,orders,products"
  includes: ""

  # 排除的表（可选，逗号分隔）
  # 当 includes 为空时生效
  # 示例: "tmp_,test_"
  excludes: ""

  # 表名正则映射（可选）
  # 按数组顺序依次应用
  table_mapper:
    - from_pattern: "^"           # 正则表达式
      to_value: "t_"              # 替换为

# ============================================================
# 目标端数据库配置
# ============================================================
target:
  # 数据库类型（必填）
  # 支持: MYSQL, POSTGRESQL, ORACLE
  type: POSTGRESQL

  # 主机地址（必填）
  host: "127.0.0.1"

  # 端口号（可选）
  port: 5432

  # 数据库名（必填）
  database: "target_db"

  # 用户名（必填）
  username: "postgres"

  # 密码（可选）
  password: "123456"

  # 是否先删除目标表再创建（可选，默认 false）
  # true: DROP TABLE IF EXISTS → CREATE TABLE
  # false: 如果表存在则 TRUNCATE，不存在则创建
  drop_target: true

  # 表名大小写转换（可选，默认 NONE）
  # NONE:   保持原样
  # UPPER:  转大写
  # LOWER:  转小写
  table_name_case: NONE

  # 写入批次大小（可选，默认 10000）
  batch_size: 10000

  # 并行迁移表数量（可选，默认 1）
  # 多表同时迁移，提高整体速度
  parallel: 1

  # 事务提交间隔（可选，默认 10）
  # 每 N 个批次提交一次事务
  commit_interval: 10

  # 是否启用流水线模式（可选，默认 false）
  # 读写并行，提高单表迁移速度
  pipeline: false
```

### 配置参数详解

#### source（源端）

| 参数 | 必填 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| type | ✅ | string | - | 数据库类型：`MYSQL`, `POSTGRESQL`, `ORACLE` |
| host | ✅ | string | - | 数据库主机地址 |
| port | ❌ | int | 自动 | 数据库端口（根据类型自动判断） |
| database | ✅ | string | - | 数据库名称 |
| username | ✅ | string | - | 登录用户名 |
| password | ❌ | string | "" | 登录密码 |
| fetch_size | ❌ | int | 10000 | 每次读取的行数 |
| includes | ❌ | string | "" | 包含的表名，逗号分隔 |
| excludes | ❌ | string | "" | 排除的表名，逗号分隔 |
| table_mapper | ❌ | array | [] | 表名正则映射规则 |

#### target（目标端）

| 参数 | 必填 | 类型 | 默认值 | 说明 |
|------|------|------|--------|------|
| type | ✅ | string | - | 数据库类型：`MYSQL`, `POSTGRESQL`, `ORACLE` |
| host | ✅ | string | - | 数据库主机地址 |
| port | ❌ | int | 自动 | 数据库端口 |
| database | ✅ | string | - | 数据库名称 |
| username | ✅ | string | - | 登录用户名 |
| password | ❌ | string | "" | 登录密码 |
| drop_target | ❌ | bool | false | 是否先删除再创建 |
| table_name_case | ❌ | string | NONE | 表名大小写转换 |
| batch_size | ❌ | int | 10000 | 写入批次大小 |
| parallel | ❌ | int | 1 | 并行迁移表数量 |
| commit_interval | ❌ | int | 10 | 事务提交间隔 |
| pipeline | ❌ | bool | false | 是否启用流水线模式 |

---

## 使用示例

### 示例 1：MySQL → PostgreSQL 迁移

```yaml
source:
  type: MYSQL
  host: "192.168.1.100"
  port: 3306
  database: "production"
  username: "readonly"
  password: "password123"

target:
  type: POSTGRESQL
  host: "192.168.1.200"
  port: 5432
  database: "backup"
  username: "postgres"
  password: "password456"
  drop_target: true
```

---

### 示例 2：Oracle → MySQL 迁移

```yaml
source:
  type: ORACLE
  host: "10.174.18.36"
  port: 1521
  database: "ORCL"
  username: "system"
  password: "oracle123"

target:
  type: MYSQL
  host: "127.0.0.1"
  port: 3306
  database: "target_db"
  username: "root"
  password: "123456"
  drop_target: true
```

---

### 示例 3：只迁移指定表

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"
  includes: "users,orders,products"       # 指定表名

target:
  type: POSTGRESQL
  host: "127.0.0.1"
  database: "target_db"
  username: "postgres"
  password: "123456"
  drop_target: true
```

---

### 示例 4：排除某些表

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"
  excludes: "tmp_users,tmp_orders,test_data"   # 排除的表

target:
  type: MYSQL
  host: "127.0.0.1"
  database: "target_db"
  username: "root"
  password: "123456"
```

---

### 示例 5：添加表名前缀

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"
  table_mapper:
    - from_pattern: "^"
      to_value: "t_"

target:
  type: POSTGRESQL
  host: "127.0.0.1"
  database: "target_db"
  username: "postgres"
  password: "123456"
```

---

### 示例 6：表名转大写

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"

target:
  type: POSTGRESQL
  host: "127.0.0.1"
  database: "target_db"
  username: "postgres"
  password: "123456"
  table_name_case: UPPER         # 转大写
```

---

### 示例 7：并行迁移优化

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"

target:
  type: POSTGRESQL
  host: "127.0.0.1"
  database: "target_db"
  username: "postgres"
  password: "123456"
  parallel: 4                    # 4个表同时迁移
  pipeline: true                 # 启用流水线模式
```

---

### 示例 8：大批量数据优化

```yaml
source:
  type: MYSQL
  host: "127.0.0.1"
  database: "source_db"
  username: "root"
  password: "123456"
  fetch_size: 50000              # 增大读取批次

target:
  type: POSTGRESQL
  host: "127.0.0.1"
  database: "target_db"
  username: "postgres"
  password: "123456"
  batch_size: 50000              # 增大写入批次
  commit_interval: 20            # 增大事务间隔
```

---

### 示例 9：Oracle RAC 连接

```yaml
source:
  type: ORACLE
  host: "10.174.18.36"
  port: 1521
  database: "ORCL"
  username: "system"
  password: "oracle#123"         # 特殊字符会自动URL编码

target:
  type: MYSQL
  host: "127.0.0.1"
  port: 3306
  database: "target_db"
  username: "root"
  password: "123456"
```

---

## 命令行参数

### 基本用法

```bash
goswitch run -c <配置文件路径> [选项]
```

### 参数列表

| 参数 | 简写 | 必填 | 说明 |
|------|------|------|------|
| `--config` | `-c` | ✅ | 配置文件路径 |
| `--log-file` | `-l` | ❌ | 日志文件路径 |

### 帮助信息

```bash
# 查看主帮助
goswitch --help

# 查看 run 命令帮助
goswitch run --help

# 查看版本
goswitch version
```

---

## 输出说明

### 迁移过程输出

```
╔════════════════════════════════════════════════════════════╗
║                    goswitch v1.2.0                        ║
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║                      配置信息                             ║
╠════════════════════════════════════════════════════════════╣
║  源端配置:                                                ║
║    类型:     MYSQL                                        ║
║    地址:     127.0.0.1:3306                                ║
║    数据库:   source_db                                     ║
║  ...                                                       ║
╚════════════════════════════════════════════════════════════╝

╔════════════════════════════════════════════════════════════╗
║                      表信息预览                           ║
╠════════════════════════════════════════════════════════════╣
║  序号 表名                           列数   数据量          ║
║  ──────────────────────────────────────────────────────── ║
║  1    users                          8      15.2K          ║
║  ...                                                       ║
╚════════════════════════════════════════════════════════════╝

[1/3] 开始迁移: users
  [████████████████░░░░░░░░░░░░░░] 53.2% | 8.1K 行 | 2.5K 行/秒 | 已用: 3.2s | ETA: 2.8s
  ✓ users 迁移完成
    总行数:   15.2K
    总耗时:   6.1s
    平均速度: 2.5K 行/秒
    数据量:   ~1.5 MB
```

### 迁移完成汇总

```
╔════════════════════════════════════════════════════════════╗
║                    迁移完成汇总                           ║
╠════════════════════════════════════════════════════════════╣
║  基本信息:                                                ║
║    总耗时:     1h32m15s                                   ║
║    总表数:     3                                          ║
║    成功:       3                                          ║
║    并行度:     1                                          ║
║                                                           ║
║  性能统计:                                                ║
║    总行数:     188.3M                                     ║
║    平均速度:   34.1K 行/秒                                ║
║    吞吐量:     3.2 MB/s                                   ║
║    数据量:     ~17.7 GB                                   ║
║                                                           ║
║  资源使用:                                                ║
║    批次大小:   10000                                      ║
║    写入批次:   10000                                      ║
║    事务间隔:   10                                         ║
║    流水线:     false                                      ║
║    内存使用:   45.2 MB                                    ║
║    GC次数:     1247                                       ║
║                                                           ║
╚════════════════════════════════════════════════════════════╝
```

### 迁移失败报告

当有表迁移失败时，会自动生成报告文件 `migration_report_日期_时间.txt`：

```
╔════════════════════════════════════════════════════════════╗
║                    迁移报告                               ║
╚════════════════════════════════════════════════════════════╝

基本信息:
  生成时间:   2026-09-08 15:04:05
  总耗时:     4h32m15s
  源端:       root@127.0.0.1:3306/source_db
  目标端:     postgres@127.0.0.1:5432/target_db

迁移统计:
  总表数:     3
  成功:       2
  失败:       1

失败详情:
────────────────────────────────────────────────────────────
表名:   RPT_ZQYY_VI_ZB_RESULT_DAY
错误:   failed to write: pq: duplicate key value violates unique constraint
耗时:   2h15m30s
────────────────────────────────────────────────────────────

所有表迁移结果:
────────────────────────────────────────────────────────────
表名                                     状态     行数         耗时
────────────────────────────────────────────────────────────
users                                    ✓ 成功   15.2K        6.1s
orders                                   ✓ 成功   98.5K        12.3s
RPT_ZQYY_VI_ZB_RESULT_DAY               ✗ 失败   0            2h15m
```

---

## 常见问题

### Q1: 连接数据库失败

**错误信息**：
```
failed to connect source database: dial tcp 127.0.0.1:3306: connectex: No connection could be made because the target machine actively refused it.
```

**解决方案**：
1. 检查数据库服务是否启动
2. 检查主机和端口是否正确
3. 检查防火墙是否允许连接
4. 检查用户是否有远程访问权限

---

### Q2: 权限不足

**错误信息**：
```
failed to list tables: command denied to user 'xxx'@'xxx' for table 'xxx'
```

**解决方案**：

源端用户需要的权限：
```sql
-- MySQL
GRANT SELECT ON source_db.* TO 'user'@'%';

-- PostgreSQL
GRANT CONNECT ON DATABASE source_db TO user;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO user;

-- Oracle
GRANT SELECT ANY TABLE TO user;
```

目标端用户需要的权限：
```sql
-- MySQL
GRANT ALL PRIVILEGES ON target_db.* TO 'user'@'%';

-- PostgreSQL
GRANT ALL PRIVILEGES ON DATABASE target_db TO user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO user;

-- Oracle
GRANT CREATE ANY TABLE TO user;
GRANT INSERT ANY TABLE TO user;
```

---

### Q3: 表已存在错误

**错误信息**：
```
failed to create table: Table 'xxx' already exists
```

**解决方案**：

在配置中设置 `drop_target: true`，会先删除再创建：
```yaml
target:
  drop_target: true
```

---

### Q4: 如何中断迁移

按 `Ctrl+C` 可以优雅中断迁移。当前正在迁移的表会完成，已迁移的数据会保留。

---

### Q5: 内存占用过高

**解决方案**：

减小批次大小：
```yaml
source:
  fetch_size: 5000        # 默认 10000，改为 5000

target:
  batch_size: 5000        # 默认 10000，改为 5000
```

---

### Q6: 迁移速度慢

**解决方案**：

1. **增大批次大小**：
   ```yaml
   source:
     fetch_size: 50000
   target:
     batch_size: 50000
   ```

2. **启用并行迁移**：
   ```yaml
   target:
     parallel: 4
   ```

3. **启用流水线模式**：
   ```yaml
   target:
     pipeline: true
   ```

---

### Q7: Oracle 特殊字符密码

如果 Oracle 密码包含 `#`, `@`, `/` 等特殊字符，程序会自动进行 URL 编码，无需手动处理。

---

### Q8: 如何查看迁移日志

使用 `-l` 参数输出到日志文件：
```bash
./goswitch run -c config.yml -l migration.log
```

日志文件包含：
- 配置信息
- 连接信息
- 每个表的迁移结果
- 错误信息

---

### Q9: 支持哪些数据库版本

| 数据库 | 最低版本 |
|--------|----------|
| MySQL | 5.7+ |
| PostgreSQL | 10+ |
| Oracle | 12c+ |

---

### Q10: 如何迁移视图

目前版本只迁移表（BASE TABLE），不迁移视图。后续版本会添加视图支持。

---

## 反馈与支持

如有问题或建议，请提交 Issue 或联系开发者。

---

## 更新日志

### v1.2.0 (2026-09-08)
- 新增 Oracle 数据库支持
- 新增跨数据库迁移（MySQL ↔ PostgreSQL ↔ Oracle）
- 新增日志文件输出功能
- 新增迁移失败报告生成
- 新增实时进度条显示
- 新增迁移统计信息
- 优化终端兼容性
- 优化输出格式

### v1.1.0 (2024-08-15)
- 新增 PostgreSQL 数据库支持
- 新增并行迁移功能
- 新增流水线模式

### v1.0.0 (2024-08-10)
- 初始版本
- 支持 MySQL → MySQL 全量迁移
- 支持表名正则映射
- 支持表名大小写转换
