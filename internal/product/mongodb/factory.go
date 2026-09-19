package mongodb

import (
	"goswitch/internal/core"
	"goswitch/pkg/database"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

func init() {
	core.Register(&Factory{})
}

// Factory MongoDB 方言工厂
type Factory struct{}

func (f *Factory) Type() database.DBType {
	return database.MongoDB
}

func (f *Factory) MetadataProvider(conn interface{}) core.MetadataProvider {
	client := conn.(*mongo.Client)
	return &Metadata{client: client}
}

func (f *Factory) DataReader(conn interface{}) core.DataReader {
	client := conn.(*mongo.Client)
	return &Reader{client: client}
}

func (f *Factory) DataWriter(conn interface{}) core.DataWriter {
	client := conn.(*mongo.Client)
	return &Writer{client: client}
}

func (f *Factory) TableManager(conn interface{}) core.TableManager {
	client := conn.(*mongo.Client)
	return &Manager{client: client}
}
