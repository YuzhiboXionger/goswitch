package config

import "goswitch/pkg/database"

// Config 配置结构
type Config struct {
	Source SourceConfig `yaml:"source"`
	Target TargetConfig `yaml:"target"`
}

// SourceConfig 源端配置
type SourceConfig struct {
	Type        database.DBType `yaml:"type"`
	Host        string          `yaml:"host"`
	Port        int             `yaml:"port"`
	Database    string          `yaml:"database"`
	Username    string          `yaml:"username"`
	Password    string          `yaml:"password"`
	FetchSize   int             `yaml:"fetch_size"`   // 批次大小，默认 10000
	Includes    string          `yaml:"includes"`     // 包含的表，逗号分隔，为空表示全部
	Excludes    string          `yaml:"excludes"`     // 排除的表，逗号分隔
	TableMapper []RegexMapper   `yaml:"table_mapper"` // 表名正则映射
}

// TargetConfig 目标端配置
type TargetConfig struct {
	Type          database.DBType `yaml:"type"`
	Host          string          `yaml:"host"`
	Port          int             `yaml:"port"`
	Database      string          `yaml:"database"`
	Username      string          `yaml:"username"`
	Password      string          `yaml:"password"`
	DropTarget    bool            `yaml:"drop_target"`      // 是否先删除再创建
	TableNameCase string          `yaml:"table_name_case"`  // NONE/UPPER/LOWER
	BatchSize     int             `yaml:"batch_size"`       // 写入批次大小，默认 10000
	Parallel      int             `yaml:"parallel"`         // 并行迁移表数量，默认 1
	CommitInterval int            `yaml:"commit_interval"`  // 每N个batch提交一次事务，默认 10
	Pipeline      bool            `yaml:"pipeline"`         // 是否启用流水线并行模式，默认 false
}

// RegexMapper 正则映射规则
type RegexMapper struct {
	FromPattern string `yaml:"from_pattern"`
	ToValue     string `yaml:"to_value"`
}

// SetDefaults 设置默认值
func (c *Config) SetDefaults() {
	if c.Source.FetchSize <= 0 {
		c.Source.FetchSize = 10000
	}
	if c.Target.BatchSize <= 0 {
		c.Target.BatchSize = 10000
	}
	// 根据数据库类型设置默认端口
	if c.Source.Port <= 0 {
		switch c.Source.Type {
		case "POSTGRESQL":
			c.Source.Port = 5432
		case "ORACLE":
			c.Source.Port = 1521
		default:
			c.Source.Port = 3306
		}
	}
	if c.Target.Port <= 0 {
		switch c.Target.Type {
		case "POSTGRESQL":
			c.Target.Port = 5432
		case "ORACLE":
			c.Target.Port = 1521
		default:
			c.Target.Port = 3306
		}
	}
	if c.Target.Parallel <= 0 {
		c.Target.Parallel = 1
	}
	if c.Target.CommitInterval <= 0 {
		c.Target.CommitInterval = 10
	}
}
