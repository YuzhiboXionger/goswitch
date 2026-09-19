package mongodb

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Writer MongoDB 数据写入器
type Writer struct {
	client     *mongo.Client
	database   string
	collection string
	columns    []string
}

// Prepare 准备写入，存储目标集合信息
func (w *Writer) Prepare(database, table string, columns []string) error {
	w.database = database
	w.collection = table
	w.columns = columns
	return nil
}

// Write 批量写入数据到 MongoDB 集合
func (w *Writer) Write(records [][]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	db := w.client.Database(w.database)
	coll := db.Collection(w.collection)

	// 将行数据转为 BSON 文档
	docs := make([]interface{}, 0, len(records))
	for _, record := range records {
		doc := bson.M{}
		for i, col := range w.columns {
			if i < len(record) {
				val := record[i]
				// 尝试将 _id 字段转为 ObjectID
				if col == "_id" {
					val = convertToID(val)
				}
				doc[col] = val
			}
		}
		docs = append(docs, doc)
	}

	// 使用 InsertMany 批量写入，无序模式允许部分失败
	opts := options.InsertMany().SetOrdered(false)
	_, err := coll.InsertMany(context.Background(), docs, opts)
	if err != nil {
		return fmt.Errorf("failed to insert documents into %s: %w", w.collection, err)
	}

	return nil
}

// Finish 完成写入（MongoDB 无需手动提交）
func (w *Writer) Finish() error {
	return nil
}

// convertToID 尝试将值转为 MongoDB ObjectID，失败则保持原值
func convertToID(val interface{}) interface{} {
	if val == nil {
		return val
	}

	// 如果已经是 ObjectID，直接返回
	if _, ok := val.(bson.ObjectID); ok {
		return val
	}

	// 尝试将字符串转为 ObjectID
	if str, ok := val.(string); ok {
		if oid, err := bson.ObjectIDFromHex(str); err == nil {
			return oid
		}
		// 无法转为 ObjectID 时保持原值（MongoDB 允许任意类型作为 _id）
		return str
	}

	return val
}
