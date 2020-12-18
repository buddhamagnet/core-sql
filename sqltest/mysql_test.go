package sqltest_test

import (
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/buddhamagnet/core-sql/sqltest"
)

func TestMySQLTruncator_TruncateTables(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	tables := []string{"markets", "products", "ingredients"}

	mock.ExpectExec("SET FOREIGN_KEY_CHECKS=?").WithArgs(false).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("TRUNCATE TABLE markets").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("TRUNCATE TABLE products").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("TRUNCATE TABLE ingredients").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("SET FOREIGN_KEY_CHECKS=?").WithArgs(true).WillReturnResult(sqlmock.NewResult(1, 1))

	truncator := sqltest.NewTruncator("mysql", db)
	if err := truncator.TruncateTables(t, tables...); err != nil {
		t.Errorf("the function returned an error: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("there were unfulfilled expectations: %s", err)
	}
}
