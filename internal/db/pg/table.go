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
func (et *EmbeddingTable) Sync(ctx context.Context, emb *embed.Embedder, batchSize int) (count int64, err error) {
	if batchSize <= 0 {
		batchSize = 20
	}
	etMap, err := et.GetSourceTableMap()
	if err != nil {
		return 0, err
	}
	if len(etMap) == 0 {
		return 0, fmt.Errorf("embedding table(%s) and src table(%s) doesn't have any foreign relation", et.name, et.srcName)
	}

	filterWhere := ""
	for idx, m := range etMap {
		if idx > 0 {
			filterWhere += " AND "
		}
		filterWhere += fmt.Sprintf("t1.%s=t2.%s", m.SrcColumn, m.Column)
	}
	filter := fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s t2 WHERE %s)", et.name, filterWhere)

	// only query items not already been processed
	if err := et.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s t1 WHERE %s;", et.srcName, filter)).Scan(&count); err != nil {
		return 0, err
	}
	if count == 0 {
		return 0, nil
	}

	startTime := time.Now()
	eg, ctx := errgroup.WithContext(ctx)
	wCount, iterCount := atomic.Int64{}, int(math.Ceil(float64(count)/float64(batchSize)))
	fmt.Printf("Start Syncing %d %s rows in %d iteration.\n", count, et.srcName, iterCount)
	for idx := range iterCount {
		eg.Go(func() error {
			if err := et.syncOffset(ctx, emb, etMap, filter, batchSize, idx*batchSize); err != nil {
				return err
			}
			wCount.Add(1)
			// \033[2K -> Wipe the text on that row
			// \r       -> Move cursor to the absolute start of that row
			fmt.Printf("\033[2K\rSyncing %s. %d of %d completed", et.srcName, wCount.Load(), iterCount)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return 0, err
	}
	fmt.Printf("Took: %v\n", time.Since(startTime))

	return count, nil
}

// ASSUMES ALL foreign keys values int, string, or bool
// returns affected row count
func (et *EmbeddingTable) syncOffset(ctx context.Context, emb *embed.Embedder, tMap []EmbeddingTableMap, filter string, limit, offset int) error {
	rows, err := et.db.Queryx(fmt.Sprintf("SELECT * FROM %s t1 WHERE %s ORDER BY %s ASC LIMIT $1 OFFSET $2;", et.srcName, filter, tMap[0].SrcColumn), limit, offset)
	if err != nil {
		return err
	}

	entries := []map[string]any{}
	for rows.Next() {
		entry := make(map[string]any)
		if err := rows.MapScan(entry); err != nil {
			return err
		}
		entries = append(entries, entry)
	}
	// manually close to relase pg conn pool
	if err := rows.Close(); err != nil {
		return err
	}

	if len(entries) == 0 {
		return nil
	}

	entryLen := len(entries)
	entryStrs := make([]string, entryLen)
	for idx := range len(entryStrs) {
		entryStrs[idx] = util.MapToReadableStr(entries[idx])
	}

	result := <-emb.Queue(strings.Join(entryStrs, "\n\n"))
	if result.Error != nil {
		return result.Error
	}

	embs := result.Data
	if len(embs) != entryLen {
		return fmt.Errorf("Embedding returned wrong size. expected %d entries go %d embedding", entryLen, len(embs))
	}

	insertValues := make([]string, entryLen)
	for idx, e := range entries {
		mappedCols := make([]string, len(tMap)+1)
		for idx, m := range tMap {
			colVal, err := util.ToValidDBValue(e[m.SrcColumn])
			if err != nil {
				return err
			}
			mappedCols[idx] = fmt.Sprintf("%v", colVal)
		}
		mappedCols[len(tMap)] = fmt.Sprintf("'%v'", util.ToEmbeddingStr(embs[idx]))
		insertValues[idx] = "(" + strings.Join(mappedCols, ",") + ")"
	}

	fkCols := make([]string, len(tMap)+1)
	for i, m := range tMap {
		fkCols[i] = m.Column
	}
	fkCols[len(tMap)] = EmbeddingColumnName

	if _, err := et.db.ExecContext(ctx, fmt.Sprintf("INSERT INTO %s(%s) VALUES %s;", et.name, strings.Join(fkCols, ","), strings.Join(insertValues, ","))); err != nil {
		return err
	}

	return nil
}

func (et *EmbeddingTable) Query(emb *embed.Embedder, text string, limit uint8, threshold float32) ([]map[string]any, error) {
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
		colStr += m.Column
	}

	embStr := util.ToEmbeddingStr(result.Data[0])
	thresholdWhere := ""
	if threshold > 0 {
		thresholdWhere = fmt.Sprintf("WHERE %s <#> '%s' < %f", EmbeddingColumnName, embStr, -1*threshold)
	}

	rows, err := et.db.Queryx(fmt.Sprintf("SELECT %s FROM %s %s ORDER BY %s <#> $1 LIMIT $2;", colStr, et.name, thresholdWhere, EmbeddingColumnName), embStr, limit)
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
			val, err := util.ToValidDBValue(queryM[m.Column])
			if err != nil {
				return nil, err
			}
			mValue += fmt.Sprintf("%s=%v", m.SrcColumn, val)
		}
		embWheres = append(embWheres, "("+mValue+")")
	}

	// close early to free up pool
	if err := rows.Close(); err != nil {
		return nil, err
	}

	if len(embWheres) == 0 {
		return []map[string]any{}, nil
	}

	results := make([]map[string]any, 0, limit)
	rows, err = et.db.Queryx(fmt.Sprintf("SELECT * FROM %s WHERE %s;", et.srcName, strings.Join(embWheres, " OR ")))
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
	Table  string `db:"eb_table"`
	Column string `db:"eb_column"`
	// source table
	SrcTable  string `db:"src_table"`
	SrcColumn string `db:"src_column"`
}

func (et *EmbeddingTable) GetSourceTableMap() ([]EmbeddingTableMap, error) {
	rows, err := et.db.Queryx(`SELECT 
    kcu1.table_name AS eb_table,
    kcu1.column_name AS eb_column,
    kcu2.table_name AS src_table,
    kcu2.column_name AS src_column 
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

		if eMap.SrcTable != et.srcName {
			continue
		}

		embeddingTableMap = append(embeddingTableMap, eMap)
	}
	return embeddingTableMap, nil
}
