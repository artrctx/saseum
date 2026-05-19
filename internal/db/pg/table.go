package pg

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"saseum/internal/db/util"
	"saseum/internal/embed"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/sync/errgroup"
)

const BatchSize int = 10

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
// TODO: Provide notificiation channel to provide ways to send progress
func (et *EmbeddingTable) Sync(ctx context.Context, emb *embed.Embedder) (count int64, err error) {
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

	startTime := time.Now()
	eg, ctx := errgroup.WithContext(ctx)
	writeCount := atomic.Int64{}
	for idx := range int(math.Ceil(float64(count) / float64(BatchSize))) {
		eg.Go(func() error {
			fmt.Printf("start syncing rows %d-%d\n", idx*BatchSize, idx*BatchSize+BatchSize)
			count, err := et.SyncOffset(ctx, emb, etMap, BatchSize, idx*BatchSize)
			if err == nil {
				fmt.Printf("Synced rows %d-%d\n", idx*BatchSize, idx*BatchSize+BatchSize)
			}
			writeCount.Add(count)
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		return 0, err
	}
	fmt.Printf("Took: %v\n", time.Since(startTime))

	return writeCount.Load(), nil
}

// ASSUMES ALL foreign keys values int, string, or bool
// returns affected row count
func (et *EmbeddingTable) SyncOffset(ctx context.Context, emb *embed.Embedder, tMap []EmbeddingTableMap, limit, offset int) (int64, error) {
	rows, err := et.db.Queryx(fmt.Sprintf("SELECT * FROM %s ORDER BY %s ASC LIMIT $1 OFFSET $2;", et.srcName, tMap[0].PrimaryColumn), limit, offset)
	if err != nil {
		return 0, err
	}
	entryPool := []map[string]any{}
	for rows.Next() {
		entry := make(map[string]any)
		if err := rows.MapScan(entry); err != nil {
			return 0, err
		}
		entryPool = append(entryPool, entry)
	}
	// manually close to relase pg conn pool
	if err := rows.Close(); err != nil {
		return 0, err
	}

	entries := []map[string]any{}
	for _, entry := range entryPool {
		mappedCols := make([]string, len(tMap))
		for idx, m := range tMap {
			calVal, err := util.ToValidDBValue(entry[m.PrimaryColumn])
			if err != nil {
				return 0, err
			}
			mappedCols[idx] = fmt.Sprintf("%s=%v", m.SrcColumn, calVal)
		}

		var rCount int
		if err := et.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s;", et.name, strings.Join(mappedCols, " AND "))).Scan(&rCount); err != nil {
			return 0, err
		}
		// if row already exists skip
		if rCount != 0 {
			continue
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		return 0, nil
	}

	entryLen := len(entries)
	entryStrs := make([]string, entryLen)
	for idx := range len(entryStrs) {
		entryStrs[idx] = util.MapToReadableStr(entries[idx])
	}

	result := <-emb.Queue(strings.Join(entryStrs, "\n\n"))
	if result.Error != nil {
		return 0, result.Error
	}

	embs := result.Data
	if len(embs) != entryLen {
		return 0, fmt.Errorf("Embedding returned wrong size. expected %d entries go %d embedding", entryLen, len(embs))
	}

	insertValues := make([]string, entryLen)
	for idx, e := range entries {
		mappedCols := make([]string, len(tMap)+1)
		for idx, m := range tMap {
			colVal, err := util.ToValidDBValue(e[m.PrimaryColumn])
			if err != nil {
				return 0, err
			}
			mappedCols[idx] = fmt.Sprintf("%v", colVal)
		}
		mappedCols[len(tMap)] = fmt.Sprintf("'%v'", util.ToEmbeddingStr(embs[idx]))
		insertValues[idx] = "(" + strings.Join(mappedCols, ",") + ")"
	}

	fkCols := make([]string, len(tMap)+1)
	for i, m := range tMap {
		fkCols[i] = m.SrcColumn
	}
	fkCols[len(tMap)] = EmbeddingColumnName

	r, err := et.db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(%s) VALUES %s;", et.name, strings.Join(fkCols, ","), strings.Join(insertValues, ",")))
	if err != nil {
		return 0, err
	}

	return r.RowsAffected()
}

func (et *EmbeddingTable) Query(emb *embed.Embedder, text string, limit uint8) ([]map[string]any, error) {
	result := <-emb.Queue(text)
	if result.Error != nil {
		return nil, result.Error
	}

	if len(result.Data) != 1 {
		return nil, fmt.Errorf("resuilting embedding contained more than one dim. if you included two breaklines in input please remove those.")
	}

	tMap, err := et.GetSourceTableMap()
	if err != nil {
		return nil, err
	}

	colStr := ""
	for idx, m := range tMap {
		if idx > 0 {
			colStr += ","
		}
		colStr += m.SrcColumn
	}

	embStr := util.ToEmbeddingStr(result.Data[0])
	rows, err := et.db.Queryx(fmt.Sprintf("SELECT %s FROM %s ORDER BY %s <-> $1 limit $2", colStr, et.name, EmbeddingColumnName), embStr, limit)
	if err != nil {
		return nil, err
	}

	embWheres := make([]string, 0, limit)
	for rows.Next() {
		queryM := make(map[string]any)
		if err := rows.MapScan(queryM); err != nil {
			return nil, err
		}

		var mValue string
		for idx, m := range tMap {
			if idx > 0 {
				mValue += " AND "
			}
			val, err := util.ToValidDBValue(queryM[m.SrcColumn])
			if err != nil {
				return nil, err
			}
			mValue += fmt.Sprintf("%s=%v", m.PrimaryColumn, val)
		}
		embWheres = append(embWheres, "("+mValue+")")
	}

	// close early to free up pool
	if err := rows.Close(); err != nil {
		return nil, err
	}

	results := make([]map[string]any, 0, limit)
	rows, err = et.db.Queryx(fmt.Sprintf("SELECT * FROM %s WHERE %s", et.srcName, strings.Join(embWheres, " OR ")))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		r := make(map[string]any)
		if err := rows.MapScan(r); err != nil {
			return nil, err
		}
		results = append(results, r)
	}

	return results, nil
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
