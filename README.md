#  MCP 工具

这是一个用 Go 实现的 MCP (Model Context Protocol) 工具，可以让 AI 助手（如通义灵码）直接查询和操作 MySQL 数据库 读网页、生成文档、压测。

## 功能特性

- ✅ 列出所有数据库
- ✅ 列出数据库中的所有表
- ✅ 查看表结构
- ✅ 执行 SQL 查询（只读）
- ✅ 查看表索引
- ✅ 支持通过环境变量配置不同项目的数据库
- ✅ 新增网页阅读功能
- ✅ 新增文档生成功能
- ✅ 新增并发请求压测功能

## 安装

### 1. 编译项目

```bash
go mod download
go build -o mysql-mcp
```

### 2. 配置 MCP

在通义灵码或其他支持 MCP 的 IDE 中配置此工具。

#### 方法一：工作区级别配置（推荐）

在项目根目录创建 `.kiro/settings/mcp.json`：

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "root",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": "your_database"
      },
      "disabled": false
    }
  }
}
```

#### 方法二：用户级别配置

在 `~/.kiro/settings/mcp.json` 中配置：

```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "args": [],
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_PORT": "3306",
        "MYSQL_USER": "root",
        "MYSQL_PASSWORD": "your_password",
        "MYSQL_DATABASE": ""
      },
      "disabled": false
    }
  }
}
```

## 环境变量配置

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| MYSQL_HOST | MySQL 服务器地址 | localhost |
| MYSQL_PORT | MySQL 端口 | 3306 |
| MYSQL_USER | 数据库用户名 | root |
| MYSQL_PASSWORD | 数据库密码 | (空) |
| MYSQL_DATABASE | 默认数据库名 | (空) |

## 在不同项目中使用

### 方案 1：每个项目单独配置

在每个项目的 `.kiro/settings/mcp.json` 中配置不同的数据库连接：

**项目 A：**
```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "localhost",
        "MYSQL_DATABASE": "project_a_db",
        "MYSQL_USER": "user_a",
        "MYSQL_PASSWORD": "pass_a"
      }
    }
  }
}
```

**项目 B：**
```json
{
  "mcpServers": {
    "mysql": {
      "command": "/path/to/mysql-mcp",
      "env": {
        "MYSQL_HOST": "192.168.1.100",
        "MYSQL_DATABASE": "project_b_db",
        "MYSQL_USER": "user_b",
        "MYSQL_PASSWORD": "pass_b"
      }
    }
  }
}
```

### 方案 2：使用 .env 文件

在项目根目录创建 `.env` 文件（记得添加到 .gitignore）：

```env
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=root
MYSQL_PASSWORD=your_password
MYSQL_DATABASE=your_database
```

然后在 MCP 配置中使用脚本加载环境变量：

```json
{
  "mcpServers": {
    "mysql": {
      "command": "bash",
      "args": ["-c", "source .env && /path/to/mysql-mcp"]
    }
  }
}
```

## 可用工具（18 个强大功能）

### 基础查询工具

#### 1. list_databases - 列出数据库
列出 MySQL 服务器上的所有数据库。

**参数：**
- `pattern` (可选): 过滤模式，使用 SQL LIKE 语法

**触发场景：**
```
列出所有数据库
有哪些数据库
显示数据库列表
列出名称包含 test 的数据库
```

#### 2. list_tables - 列出表
列出指定数据库中的所有表。

**参数：**
- `database` (必需): 数据库名称

**触发场景：**
```
列出 mydb 数据库中的所有表
mydb 有哪些表
显示表列表
```

#### 3. describe_table - 查看表结构
查看表的详细结构信息，包括字段名、类型、是否为空、键类型、默认值等。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称

**触发场景：**
```
查看 users 表的结构
users 表有哪些字段
显示 users 表的设计
users 表的字段类型是什么
```

#### 4. execute_query - 执行查询
执行 SQL 查询语句（SELECT、SHOW、DESCRIBE）。

**参数：**
- `query` (必需): 要执行的 SQL 查询语句
- `limit` (可选): 返回结果的最大行数，默认 100

**触发场景：**
```
查询 users 表中的所有数据
查询年龄大于 18 的用户
统计每个城市的用户数量
找出最近 7 天注册的用户
执行 SQL: SELECT * FROM orders WHERE status = 'pending'
```

### 性能分析工具

#### 5. show_indexes - 查看索引
查看表的索引信息，包括索引名称、类型、字段等。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称

**触发场景：**
```
查看 users 表的索引
users 表有哪些索引
显示索引信息
这个表的索引优化得怎么样
```

#### 6. get_table_stats - 表统计信息
获取表的统计信息，包括行数、数据大小、索引大小、创建时间等。

**参数：**
- `database` (必需): 数据库名称
- `table` (可选): 表名称，不指定则返回所有表

**触发场景：**
```
users 表有多少行数据
查看表的大小
哪个表占用空间最大
显示所有表的统计信息
这个表什么时候创建的
```

#### 7. analyze_column - 字段数据分析
分析某个字段的数据分布，包括唯一值数量、最大值、最小值、空值数量、最常见的值等。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称
- `column` (必需): 字段名称

**触发场景：**
```
分析 users 表的 age 字段
email 字段有多少个唯一值
status 字段的数据分布
哪些值最常见
有多少空值
```

### 数据库结构工具

#### 8. show_foreign_keys - 查看外键关系
查看表的外键关系，了解表之间的关联。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称

**触发场景：**
```
查看 orders 表的外键
这个表关联了哪些表
显示表关系
外键约束有哪些
```

#### 9. search_schema - 搜索表或字段
在数据库中搜索表名或字段名。

**参数：**
- `database` (必需): 数据库名称
- `keyword` (必需): 搜索关键词
- `search_type` (可选): table/column/both，默认 both

**触发场景：**
```
找到包含 user 的表
哪个表有 email 字段
搜索包含 created 的字段
找到所有带 _id 的字段
```

#### 10. show_create_table - 查看建表语句
查看表的完整建表语句（CREATE TABLE）。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称

**触发场景：**
```
显示 users 表的建表语句
生成 CREATE TABLE 语句
查看表的完整定义
导出表结构
```

#### 11. show_triggers - 查看触发器
查看表的触发器信息。

**参数：**
- `database` (必需): 数据库名称
- `table` (可选): 表名称，不指定则返回所有触发器

**触发场景：**
```
查看 users 表的触发器
有哪些触发器
显示所有触发器
这个表有自动化逻辑吗
```

#### 12. show_table_charset - 查看字符集
查看表的字符集和排序规则信息。

**参数：**
- `database` (必需): 数据库名称
- `table` (必需): 表名称

**触发场景：**
```
查看 users 表的字符集
这个表用的什么编码
显示排序规则
为什么会有乱码
```

### 监控和诊断工具

#### 13. show_variables - 查看系统变量
查看 MySQL 系统变量配置。

**参数：**
- `pattern` (可选): 变量名模式，如 '%timeout%'

**触发场景：**
```
查看所有系统变量
显示超时相关的配置
max_connections 是多少
查看字符集配置
```

#### 14. show_status - 查看数据库状态
查看 MySQL 运行状态信息。

**参数：**
- `pattern` (可选): 状态变量名模式

**触发场景：**
```
查看数据库状态
当前有多少连接
显示查询统计
慢查询有多少
```

#### 15. show_processlist - 查看正在执行的查询
查看当前正在执行的查询和连接。

**触发场景：**
```
查看正在执行的查询
有哪些活动连接
显示当前进程
哪些查询在运行
有慢查询吗
```

### 扩展工具

#### 16. read_webpage_with_browser - 网页阅读工具
使用浏览器打开并读取网页内容。

**参数：**
- `url` (必需): 要读取的网页 URL

**触发场景：**
```
阅读这个网页的内容
帮我打开这个文档链接
分析这个页面的信息
```

#### 17. document_generator - 文档生成工具
生成标准化 Markdown 文档。

**参数：**
- `type` (必需): 文档类型（api/table/module/custom）
- `title` (必需): 文档标题
- `content` (必需): 原始内容
- `format` (可选): 输出格式，默认为 markdown

**触发场景：**
```
生成接口文档
创建表结构文档
制作模块设计文档
生成自定义文档
```

#### 18. concurrent_request_runner - 并发请求工具
执行并发 HTTP 请求，支持压力测试。

**参数：**
- `url` (必需): 请求链接
- `method` (可选): 请求方法，默认 GET
- `headers_json` (可选): 请求头 JSON 字符串
- `params_json` (可选): 请求参数 JSON 字符串
- `threads` (必需): 并发线程数
- `iterations` (必需): 每线程执行次数
- `random_param_json` (可选): 随机参数规则 JSON 字符串

**触发场景：**
```
对这个接口进行压力测试
并发请求验证接口稳定性
模拟多个用户同时访问
```

## 安全说明

- 此工具只允许执行只读查询（SELECT、SHOW、DESCRIBE）
- 不支持 INSERT、UPDATE、DELETE 等修改操作
- 建议使用只读权限的数据库用户
- 不要在配置文件中硬编码敏感信息，使用环境变量或 .env 文件

## 故障排查

### 连接失败
1. 检查 MySQL 服务是否运行
2. 验证主机地址和端口是否正确
3. 确认用户名和密码是否正确
4. 检查数据库用户是否有相应权限

### 工具未显示
1. 检查 MCP 配置文件路径是否正确
2. 确认可执行文件路径是否正确
3. 在 IDE 中重新加载 MCP 服务器

## 开发

### 添加新工具

在 `main.go` 的 `registerTools` 函数中添加新工具：

```go
s.AddTool(mcp.NewTool("tool_name",
    mcp.WithDescription("工具描述"),
    mcp.WithString("param1",
        mcp.Description("参数描述"),
        mcp.Required(true),
    ),
), handlerFunction)
```

在 `handlers.go` 中实现处理函数：

```go
func handlerFunction(ctx context.Context, request map[string]interface{}) (*string, error) {
    // 实现逻辑
    return &result, nil
}
```

## 许可证

MIT License