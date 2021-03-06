package sqltest

import (
	"context"
	"fmt"
	"testing"
	"time"
)

const (
	truncateStmtFmt = "TRUNCATE TABLE %s"
	showTablesStmt  = "SHOW TABLES"
)

var (
	// TruncateTimeout is the maximum time duration for truncation calls. (Default: 5 seconds)
	TruncateTimeout = 5 * time.Second
)

// GenericTruncator represents a common set of methods for truncating a database during testing.
type GenericTruncator struct {
	agent Agent
}

// MustTruncateAll will run TruncateAll and fail test if it's unsuccessful.
func (tr *GenericTruncator) MustTruncateAll(t testing.TB) {
	mustTruncateAll(t, tr)
}

// TruncateAll will empty all tables in the database.
func (tr *GenericTruncator) TruncateAll(t testing.TB) error {
	return truncateAll(t, tr, tr.agent)
}

// MustTruncateTables will run TruncateTables and will fail test if it can't.
func (tr *GenericTruncator) MustTruncateTables(t testing.TB, tables ...string) {
	mustTruncateTables(t, tr, tables...)
}

// TruncateTables removes all content in the given tables.
func (tr *GenericTruncator) TruncateTables(t testing.TB, tables ...string) error {
	return truncateTables(t, tr, tr.agent, tables...)
}

func mustTruncateAll(t testing.TB, tr Truncator) {
	if err := tr.TruncateAll(t); err != nil {
		t.Error(err)
	}
}

func truncateAll(t testing.TB, tr Truncator, agent Agent) error {
	ctx, cancel := context.WithTimeout(context.Background(), TruncateTimeout)
	defer cancel()

	rows, err := agent.QueryContext(ctx, showTablesStmt)
	if err != nil {
		return err
	}
	var tables []string
	for rows.Next() {
		var table string
		rows.Scan(&table)
		switch table {
		case "schema_migrations", "schema_lock":
		default:
			tables = append(tables, table)
		}
	}
	return tr.TruncateTables(t, tables...)
}

func mustTruncateTables(t testing.TB, tr Truncator, tables ...string) {
	if err := tr.TruncateTables(t, tables...); err != nil {
		t.Error(err)
	}
}

func truncateTables(t testing.TB, tr Truncator, agent Agent, tables ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), TruncateTimeout)
	defer cancel()
	ch := make(chan error)
	for _, table := range tables {
		go func(table string) {
			_, err := agent.ExecContext(ctx, fmt.Sprintf(truncateStmtFmt, table))
			for {
				select {
				case <-ctx.Done():
					return // returning not to leak the goroutine
				case ch <- err:
					return
				}
			}
		}(table)
	}
	// This will read the channel once for each goroutine launched, thereby
	// blocking until all goroutines have finished.
	for range tables {
		if err := <-ch; err != nil {
			// The deferred cancel() will kill any blocked goroutines.
			return err
		}
	}
	return nil
}
