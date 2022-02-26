package sqlz_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	. "github.com/go-tk/sqlz"
	"github.com/stretchr/testify/assert"
)

func Test_BeginTx(t *testing.T) {
	f := func(db *sql.DB) (returnedErr error) {
		tx, closeTx, err := BeginTx(context.Background(), db, nil, &returnedErr)
		if err != nil {
			return err
		}
		defer closeTx()
		var n int
		if err := tx.QueryRow("select x from foo").Scan(&n); err != nil {
			return fmt.Errorf("scan row: %w", err)
		}
		return nil
	}
	t.Run("1", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("select x from foo").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		mock.ExpectCommit()
		f(db)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("2", func(t *testing.T) {
		db, mock := newMockDB(t)
		myErr := errors.New("failed")
		mock.ExpectBegin().WillReturnError(myErr)
		err := f(db)
		assert.ErrorIs(t, err, myErr)
		assert.EqualError(t, err, "begin tx: failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("3", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		myErr := errors.New("failed")
		mock.ExpectQuery("select x from foo").WillReturnError(myErr)
		mock.ExpectRollback()
		err := f(db)
		assert.ErrorIs(t, err, myErr)
		assert.EqualError(t, err, "scan row: failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
	t.Run("4", func(t *testing.T) {
		db, mock := newMockDB(t)
		mock.ExpectBegin()
		mock.ExpectQuery("select x from foo").WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))
		myErr := errors.New("failed")
		mock.ExpectCommit().WillReturnError(myErr)
		err := f(db)
		assert.ErrorIs(t, err, myErr)
		assert.EqualError(t, err, "commit tx: failed")
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
