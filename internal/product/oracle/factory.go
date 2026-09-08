package oracle

import (
	"database/sql"
	"goswitch/internal/core"
	"goswitch/pkg/database"
)

func init() {
	core.Register(&Factory{})
}

// Factory Oracle 方言工厂
type Factory struct{}

func (f *Factory) Type() database.DBType {
	return database.Oracle
}

func (f *Factory) MetadataProvider(db *sql.DB) core.MetadataProvider {
	return &Metadata{db: db}
}

func (f *Factory) DataReader(db *sql.DB) core.DataReader {
	return &Reader{db: db}
}

func (f *Factory) DataWriter(db *sql.DB) core.DataWriter {
	return &Writer{db: db}
}

func (f *Factory) TableManager(db *sql.DB) core.TableManager {
	return &Manager{db: db}
}
