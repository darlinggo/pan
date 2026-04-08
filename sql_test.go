package pan

import (
	"testing"
	"time"
)

type testPost struct {
	ID       int
	Title    string
	Author   int `sql_column:"author_id"`
	Body     string
	Created  time.Time
	Modified *time.Time
}

func (testPost) GetSQLTableName() string {
	return "test_data"
}

func init() {
	post := testPost{123, "my post", 1, "this is a test post", time.Now(), nil}
	sqlTable[Insert(post)] = queryResult{
		mysql:    "INSERT INTO test_data (id, title, author_id, body, created, modified) VALUES (?, ?, ?, ?, ?, ?);",
		postgres: "INSERT INTO test_data (id, title, author_id, body, created, modified) VALUES ($1, $2, $3, $4, $5, $6);",
	}
	sqlTable[New("UPDATE "+Table(post)+" SET").Assign(Column(post, "Title"), post.Title).Assign(Column(post, "Author"), post.Author).Flush(", ").Where().Comparison(Column(post, "ID"), "=", post.ID).Flush(" ")] = queryResult{
		mysql:    "UPDATE test_data SET title = ?, author_id = ? WHERE id = ?;",
		postgres: "UPDATE test_data SET title = $1, author_id = $2 WHERE id = $3;",
	}
	sqlTable[New("SELECT "+Columns(post).String()+" FROM "+Table(post)).Where().Expression(Column(post, "Created")+" > (SELECT "+Column(post, "Created")+" FROM "+Table(post)+" WHERE "+Column(post, "ID")+" = ?)", 123).Where().OrderByDesc(Column(post, "Created")).Limit(19).Flush(" ")] = queryResult{
		postgres: "SELECT id, title, author_id, body, created, modified FROM test_data WHERE created > (SELECT created FROM test_data WHERE id = $1) ORDER BY created DESC LIMIT $2;",
		mysql:    "SELECT id, title, author_id, body, created, modified FROM test_data WHERE created > (SELECT created FROM test_data WHERE id = ?) ORDER BY created DESC LIMIT ?;",
	}
}

var sqlTable = map[*Query]queryResult{
	New("INSERT INTO "+Table(testPost{})).Expression("("+Placeholders(4)+")", "a", "b", "c", "d").Expression("VALUES").Expression("("+Placeholders(4)+")", 0, 1, 2, 3).Flush(" "): {
		mysql:    "INSERT INTO test_data (?, ?, ?, ?) VALUES (?, ?, ?, ?);",
		postgres: "INSERT INTO test_data ($1, $2, $3, $4) VALUES ($5, $6, $7, $8);",
	},
}

func TestSQLTable(t *testing.T) {
	t.Parallel()
	for query, expectation := range sqlTable {
		t.Log(query.String())
		mysql, err := query.MySQLString()
		if err != nil {
			t.Errorf("Unexpected error: %+v\n", err)
		}
		postgres, err := query.PostgreSQLString()
		if err != nil {
			t.Errorf("Unexpected error: %+v\n", err)
		}
		if mysql != expectation.mysql {
			t.Errorf("Expected '%s' got '%s'", expectation.mysql, mysql)
		}
		if postgres != expectation.postgres {
			t.Errorf("Expected '%s' got '%s'", expectation.postgres, postgres)
		}
	}
}

func BenchmarkInsertGeneration(b *testing.B) {
	post := testPost{123, "my post", 1, "this is a test post", time.Now(), nil}
	for b.Loop() {
		Insert(post)
	}
}

func BenchmarkQueryGeneration(b *testing.B) {
	post := testPost{123, "my post", 1, "this is a test post", time.Now(), nil}
	for b.Loop() {
		New("SELECT "+Columns(post).String()+" FROM "+Table(post)).Where().Comparison(Column(post, "ID"), "=", post.ID).Expression("OR").In(Column(post, "ID"), 123, 456, 789, 101112, 131415).OrderBy(Column(post, "Created")).Limit(10).Flush(" ")
	}
}
