package sqlz

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
)

// Stmt represents a SQL statement.
type Stmt struct {
	sqlBuffer bytes.Buffer
	lastChar  byte
	values    []interface{}
	args      []interface{}
}

// NewStmt returns a Stmt with the given SQL fragment.
func NewStmt(sqlFrag string) *Stmt {
	var s Stmt
	s.sqlBuffer.WriteString(sqlFrag)
	s.lastChar = sqlFrag[len(sqlFrag)-1]
	return &s
}

// Append adds the given SQL fragment to the end of the Stmt.
func (s *Stmt) Append(sqlFrag string) *Stmt {
	if s.lastChar != ' ' {
		s.sqlBuffer.WriteByte(' ')
	}
	s.sqlBuffer.WriteString(sqlFrag)
	s.lastChar = sqlFrag[len(sqlFrag)-1]
	return s
}

// Trim removes the given SQL fragment from the end of the Stmt.
func (s *Stmt) Trim(sqlFrag string) *Stmt {
	if bytes.HasSuffix(s.sqlBuffer.Bytes(), []byte(sqlFrag)) {
		s.sqlBuffer.Truncate(s.sqlBuffer.Len() - len(sqlFrag))
	}
	return s
}

// Scan adds values as the output of the Stmt.
func (s *Stmt) Scan(values ...interface{}) *Stmt {
	s.values = append(s.values, values...)
	return s
}

// Format adds arguments as the input of the Stmt.
func (s *Stmt) Format(args ...interface{}) *Stmt {
	s.args = append(s.args, args...)
	return s
}

// Execer is an interface implemented by sql.DB and sql.Tx.
type Execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error)
}

var _, _ Execer = (*sql.DB)(nil), (*sql.Tx)(nil)

// Exec executes the Stmt.
func (s *Stmt) Exec(ctx context.Context, execer Execer) (sql.Result, error) {
	sql := s.SQL()
	result, err := execer.ExecContext(ctx, sql, s.args...)
	if err != nil {
		return nil, fmt.Errorf("execute statement; sql=%q: %w", sql, err)
	}
	return result, err
}

// Queryer is an interface implemented by sql.DB and sql.Tx.
type Queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) (row *sql.Row)
	QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error)
}

var _, _ Queryer = (*sql.DB)(nil), (*sql.Tx)(nil)

// QueryRow executes the Stmt as a query to retrieve a single row.
func (s *Stmt) QueryRow(ctx context.Context, queryer Queryer) error {
	sql := s.SQL()
	row := queryer.QueryRowContext(ctx, sql, s.args...)
	if err := row.Scan(s.values...); err != nil {
		return fmt.Errorf("scan row; sql=%q: %w", sql, err)
	}
	return nil
}

// Query executes the Stmt as a query to retrieve rows.
// The given callback will be called for each row retrieved. If the callback returns false,
// the iteration will be stopped.
func (s *Stmt) Query(ctx context.Context, queryer Queryer, callback func() bool) error {
	sql := s.SQL()
	rows, err := queryer.QueryContext(ctx, sql, s.args...)
	if err != nil {
		return fmt.Errorf("execute query; sql=%q: %w", sql, err)
	}
	for rows.Next() {
		if err := rows.Scan(s.values...); err != nil {
			rows.Close()
			return fmt.Errorf("scan row; sql=%q: %w", sql, err)
		}
		if !callback() {
			if err := rows.Close(); err != nil {
				return fmt.Errorf("close rows; sql=%q: %w", sql, err)
			}
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows; sql=%q: %w", sql, err)
	}
	return nil
}

// SQL returns the underlying SQL to be executed.
func (s *Stmt) SQL() string { return s.sqlBuffer.String() }
