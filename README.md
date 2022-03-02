# sqlz

[![GoDev](https://pkg.go.dev/badge/golang.org/x/pkgsite.svg)](https://pkg.go.dev/github.com/go-tk/sqlz)
[![Workflow Status](https://github.com/go-tk/sqlz/actions/workflows/ci.yaml/badge.svg?branch=main)](https://github.com/go-tk/sqlz/actions/workflows/ci.yaml?query=branch%3Amain)
[![Coverage Status](https://codecov.io/gh/go-tk/sqlz/branch/main/graph/badge.svg)](https://codecov.io/gh/go-tk/sqlz/branch/main)

`sqlz` is an extremely simple alternative to [`sqlx`](https://github.com/jmoiron/sqlx),
convenient helper for working with `database/sql`.

## Motivation

- I'm not a fan of ORM and code generation is not a choice to me.
- I want to write maintainable and less bug prone SQL by hand.
- Can't find a library for the same purpose that seems simple enough to me.

## Usage

### Stmt

The common use cases of `Stmt` are as follows:

```go
package main

import (
        "context"
        "database/sql"
        "fmt"

        _ "github.com/go-sql-driver/mysql"
        "github.com/go-tk/sqlz"
)

const createTable = `
DROP TABLE IF EXISTS person;
CREATE TABLE person (
    first_name VARCHAR(255) NOT NULL,
    last_name VARCHAR(255) NOT NULL,
    age INT NOT NULL
);
`

type Person struct {
        FirstName string
        LastName  string
        Age       int
}

func main() {
        db, err := sql.Open("mysql", "root:password@tcp(127.0.0.1:3306)/test?multiStatements=true")
        if err != nil {
                panic(err)
        }
        _, err = db.Exec(createTable)
        if err != nil {
                panic(err)
        }

        // 1. Insert Persons in bulk
        {
                stmt := sqlz.NewStmt("INSERT INTO person ( first_name, last_name, age ) VALUES")
                for _, person := range []Person{
                        {"Jason", "Moiron", 12},
                        {"John", "Doe", 9},
                        {"Peter", "Pan", 13},
                } {
                        stmt.Append("( ?, ?, ? ),").Format(person.FirstName, person.LastName, person.Age)
                }
                stmt.Trim(",") // Remove the trailing ','

                if _, err := stmt.Exec(context.Background(), db); err != nil {
                        panic(err)
                }
        }

        // Define a helper function
        selectPerson := func(person *Person) *sqlz.Stmt {
                return sqlz.NewStmt("SELECT").
                        Append("first_name,").Scan(&person.FirstName).
                        Append("last_name,").Scan(&person.LastName).
                        Append("age,").Scan(&person.Age).
                        Trim(","). // Remove the trailing ','
                        Append("FROM person")
        }

        // 2. Get a single Person
        {
                var person Person
                stmt := selectPerson(&person).
                        Append("WHERE").
                        Append("age BETWEEN ? AND ?").Format(12, 13).
                        Append("AND").
                        Append("last_name = ?").Format("Pan").
                        Append("LIMIT 1")

                if err := stmt.QueryRow(context.Background(), db); err != nil {
                        panic(err)
                }
                fmt.Printf("%v\n", person)
                // Output: {Peter Pan 13}
        }

        // 3. Get all Persons
        {
                var temp Person
                stmt := selectPerson(&temp).Append("LIMIT 100")

                var persons []Person
                if err := stmt.Query(context.Background(), db, func() bool {
                        // Be called back for each row
                        persons = append(persons, temp)
                        return true
                }); err != nil {
                        panic(err)
                }
                fmt.Printf("%v\n", persons)
                // Output: [{Jason Moiron 12} {John Doe 9} {Peter Pan 13}]
        }
}
```

### BeginTx

See the comment in the code below:

```go
func DoSomething(ctx context.Context, db *sql.DB) (returnedErr error) {
        tx, closeTx, err := sqlz.BeginTx(ctx, db, nil, &returnedErr)
        if err != nil {
                return err
        }
        defer closeTx() // Automatically call tx.Commit() or tx.Rollback() according to returnedErr
        ...
        if err != nil {
                return err
        }
        ...
        return nil
}
```
