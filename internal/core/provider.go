package core

import (
	"fmt"
	"goswitch/pkg/database"
)

// Factory 数据库方言工厂接口
// 后续添加新数据库只需实现此接口
type Factory interface {
	// Type 返回数据库类型
	Type() database.DBType

	// MetadataProvider 创建元数据查询提供者
	// conn 参数: SQL 数据库为 *sql.DB, MongoDB 为 *mongo.Client
	MetadataProvider(conn interface{}) MetadataProvider

	// DataReader 创建数据读取提供者
	// conn 参数: SQL 数据库为 *sql.DB, MongoDB 为 *mongo.Client
	DataReader(conn interface{}) DataReader

	// DataWriter 创建数据写入提供者
	// conn 参数: SQL 数据库为 *sql.DB, MongoDB 为 *mongo.Client
	DataWriter(conn interface{}) DataWriter

	// TableManager 创建表管理提供者
	// conn 参数: SQL 数据库为 *sql.DB, MongoDB 为 *mongo.Client
	TableManager(conn interface{}) TableManager
}

// 全局注册中心
var Registry = make(map[database.DBType]Factory)

// Register 注册数据库方言
func Register(factory Factory) {
	Registry[factory.Type()] = factory
}

// GetFactory 获取数据库方言工厂
func GetFactory(dbType database.DBType) (Factory, error) {
	f, ok := Registry[dbType]
	if !ok {
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
	return f, nil
}
