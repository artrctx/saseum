package pg

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"saseum/internal/db/util"
	"saseum/internal/embed"
	"strings"

	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

const BatchSize int = 100

type EmbeddingTable struct {
	// src table name
	srcName string
	name    string
	db      *sqlx.DB
}

func (et *EmbeddingTable) Name() string {
	return et.name
}

func (et *EmbeddingTable) DeleteWithTx(tx *sql.Tx) error {
	if _, err := tx.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %s;", et.name)); err != nil {
		return err
	}
	return nil
}

// syncs embedding between src
func (et *EmbeddingTable) Sync(ctx context.Context, emb *embed.Embedder) (count int, err error) {
	etMap, err := et.GetSourceTableMap()
	if err != nil {
		return 0, err
	}
	if len(etMap) == 0 {
		return 0, fmt.Errorf("embedding table(%s) and src table(%s) doesn't have any foreign relation", et.name, et.srcName)
	}

	if err := et.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s;", et.srcName)).Scan(&count); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}

	eg, ctx := errgroup.WithContext(ctx)
	for idx := range int(math.Ceil(float64(count) / float64(BatchSize))) {
		eg.Go(func() error {
			return et.SyncOffset(ctx, emb, etMap[0].PrimaryColumn, BatchSize, idx*BatchSize)
		})
	}

	if err := eg.Wait(); err != nil {
		return 0, err
	}

	return
}

func (et *EmbeddingTable) SyncOffset(ctx context.Context, emb *embed.Embedder, orderKey string, limit, offset int) error {
	rows, err := et.db.Queryx(fmt.Sprintf("SELECT * FROM %s ORDER BY %s ASC LIMIT $1 OFFSET $2;", et.srcName, orderKey), limit, offset)
	if err != nil {
		return err
	}
	defer rows.Close()
	entries := []map[string]any{}
	for rows.Next() {
		entry := make(map[string]any)
		if err := rows.MapScan(entry); err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	fmt.Println(len(entries), "TEST")

	entryLen := len(entries)
	entryStrs := make([]string, entryLen)
	for idx := range len(entryStrs) {
		entryStrs[idx] = util.MapToReadableStr(entries[idx])
	}

	embs, err := emb.Generate(strings.Join(entryStrs, "\n\n"))
	if err != nil {
		return err
	}

	if len(embs) != entryLen {
		return fmt.Errorf("Embedding returned wrong size. expected %d entries go %d embedding", entryLen, len(embs))
	}
	//TODO insert into table
	//TODO embedding takes long there should be ways to send progress msg

	return nil
}

type EmbeddingTableMap struct {
	ForeignKeyName string `db:"foreign_key_name"`
	// embedding table
	SrcTable  string `db:"source_table"`
	SrcColumn string `db:"source_column"`
	// source table
	PrimaryTable  string `db:"primary_table"`
	PrimaryColumn string `db:"primary_column"`
}

func (et *EmbeddingTable) GetSourceTableMap() ([]EmbeddingTableMap, error) {
	rows, err := et.db.Queryx(`SELECT 
    rc.constraint_name AS foreign_key_name,
    kcu1.table_name AS source_table,
    kcu1.column_name AS source_column,
    kcu2.table_name AS primary_table,
    kcu2.column_name AS primary_column
FROM information_schema.referential_constraints rc
JOIN information_schema.key_column_usage kcu1 ON rc.constraint_name = kcu1.constraint_name
JOIN information_schema.key_column_usage kcu2 ON rc.unique_constraint_name = kcu2.constraint_name
    AND kcu1.ordinal_position = kcu2.ordinal_position
WHERE kcu1.table_name = $1;`, et.name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	embeddingTableMap := []EmbeddingTableMap{}
	for rows.Next() {
		var eMap EmbeddingTableMap
		if err := rows.StructScan(&eMap); err != nil {
			return nil, err
		}

		if eMap.PrimaryTable != et.srcName {
			continue
		}

		embeddingTableMap = append(embeddingTableMap, eMap)
	}
	return embeddingTableMap, nil
}
