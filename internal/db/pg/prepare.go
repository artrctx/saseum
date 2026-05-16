package pg

import (
	"database/sql"
	"errors"
	"fmt"
	"saseum/internal/db/util"
	"strings"
)

const EmbeddingColumnName string = "embedding"
const EmbeddingTablePostfix string = "emb"

type ColumnInfo struct {
	Name       string `db:"column_name"`
	DataType   string `db:"data_type"`
	IsNullable bool   `db:"is_nullable"`
	IsPK       bool   `db:"is_pk"`
}

// returns created vector table name or error | name will be {taget}_emb
// list as a primary key wont be supported
// embedding col will be indexed with Hierarchical Navigable Small World (HNSW)
// m = 16 (default, opt 24, 32) | maximum number of bidirectional links (connections)
// ef_construction = 64 (default, opt 128, 256) |  size of the dynamic candidate list
// using vector_ip_ops since passed vector will be normalized
// NOTE: SET hnsw.ef_search = 100; -- High accuracy: set to 128 or 256 when querying embedding table
func (c *Client) Prepare(target string, vecDim int, clean bool) (embeddingtable util.EmbeddingTable, err error) {
	existingEmb, err := c.embeddingTableExists(target)
	if err != nil {
		return nil, err
	}
	tx, err := c.db.Begin()
	if err != nil {
		return nil, err
	}
	// switching to named return
	defer func() {
		if txErr := tx.Rollback(); txErr != nil && !errors.Is(txErr, sql.ErrTxDone) {
			err = errors.Join(err, txErr)
		}
	}()

	if existingEmb != nil {
		// if table exists and clean request delete else check existing table embedding dimension and return
		if clean {
			if err = existingEmb.DeleteWithTx(tx); err != nil {
				return nil, fmt.Errorf("Failed to delete existing embedding table %s with err: %w", existingEmb.name, err)
			}
		} else {
			tDim, err := c.embeddingColDimension(existingEmb.name, EmbeddingColumnName)
			if err != nil {
				return nil, err
			}
			if tDim != vecDim {
				return nil, fmt.Errorf("embedding table already exists for %s (%s); existsing table vector col (%s) dimension is %d but requested dim is %d", target, existingEmb.name, EmbeddingColumnName, tDim, vecDim)
			}
			return existingEmb, nil
		}
	}

	primaryCols, err := c.getPrimaryColumnInfos(target)
	if err != nil {
		return nil, err
	}

	pLen := len(primaryCols)
	colSet, colNameSet, pColNameSet := make([]string, pLen), make([]string, pLen), make([]string, pLen)
	for idx, col := range primaryCols {
		colName := target + "_" + col.Name
		colSet[idx] = fmt.Sprintf("%s %s", colName, col.DataType)
		colNameSet[idx] = colName
		pColNameSet[idx] = col.Name
	}

	hswnM, hswnEfConstruction := getHnswValue(vecDim)
	colNamesStr, vecTableName := strings.Join(colNameSet, ","), embeddingTableName(target)
	tQuery := fmt.Sprintf(`
CREATE TABLE %s (
%s vector(%d),%s,
PRIMARY KEY (%s), 
FOREIGN KEY (%s) REFERENCES %s(%s));
CREATE INDEX ON %s USING hnsw (embedding vector_ip_ops) WITH (m = %d, ef_construction = %d);
`, vecTableName, EmbeddingColumnName, vecDim, strings.Join(colSet, ","), colNamesStr, colNamesStr, target, strings.Join(pColNameSet, ","), vecTableName, hswnM, hswnEfConstruction)
	if _, err := tx.Exec(tQuery); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return &EmbeddingTable{src: target, name: vecTableName}, nil
}

// this assumes embedding column name is embedding
func (c *Client) embeddingColDimension(table, embeddingColName string) (int, error) {
	var dimension int
	if err := c.db.QueryRow(`SELECT atttypmod AS dimensions FROM pg_attribute WHERE attname = $1 AND attrelid = $2::regclass;`, embeddingColName, table).Scan(&dimension); err != nil {
		return 0, err
	}
	return dimension, nil
}

// pass source table to inclu
func (c *Client) embeddingTableExists(table string) (*EmbeddingTable, error) {
	vecTableName := embeddingTableName(table)

	var res sql.NullString
	if err := c.db.QueryRow("SELECT to_regclass($1)", vecTableName).Scan(&res); err != nil {
		return nil, err
	}
	if !res.Valid {
		return nil, nil
	}

	return &EmbeddingTable{src: table, name: vecTableName}, nil
}

func (c *Client) currentSchema() (string, error) {
	var schema string
	if err := c.db.QueryRow("SELECT current_schema()").Scan(&schema); err != nil {
		return "", err
	}
	return schema, nil
}

func (c *Client) getColumnInfos(table string) ([]ColumnInfo, error) {
	schema, err := c.currentSchema()
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Queryx(
		`SELECT c.column_name, c.data_type, (c.is_nullable::boolean) as is_nullable,
        (pk.column_name IS NOT NULL) as is_pk
    FROM information_schema.columns c
    LEFT JOIN (
        SELECT ku.table_schema, ku.table_name, ku.column_name
        FROM information_schema.key_column_usage ku
        JOIN information_schema.table_constraints tc 
          ON ku.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'PRIMARY KEY'
    ) pk ON c.table_schema = pk.table_schema 
        AND c.table_name = pk.table_name 
        AND c.column_name = pk.column_name
    WHERE c.table_schema = $1 AND c.table_name = $2
    ORDER BY c.ordinal_position;`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cols []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		err := rows.StructScan(&col)
		if err != nil {
			return nil, err
		}

		cols = append(cols, col)
	}

	if len(cols) == 0 {
		return nil, fmt.Errorf("target table does not exists")
	}
	return cols, nil
}

func (c *Client) getPrimaryColumnInfos(table string) ([]ColumnInfo, error) {
	schema, err := c.currentSchema()
	if err != nil {
		return nil, err
	}
	rows, err := c.db.Queryx(
		`SELECT c.column_name, c.data_type, (c.is_nullable::boolean) as is_nullable,
        (pk.column_name IS NOT NULL) as is_pk
    FROM information_schema.columns c
    LEFT JOIN (
        SELECT ku.table_schema, ku.table_name, ku.column_name
        FROM information_schema.key_column_usage ku 
        JOIN information_schema.table_constraints tc 
          ON ku.constraint_name = tc.constraint_name
        WHERE tc.constraint_type = 'PRIMARY KEY'
    ) pk ON c.table_schema = pk.table_schema 
        AND c.table_name = pk.table_name 
        AND c.column_name = pk.column_name
    WHERE pk.column_name IS NOT NULL AND c.table_schema = $1 AND c.table_name = $2
    ORDER BY c.ordinal_position;`, schema, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var primaryCols []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		err := rows.StructScan(&col)
		if err != nil {
			return nil, err
		}
		primaryCols = append(primaryCols, col)
	}

	if len(primaryCols) == 0 {
		return nil, fmt.Errorf("target table does not exists or does not have primary keys")
	}

	return primaryCols, nil
}

// returns (m, ef_construction)util.EmbeddingTable
func getHnswValue(dim int) (int, int) {
	switch {
	case dim >= 1024:
		return 48, 192
	case dim >= 768:
		return 32, 128
	default:
		return 16, 68
	}
}

func embeddingTableName(table string) string {
	return fmt.Sprintf("%s_%s", table, EmbeddingTablePostfix)
}
