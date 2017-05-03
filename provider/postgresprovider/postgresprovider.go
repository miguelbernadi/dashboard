package postgresprovider

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	// Use postgresql lib
	_ "github.com/lib/pq"

	"github.com/miguelbernadi/dashboard/provider"
)

// PostgresProvider collects all information needed to set up the DB
// connection
type PostgresProvider struct {
	Host     string
	User     string
	Port     int
	Password string
	DBName   string
	db       *sql.DB
}

// Setup initializes a connection to the DB
func Setup(
	user, password, host, dbname string,
) (
	ret PostgresProvider, err error,
) {
	ret = PostgresProvider{
		Host:     host,
		User:     user,
		Password: password,
		DBName:   dbname,
	}
	log.Println("Logging into", dbname)
	db1, err := sql.Open(
		"postgres",
		//"postgres://pqgotest:password@localhost/pqgotest",
		fmt.Sprintf(
			"postgres://%s:%s@%s/%s",
			user,
			password,
			host,
			dbname,
		),
	)
	if err != nil {
		return
	}
	ret.db = db1
	return
}

// Login logs into the database
func (p PostgresProvider) Login() error {
	return nil
}

// Register returns a QueryList with all query functions we provide
func (p PostgresProvider) Register() (list provider.QueryList, err error) {
	list = make(provider.QueryList)
	list["simpleQueryFunc"] = p.simpleQuery()
	return
}

func (p PostgresProvider) genericQuery(
	ctx context.Context,
	query string,
	args ...interface{},
) (
	value interface{},
	err error,
) {
	var result int
	if p.db == nil {
		log.Fatal("BLAH!")
	}
	rows, err := p.db.QueryContext(ctx, query)
	if err != nil {
		return
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Println(err)
			return
		}
	}()

	if !rows.Next() {
		err = rows.Err()
		log.Println(err)
		return
	}
	err = rows.Scan(&result)
	if err != nil {
		log.Println(err)
		return
	}
	value = result
	return
}

func (p PostgresProvider) simpleQuery() provider.QueryFunction {
	return func(
		ctx context.Context,
		begin, end time.Time,
	) (
		list provider.ResultList,
		err error,
	) {
		query := fmt.Sprintf(
			`SELECT COUNT(*)
	                   FROM articles
		          WHERE updated_at > %s
		            AND updated_at < %s`,
			//'2017-04-29 00:00:00'::TIMESTAMP
			fmt.Sprintf("'%s'::TIMESTAMP", begin.Format("2006-01-02 15:04:05")),
			fmt.Sprintf("'%s'::TIMESTAMP", end.Format("2006-01-02 15:04:05")),
		)
		value, err := p.genericQuery(
			ctx,
			query,
		)
		if err != nil {
			err = fmt.Errorf("postgresprovider.simpleQuery: %s\n", err.Error())
			return
		}
		list = provider.ResultList{"simpleQueryValue": value}
		return
	}
}
