package postgresql

import (
	"database/sql"
	"goswitch/internal/core"
	"goswitch/pkg/database"
)

func init() {
	core.Register(&Factory{})
}

// Factory PostgreSQL 方言工厂
type Factory struct{}

func (f *Factory) Type() database.DBType {
	return database.PostgreSQL
}

func (f *Factory) MetadataProvider(conn interface{}) core.MetadataProvider {
	db := conn.(*sql.DB)
	return &Metadata{db: db}
}

func (f *Factory) DataReader(conn interface{}) core.DataReader {
	db := conn.(*sql.DB)
	return &Reader{db: db}
}

func (f *Factory) DataWriter(conn interface{}) core.DataWriter {
	db := conn.(*sql.DB)
	return &Writer{db: db}
}

func (f *Factory) TableManager(conn interface{}) core.TableManager {
	db := conn.(*sql.DB)
	return &Manager{db: db}
}
