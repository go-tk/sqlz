package sqlz_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/go-tk/sqlz"
	"github.com/stretchr/testify/assert"
)

func TestStmt_Append(t *testing.T) {
	sql := NewStmt("insert").Append("into").Append("foo").SQL()
	assert.Equal(t, "insert into foo", sql)
}

func TestStmt_Trim(t *testing.T) {
	sql := NewStmt("select").Append("a,").Append("b,").Append("c,").Trim(",").SQL()
	assert.Equal(t, "select a, b, c", sql)
}

func TestStmt_Exec(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectExec("insert into foo \\( a, b, c \\) values \\( \\?, \\?, \\? \\)").
		WithArgs(1, 2, 3).
		WillReturnResult(sqlmock.NewResult(99, 100))
	result, err := NewStmt("insert into foo ( a, b, c ) values (").
		Append("?,").Format(1).
		Append("?,").Format(2).
		Append("?,").Format(3).
		Trim(",").
		Append(")").
		Exec(context.Background(), db)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	lii, _ := result.LastInsertId()
	ra, _ := result.RowsAffected()
	assert.Equal(t, lii, int64(99))
	assert.Equal(t, ra, int64(100))
}

func TestStmt_QueryRow(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("select a, b, c from foo where a = \\? and b = \\?").
		RowsWillBeClosed().
		WithArgs(1, 2).
		WillReturnRows(
			sqlmock.NewRows([]string{"a", "b", "c"}).
				AddRow(1, 2, 3))
	var a, b, c int
	err := NewStmt("select").
		Append("a,").Scan(&a).
		Append("b,").Scan(&b).
		Append("c,").Scan(&c).
		Trim(",").
		Append("from foo where").
		Append("a = ?").Format(1).
		Append("and").
		Append("b = ?").Format(2).
		QueryRow(context.Background(), db)
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.Equal(t, a, 1)
	assert.Equal(t, b, 2)
	assert.Equal(t, c, 3)
}

func TestStmt_Query(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("select a, b, c from foo").
		RowsWillBeClosed().
		WillReturnRows(
			sqlmock.NewRows([]string{"a", "b", "c"}).
				AddRow(1, 2, 3).
				AddRow(4, nil, 6).
				AddRow(7, 8, 9).
				AddRow(10, 11, 12))
	type Foo struct {
		A int
		B *int
		C int
	}
	var temp Foo
	var foos []Foo
	err := NewStmt("select").
		Append("a,").Scan(&temp.A).
		Append("b,").Scan(&temp.B).
		Append("c,").Scan(&temp.C).
		Trim(",").
		Append("from foo").
		Query(context.Background(), db, func() bool {
			if len(foos) == 3 {
				return false
			}
			foos = append(foos, temp)
			return true
		})
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	intp := func(i int) *int { return &i }
	assert.Equal(t, []Foo{
		{1, intp(2), 3},
		{4, nil, 6},
		{7, intp(8), 9},
	}, foos)
}

func TestStmt_Exec_Failed(t *testing.T) {
	db, mock := newMockDB(t)
	myErr := errors.New("failed")
	mock.ExpectExec("insert into foo \\( a, b, c \\) values \\( 1, 2, 3 \\)").
		WillReturnError(myErr)
	_, err := NewStmt("insert into foo ( a, b, c ) values ( 1, 2, 3 )").
		Exec(context.Background(), db)
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.ErrorIs(t, err, myErr)
	assert.EqualError(t, err, "execute statement; sql=\"insert into foo ( a, b, c ) values ( 1, 2, 3 )\": failed")
}

func TestStmt_QueryRow_Failed(t *testing.T) {
	db, mock := newMockDB(t)
	myErr := errors.New("failed")
	mock.ExpectQuery("select a, b, c from foo limit 1").
		WillReturnError(myErr)
	err := NewStmt("select a, b, c from foo limit 1").
		QueryRow(context.Background(), db)
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.ErrorIs(t, err, myErr)
	assert.EqualError(t, err, "scan row; sql=\"select a, b, c from foo limit 1\": failed")
}

func TestStmt_Query_Failed1(t *testing.T) {
	db, mock := newMockDB(t)
	myErr := errors.New("failed")
	mock.ExpectQuery("select a, b, c from foo").
		WillReturnError(myErr)
	err := NewStmt("select a, b, c from foo").
		Query(context.Background(), db, func() bool {
			return true
		})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.ErrorIs(t, err, myErr)
	assert.EqualError(t, err, "execute query; sql=\"select a, b, c from foo\": failed")
}

func TestStmt_Query_Failed2(t *testing.T) {
	db, mock := newMockDB(t)
	mock.ExpectQuery("select a, b, c from foo").
		RowsWillBeClosed().
		WillReturnRows(
			sqlmock.NewRows([]string{"a", "b", "c"}).
				AddRow(1, "hello", 2))
	var a, b, c int
	err := NewStmt("select a, b, c from foo").Scan(&a, &b, &c).
		Query(context.Background(), db, func() bool {
			return true
		})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.EqualError(t, err, "scan row; sql=\"select a, b, c from foo\": sql: Scan error on column index 1, name \"b\": converting driver.Value type string (\"hello\") to a int: invalid syntax")
}

func TestStmt_Query_Failed3(t *testing.T) {
	db, mock := newMockDB(t)
	myErr := errors.New("failed")
	mock.ExpectQuery("select a, b, c from foo").
		RowsWillBeClosed().
		WillReturnRows(
			sqlmock.NewRows([]string{"a", "b", "c"}).
				AddRow(1, 2, 3).
				AddRow(4, 5, 6).
				CloseError(myErr))
	var a, b, c int
	err := NewStmt("select a, b, c from foo").Scan(&a, &b, &c).
		Query(context.Background(), db, func() bool {
			return false
		})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.ErrorIs(t, err, myErr)
	assert.EqualError(t, err, "close rows; sql=\"select a, b, c from foo\": failed")
}

func TestStmt_Query_Failed4(t *testing.T) {
	db, mock := newMockDB(t)
	myErr := errors.New("failed")
	mock.ExpectQuery("select a, b, c from foo").
		RowsWillBeClosed().
		WillReturnRows(
			sqlmock.NewRows([]string{"a", "b", "c"}).
				AddRow(1, 2, 3).
				RowError(0, myErr))
	var a, b, c int
	err := NewStmt("select a, b, c from foo").Scan(&a, &b, &c).
		Query(context.Background(), db, func() bool {
			return false
		})
	if !assert.Error(t, err) {
		t.FailNow()
	}
	if !assert.NoError(t, mock.ExpectationsWereMet()) {
		t.FailNow()
	}
	assert.ErrorIs(t, err, myErr)
	assert.EqualError(t, err, "iterate rows; sql=\"select a, b, c from foo\": failed")
}

func newMockDB(t *testing.T) (*sql.DB, sqlmock.Sqlmock) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, mock
}
