package database

import "fmt"

// DBType 数据库类型
type DBType string

const (
	MySQL      DBType = "MYSQL"
	PostgreSQL DBType = "POSTGRESQL"
	Oracle     DBType = "ORACLE"
	MongoDB    DBType = "MONGODB"
)

// ProductInfo 数据库产品元信息
type ProductInfo struct {
	Type        DBType
	Name        string
	QuoteChar   byte // 标识符引用字符: ` " 或空
	DefaultPort int
}

// Products 支持的数据库产品映射
var Products = map[DBType]ProductInfo{
	MySQL:      {Type: MySQL, Name: "MySQL", QuoteChar: '`', DefaultPort: 3306},
	PostgreSQL: {Type: PostgreSQL, Name: "PostgreSQL", QuoteChar: '"', DefaultPort: 5432},
	Oracle:     {Type: Oracle, Name: "Oracle", QuoteChar: '"', DefaultPort: 1521},
	MongoDB:    {Type: MongoDB, Name: "MongoDB", QuoteChar: 0, DefaultPort: 27017},
}

// GetProduct 获取数据库产品信息
func GetProduct(dbType DBType) (ProductInfo, error) {
	p, ok := Products[dbType]
	if !ok {
		return ProductInfo{}, fmt.Errorf("unsupported database type: %s", dbType)
	}
	return p, nil
}

// QuoteIdentifier 引用 SQL 标识符
func QuoteIdentifier(quoteChar byte, name string) string {
	return string(quoteChar) + name + string(quoteChar)
}
