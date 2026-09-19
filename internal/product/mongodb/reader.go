package mongodb

import (
	"context"
	"encoding/json"
	"fmt"
	"goswitch/internal/core"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Reader MongoDB 数据读取器
type Reader struct {
	client *mongo.Client
}

// QueryData 读取集合数据，返回 DataBatch channel
func (r *Reader) QueryData(database, table string, columns []string, fetchSize int) (<-chan core.DataBatch, error) {
	db := r.client.Database(database)
	collection := db.Collection(table)

	// 只投影需要的列
	projection := bson.M{}
	for _, col := range columns {
		projection[col] = 1
	}
	opts := options.Find().SetProjection(projection)

	cursor, err := collection.Find(context.Background(), bson.M{}, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query collection %s: %w", table, err)
	}

	batchCh := make(chan core.DataBatch, 10)

	go func() {
		defer close(batchCh)
		defer cursor.Close(context.Background())

		var rows [][]interface{}
		for cursor.Next(context.Background()) {
			var doc bson.M
			if err := cursor.Decode(&doc); err != nil {
				batchCh <- core.DataBatch{Error: fmt.Errorf("failed to decode document: %w", err)}
				return
			}

			// 将文档按列顺序转为行
			row := make([]interface{}, len(columns))
			for i, col := range columns {
				val, exists := doc[col]
				if !exists {
					row[i] = nil
					continue
				}
				row[i] = convertBSONValue(val)
			}

			rows = append(rows, row)

			// 达到批次大小时发送
			if len(rows) >= fetchSize {
				batchCh <- core.DataBatch{Columns: columns, Rows: rows}
				rows = nil
			}
		}

		// 检查游标错误
		if err := cursor.Err(); err != nil {
			batchCh <- core.DataBatch{Error: fmt.Errorf("cursor error: %w", err)}
			return
		}

		// 发送剩余数据
		if len(rows) > 0 {
			batchCh <- core.DataBatch{Columns: columns, Rows: rows}
		}
	}()

	return batchCh, nil
}

// convertBSONValue 将 BSON 值转换为可写入其他数据库的 Go 值
func convertBSONValue(value interface{}) interface{} {
	switch v := value.(type) {
	case bson.ObjectID:
		return v.Hex()
	case bson.Decimal128:
		return v.String()
	case bson.A:
		// 数组序列化为 JSON 字符串
		data, _ := json.Marshal(v)
		return string(data)
	case bson.M, bson.D, map[string]interface{}, bson.Raw:
		// 嵌套对象序列化为 JSON 字符串
		data, _ := json.Marshal(v)
		return string(data)
	case nil:
		return nil
	default:
		return v
	}
}
