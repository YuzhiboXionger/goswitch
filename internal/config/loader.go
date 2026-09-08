package config

import (
	"fmt"
	"net/url"
	"os"

	"gopkg.in/yaml.v3"
)

// Load 从文件加载配置
func Load(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// 设置默认值
	cfg.SetDefaults()

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// Validate 验证配置
func (c *Config) Validate() error {
	// 验证源端
	if c.Source.Type == "" {
		return fmt.Errorf("source.type is required")
	}
	if c.Source.Host == "" {
		return fmt.Errorf("source.host is required")
	}
	if c.Source.Database == "" {
		return fmt.Errorf("source.database is required")
	}
	if c.Source.Username == "" {
		return fmt.Errorf("source.username is required")
	}

	// 验证目标端
	if c.Target.Type == "" {
		return fmt.Errorf("target.type is required")
	}
	if c.Target.Host == "" {
		return fmt.Errorf("target.host is required")
	}
	if c.Target.Database == "" {
		return fmt.Errorf("target.database is required")
	}
	if c.Target.Username == "" {
		return fmt.Errorf("target.username is required")
	}

	return nil
}

// GetSourceDSN 获取源端 DSN
func (c *Config) GetSourceDSN() string {
	switch c.Source.Type {
	case "POSTGRESQL":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Source.Host, c.Source.Port, c.Source.Username, c.Source.Password, c.Source.Database)
	case "ORACLE":
		// Oracle DSN 格式: oracle://user:password@host:port/service_name
		// 密码需要 URL 编码以处理特殊字符（如 #、@、:）
		return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			url.QueryEscape(c.Source.Username), url.QueryEscape(c.Source.Password),
			c.Source.Host, c.Source.Port, c.Source.Database)
	default:
		// MySQL
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Source.Username, c.Source.Password, c.Source.Host, c.Source.Port, c.Source.Database)
	}
}

// GetTargetDSN 获取目标端 DSN
func (c *Config) GetTargetDSN() string {
	switch c.Target.Type {
	case "POSTGRESQL":
		return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
			c.Target.Host, c.Target.Port, c.Target.Username, c.Target.Password, c.Target.Database)
	case "ORACLE":
		// Oracle DSN 格式: oracle://user:password@host:port/service_name
		// 密码需要 URL 编码以处理特殊字符（如 #、@、:）
		return fmt.Sprintf("oracle://%s:%s@%s:%d/%s",
			url.QueryEscape(c.Target.Username), url.QueryEscape(c.Target.Password),
			c.Target.Host, c.Target.Port, c.Target.Database)
	default:
		// MySQL
		return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			c.Target.Username, c.Target.Password, c.Target.Host, c.Target.Port, c.Target.Database)
	}
}
