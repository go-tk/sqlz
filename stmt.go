package sqlz

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
)

type Stmt struct {
	sql      bytes.Buffer
	lastChar byte
	values   []interface{}
	args     []interface{}
}

func NewStmt(sql string) *Stmt {
	var s Stmt
	s.sql.WriteString(sql)
	s.lastChar = sql[len(sql)-1]
	return &s
}

func (s *Stmt) Append(sql string) *Stmt {
	if s.lastChar != ' ' {
		s.sql.WriteByte(' ')
	}
	s.sql.WriteString(sql)
	s.lastChar = sql[len(sql)-1]
	return s
}

func (s *Stmt) Trim(sql string) *Stmt {
	if bytes.HasSuffix(s.sql.Bytes(), []byte(sql)) {
		s.sql.Truncate(s.sql.Len() - len(sql))
	}
	return s
}

func (s *Stmt) Scan(values ...interface{}) *Stmt {
	s.values = append(s.values, values...)
	return s
}

func (s *Stmt) Format(args ...interface{}) *Stmt {
	s.args = append(s.args, args...)
	return s
}

type Execer interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (result sql.Result, err error)
}

var _, _ Execer = (*sql.DB)(nil), (*sql.Tx)(nil)

func (s *Stmt) Exec(ctx context.Context, execer Execer) (sql.Result, error) {
	sql := s.sql.String()
	result, err := execer.ExecContext(ctx, sql, s.args...)
	if err != nil {
		return nil, fmt.Errorf("execute statement; sql=%q: %w", sql, err)
	}
	return result, err
}

type Queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) (row *sql.Row)
	QueryContext(ctx context.Context, query string, args ...interface{}) (rows *sql.Rows, err error)
}

var _, _ Queryer = (*sql.DB)(nil), (*sql.Tx)(nil)

func (s *Stmt) QueryRow(ctx context.Context, queryer Queryer) error {
	sql := s.sql.String()
	row := queryer.QueryRowContext(ctx, sql, s.args...)
	if err := row.Scan(s.values...); err != nil {
		return fmt.Errorf("scan row; sql=%q: %w", sql, err)
	}
	return nil
}

func (s *Stmt) Query(ctx context.Context, queryer Queryer, callback func() bool) error {
	sql := s.sql.String()
	rows, err := queryer.QueryContext(ctx, sql, s.args...)
	if err != nil {
		return fmt.Errorf("execute query; sql=%q: %w", sql, err)
	}
	for rows.Next() {
		if err := rows.Scan(s.values...); err != nil {
			return fmt.Errorf("scan row; sql=%q: %w", sql, err)
		}
		if !callback() {
			break
		}
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close rows; sql=%q: %w", sql, err)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate rows; sql=%q: %w", sql, err)
	}
	return nil
}

func (s *Stmt) NValues() int { return len(s.values) }
func (s *Stmt) NArgs() int   { return len(s.args) }
