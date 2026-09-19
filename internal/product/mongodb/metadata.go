package mongodb

import (
	"context"
	"fmt"
	"goswitch/internal/core"
	"sort"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Metadata MongoDB 元数据提供者
type Metadata struct {
	client *mongo.Client
}

// ListTables 列出所有集合（映射为"表"）
func (m *Metadata) ListTables(database string) ([]string, error) {
	db := m.client.Database(database)
	collections, err := db.ListCollectionNames(context.Background(), bson.M{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	sort.Strings(collections)
	return collections, nil
}

// GetColumns 通过采样文档推断集合的列（字段）信息
func (m *Metadata) GetColumns(database, table string) ([]core.ColumnMeta, error) {
	db := m.client.Database(database)
	collection := db.Collection(table)

	// 采样最多 100 条文档来推断 schema
	limit := int64(100)
	opts := options.Find().SetLimit(limit)
	cursor, err := collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to sample documents: %w", err)
	}
	defer cursor.Close(context.Background())

	// 收集所有字段名和类型
	fieldTypes := make(map[string]string)
	var docs []bson.M
	if err := cursor.All(context.Background(), &docs); err != nil {
		return nil, fmt.Errorf("failed to decode documents: %w", err)
	}

	for _, doc := range docs {
		for key, value := range doc {
			// 将 _id 排除在普通列之外，它作为主键单独处理
			if key == "_id" {
				continue
			}
			if _, exists := fieldTypes[key]; !exists {
				fieldTypes[key] = getBSONTypeName(value)
			}
		}
	}

	// 按字段名排序，保证顺序一致
	fieldNames := make([]string, 0, len(fieldTypes))
	for name := range fieldTypes {
		fieldNames = append(fieldNames, name)
	}
	sort.Strings(fieldNames)

	// 构建 ColumnMeta 列表，_id 作为第一列
	columns := []core.ColumnMeta{
		{
			Name:     "_id",
			DataType: "objectId",
			Nullable: false,
			Comment:  "MongoDB 默认主键",
		},
	}

	for _, name := range fieldNames {
		columns = append(columns, core.ColumnMeta{
			Name:     name,
			DataType: fieldTypes[name],
			Nullable: true,
		})
	}

	return columns, nil
}

// GetPrimaryKeys 返回主键列表（MongoDB 固定为 _id）
func (m *Metadata) GetPrimaryKeys(database, table string) ([]string, error) {
	return []string{"_id"}, nil
}

// GenerateCreateTableDDL 生成集合的描述性 DDL（MongoDB 无传统 DDL，生成注释说明）
func (m *Metadata) GenerateCreateTableDDL(database, table string, columns []core.ColumnMeta, primaryKeys []string) (string, error) {
	// MongoDB 不需要真正的 CREATE TABLE DDL
	// 返回一个描述性字符串，TableManager.CreateTable 会忽略它直接创建集合
	ddl := fmt.Sprintf("-- MongoDB Collection: %s\n", table)
	ddl += fmt.Sprintf("-- Database: %s\n", database)
	ddl += fmt.Sprintf("-- Columns inferred from source schema:\n")
	for _, col := range columns {
		nullable := "NULL"
		if !col.Nullable {
			nullable = "NOT NULL"
		}
		ddl += fmt.Sprintf("--   %s %s %s\n", col.Name, col.DataType, nullable)
	}
	if len(primaryKeys) > 0 {
		ddl += fmt.Sprintf("-- Primary Key: %v\n", primaryKeys)
	}
	return ddl, nil
}

// getBSONTypeName 根据 Go 值推断类型名（使用 SQL 兼容的类型名，方便跨库迁移）
func getBSONTypeName(value interface{}) string {
	switch value.(type) {
	case bson.Decimal128:
		return "numeric"
	case bson.Raw:
		return "json"
	case bson.A:
		return "json"
	case bson.ObjectID:
		return "objectId" // 保持特殊标记，在各目标库中有专门映射
	case string:
		return "text"
	case bool:
		return "boolean"
	case int32:
		return "integer"
	case int64:
		return "bigint"
	case float64:
		return "double precision"
	case nil:
		return "text"
	default:
		return "text"
	}
}
