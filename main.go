package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var db *sql.DB

func main() {
	// 从环境变量获取数据库连接信息
	dbHost := getEnv("MYSQL_HOST", "localhost")
	dbPort := getEnv("MYSQL_PORT", "3306")
	dbUser := getEnv("MYSQL_USER", "root")
	dbPass := getEnv("MYSQL_PASSWORD", "root")
	dbName := getEnv("MYSQL_DATABASE", "")

	// 构建 DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4",
		dbUser, dbPass, dbHost, dbPort, dbName)

	// 连接数据库
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	// 创建 MCP 服务器
	s := server.NewMCPServer(
		"MySQL MCP Server",
		"2.0.0",
	)

	// 注册工具
	registerTools(s)

	// 启动服务器
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func registerTools(s *server.MCPServer) {
	// 1. 列出所有数据库
	s.AddTool(mcp.NewTool("list_databases",
		mcp.WithDescription("当用户询问“有哪些数据库”、“列出全部数据库”、“show databases”时调用。返回 MySQL 中的数据库列表。"),
		mcp.WithString("pattern",
			mcp.Description("可选的过滤模式，使用 SQL LIKE 语法"),
		),
	), listDatabases)

	// 2. 列出数据库中的所有表
	s.AddTool(mcp.NewTool("list_tables",
		mcp.WithDescription("当用户问“这个数据库有哪些表”、“列出所有表”、“show tables”时调用。返回数据库中的表列表。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
	), listTables)

	// 3. 查看表结构
	s.AddTool(mcp.NewTool("describe_table",
		mcp.WithDescription("当用户问“表结构是什么”、“字段有哪些”、“describe table”、“字段类型”等使用。返回字段名、类型、主键等。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
	), describeTable)

	// 4. 执行查询
	s.AddTool(mcp.NewTool("execute_query",
		mcp.WithDescription("当用户问“执行 SQL”、“查询数据”、“select 语句”、“运行 SQL”时调用。仅用于执行 SELECT/SHOW/DESCRIBE。"),
		mcp.WithString("query",
			mcp.Description("要执行的 SQL 查询语句"),
			mcp.Required(),
		),
		mcp.WithNumber("limit",
			mcp.Description("返回最大行数（默认100）"),
		),
	), executeQuery)

	// 5. 查看表索引
	s.AddTool(mcp.NewTool("show_indexes",
		mcp.WithDescription("当用户问“索引是什么”、“怎么看索引”、“show index”、“索引结构”时调用。返回索引字段和类型。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
	), showIndexes)

	// 6. 获取表统计信息
	s.AddTool(mcp.NewTool("get_table_stats",
		mcp.WithDescription("当用户问“表有多少行”、“表多大”、“表统计信息”、“容量情况”时调用。返回行数、大小等统计信息。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称，可选。不指定则返回所有表的统计"),
		),
	), getTableStats)

	// 7. 外键关系
	s.AddTool(mcp.NewTool("show_foreign_keys",
		mcp.WithDescription("当用户问“外键关系是什么”、“表关联”、“外键约束”、“依赖关系”时调用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
	), showForeignKeys)

	// 8. 搜索表或字段
	s.AddTool(mcp.NewTool("search_schema",
		mcp.WithDescription("当用户问“某个字段在哪个表”、“包含 XXX 的表”、“搜索表名/列名”时调用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("keyword",
			mcp.Description("搜索关键词"),
			mcp.Required(),
		),
		mcp.WithString("search_type",
			mcp.Description("table / column / both"),
			mcp.DefaultString("both"),
		),
	), searchSchema)

	// 9. 查看建表语句
	s.AddTool(mcp.NewTool("show_create_table",
		mcp.WithDescription("当用户问“建表语句是什么”、“show create table”、“导出表结构”时调用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
	), showCreateTable)

	// 10. 分析字段值分布
	s.AddTool(mcp.NewTool("analyze_column",
		mcp.WithDescription("当用户问“字段值分布”、“最大值最小值”、“空值数量”、“distinct 数量”等使用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
		mcp.WithString("column",
			mcp.Description("字段名称"),
			mcp.Required(),
		),
	), analyzeColumn)

	// 11. 查看触发器
	s.AddTool(mcp.NewTool("show_triggers",
		mcp.WithDescription("当用户问“触发器是什么”、“有哪些触发器”、“表的触发器”时调用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称，可选"),
		),
	), showTriggers)

	// 12. 查看系统变量
	s.AddTool(mcp.NewTool("show_variables",
		mcp.WithDescription("当用户问“数据库配置是什么”、“查看变量”、“show variables”时调用。"),
		mcp.WithString("pattern",
			mcp.Description("变量名过滤条件"),
		),
	), showVariables)

	// 13. 查看运行状态
	s.AddTool(mcp.NewTool("show_status",
		mcp.WithDescription("当用户问“数据库状态”、“连接数”、“性能信息”、“show status”时调用。"),
		mcp.WithString("pattern",
			mcp.Description("状态名过滤模式"),
		),
	), showStatus)

	// 14. 查看进程列表
	s.AddTool(mcp.NewTool("show_processlist",
		mcp.WithDescription("当用户问“有哪些 SQL 在执行”、“阻塞查询”、“连接状态”时调用。"),
	), showProcesslist)

	// 15. 查看表字符集
	s.AddTool(mcp.NewTool("show_table_charset",
		mcp.WithDescription("当用户问“表的字符集是什么”、“编码问题”、“排序规则是什么”时调用。"),
		mcp.WithString("database",
			mcp.Description("数据库名称"),
			mcp.Required(),
		),
		mcp.WithString("table",
			mcp.Description("表名称"),
			mcp.Required(),
		),
	), showTableCharset)

	// 16. 辅助ai阅读网页
	s.AddTool(mcp.NewTool("read_webpage_with_browser",
		mcp.WithDescription(`
			读取网页内容的专用工具。
			当用户要求“阅读网页/阅读文档/打开 URL”时，必须调用本工具。
			
			本工具要求：
			1. 禁止使用普通Web搜索。
			2. 禁止使用HTTP请求获取HTML。
			3. 必须调用 chrome-devtools-mcp 来打开网页。
			4. 必须 navigate → wait_for_dom → get_html/extract_text。
			5. 返回网页的真实渲染内容，用于后续生成文档、分析、处理。
			`),
		mcp.WithString("url",
			mcp.Description("用户希望阅读的网页 URL"),
			mcp.Required(),
		),
	), forceBrowserReadHandler)

	//17.生成文档
	s.AddTool(mcp.NewTool("document_generator",
		mcp.WithDescription(`
		生成标准化 Markdown 文档工具。
		支持类型：
		- api: 接口文档
		- table: 表结构文档
		- module: 模块设计文档
		- custom: 自定义文档
		
		使用说明：
		1. 输入 type/title/content。
		2. 输出 Markdown 格式文档。
		3. AI 可直接调用生成规范化文档。
	`),
		mcp.WithString("type",
			mcp.Description("文档类型：api / table / module / custom"),
			mcp.Required(),
		),
		mcp.WithString("title",
			mcp.Description("文档标题"),
			mcp.Required(),
		),
		mcp.WithString("content",
			mcp.Description("原始内容，如接口字段、表结构、业务说明等"),
			mcp.Required(),
		),
		mcp.WithString("format",
			mcp.Description("输出格式（目前仅支持 markdown）"),
			mcp.DefaultString("markdown"),
		),
	), documentGeneratorHandler)

	// 18. 压测工具
	s.AddTool(mcp.NewTool("concurrent_request_runner",
		mcp.WithDescription("执行并发 HTTP 请求，支持随机参数、线程数和每线程请求次数。headers/params/random_param 使用 JSON 字符串传入"),
		mcp.WithString("url", mcp.Description("请求链接"), mcp.Required()),
		mcp.WithString("method", mcp.Description("请求方法"), mcp.DefaultString("GET")),
		mcp.WithString("headers_json", mcp.Description("请求头 JSON 字符串，例如 '{\"Authorization\":\"Bearer xxx\"}'")),
		mcp.WithString("params_json", mcp.Description("请求参数 JSON 字符串")),
		mcp.WithNumber("threads", mcp.Description("并发线程数"), mcp.Required()),
		mcp.WithNumber("iterations", mcp.Description("每线程执行次数"), mcp.Required()),
		mcp.WithString("random_param_json", mcp.Description("随机参数规则 JSON 字符串，例如 '{\"id\":\"1-1000\"}'")),
	), concurrentRequestHandler)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
