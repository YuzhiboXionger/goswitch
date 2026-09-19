package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// Manager MongoDB 表（集合）管理器
type Manager struct {
	client *mongo.Client
}

// TableExists 检查集合是否存在
func (m *Manager) TableExists(database, table string) (bool, error) {
	db := m.client.Database(database)
	collections, err := db.ListCollectionNames(context.Background(), bson.M{"name": table})
	if err != nil {
		return false, fmt.Errorf("failed to check collection existence: %w", err)
	}
	return len(collections) > 0, nil
}

// CreateTable 创建集合（DDL 参数被忽略，MongoDB 无需 DDL）
func (m *Manager) CreateTable(database, table, ddl string) error {
	db := m.client.Database(database)
	if err := db.CreateCollection(context.Background(), table); err != nil {
		return fmt.Errorf("failed to create collection %s: %w", table, err)
	}
	return nil
}

// DropTable 删除集合
func (m *Manager) DropTable(database, table string) error {
	db := m.client.Database(database)
	coll := db.Collection(table)
	if err := coll.Drop(context.Background()); err != nil {
		return fmt.Errorf("failed to drop collection %s: %w", table, err)
	}
	return nil
}

// TruncateTable 清空集合（删除所有文档）
func (m *Manager) TruncateTable(database, table string) error {
	db := m.client.Database(database)
	coll := db.Collection(table)
	_, err := coll.DeleteMany(context.Background(), bson.M{})
	if err != nil {
		return fmt.Errorf("failed to truncate collection %s: %w", table, err)
	}
	return nil
}
