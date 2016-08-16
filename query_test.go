package pan

import (
	"fmt"
	"testing"
)

type queryTest struct {
	ExpectedResult queryResult
	Query          *Query
}

type queryResult struct {
	postgres string
	mysql    string
	err      error
}

var queryTests = []queryTest{
	queryTest{
		ExpectedResult: queryResult{
			postgres: "This query expects $1 one arg;",
			mysql:    "This query expects ? one arg;",
			err:      nil,
		},
		Query: &Query{
			sql:  "This query expects ? one arg",
			args: []interface{}{0},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "",
			mysql:    "",
			err: ErrWrongNumberArgs{
				NumExpected: 1,
				NumFound:    0,
			},
		},
		Query: &Query{
			sql:  "This query expects ? one arg but won't get it;",
			args: []interface{}{},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "",
			mysql:    "",
			err: ErrWrongNumberArgs{
				NumExpected: 0,
				NumFound:    1,
			},
		},
		Query: &Query{
			sql:  "This query expects no arguments but will get one;",
			args: []interface{}{0},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "",
			mysql:    "",
			err: ErrWrongNumberArgs{
				NumExpected: 2,
				NumFound:    1,
			},
		},
		Query: &Query{
			sql:  "This query expects ? two args ? but will get one;",
			args: []interface{}{0},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "",
			mysql:    "",
			err: ErrWrongNumberArgs{
				NumExpected: 2,
				NumFound:    3,
			},
		},
		Query: &Query{
			sql:  "This query expects ? ? two args but will get three;",
			args: []interface{}{0, 1, 2},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "Unicode test 世 $1;",
			mysql:    "Unicode test 世 ?;",
			err:      nil,
		},
		Query: &Query{
			sql:  "Unicode test 世 ?",
			args: []interface{}{0},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			postgres: "Unicode boundary test $1 " + string(rune(0x80)) + ";",
			mysql:    "Unicode boundary test ? " + string(rune(0x80)) + ";",
			err:      nil,
		},
		Query: &Query{
			sql:  "Unicode boundary test ? " + string(rune(0x80)),
			args: []interface{}{0},
		},
	},
	queryTest{
		ExpectedResult: queryResult{
			err: ErrNeedsFlush,
		},
		Query: &Query{
			sql:         "SELECT * FROM mytable WHERE",
			args:        []interface{}{0},
			expressions: []string{"this = ?"},
		},
	},
}

func init() {
	postgres := "lots of args"
	mysql := "lots of args"
	sql := "lots of args"
	args := []interface{}{}
	for i := 1; i < 1001; i++ {
		sql += " ?"
		mysql += " ?"
		postgres += fmt.Sprintf(" $%d", i)
		args = append(args, false)
		if i == 10 || i == 100 || i == 1000 {
			queryTests = append(queryTests, queryTest{
				ExpectedResult: queryResult{
					mysql:    mysql + ";",
					postgres: postgres + ";",
					err:      nil,
				},
				Query: &Query{
					sql:  sql,
					args: args,
				},
			})
		}
	}
}

func TestQueriesFromTable(t *testing.T) {
	t.Parallel()
	for pos, test := range queryTests {
		t.Logf("Testing: %s", test.Query.sql)
		mysql, mErr := test.Query.MySQLString()
		postgres, pErr := test.Query.PostgreSQLString()
		if (mErr != nil && pErr == nil) || (pErr != nil && mErr == nil) || (mErr != nil && pErr != nil && mErr.Error() != pErr.Error()) {
			t.Errorf("Expected %v and %v to be the same.\n", mErr, pErr)
		}
		if (mErr != nil && test.ExpectedResult.err == nil) || (mErr == nil && test.ExpectedResult.err != nil) || (mErr != nil && test.ExpectedResult.err != nil && mErr.Error() != test.ExpectedResult.err.Error()) {
			t.Errorf("Expected error to be %v, got %v\n", mErr, test.ExpectedResult.err)
		}
		if mysql != test.ExpectedResult.mysql {
			t.Errorf("Query test %d failed. Expected MySQL to be \"%s\", got \"%s\".\n", pos+1, test.ExpectedResult.mysql, mysql)
		}
		if postgres != test.ExpectedResult.postgres {
			t.Errorf("Query test %d failed: Expected PostgreSQL to be \"%s\", got \"%s\".\n", pos+1, test.ExpectedResult.postgres, postgres)
		}
	}
}

func TestErrWrongNumberArgs(t *testing.T) {
	t.Parallel()
	q := New("?")
	q.args = append(q.args, 1, 2, 3)
	err := q.checkCounts()
	if err == nil {
		t.Errorf("Expected error.")
	}
	if e, ok := err.(ErrWrongNumberArgs); !ok {
		t.Errorf("Error was not an ErrWrongNumberArgs.")
	} else {
		if e.NumExpected != 1 {
			t.Errorf("Expected %d expectations, got %d", 1, e.NumExpected)
		}
		if e.NumFound != 3 {
			t.Errorf("Expected %d args found, got %d", 3, e.NumFound)
		}
	}
	if err.Error() != "Expected 1 arguments, got 3." {
		t.Errorf("Error message was expected to be `%s`, was `%s` instead.", "Expected 1 arguments, got 3.", err.Error())
	}
}

func TestRepeatedOrder(t *testing.T) {
	t.Parallel()
	q := New("SELECT * FROM test_data")
	q.OrderBy("id")
	q.OrderBy("name")
	q.OrderByDesc("date")
	res, err := q.Flush(" ").MySQLString()
	if err != nil {
		t.Errorf("Unexpected error: %+v\n", err)
	}
	if res != "SELECT * FROM test_data ORDER BY id , name , date DESC;" {
		t.Errorf("Expected `%s`, got `%s`", "SELECT * FROM test_data ORDER BY id , name , date DESC;", res)
	}
}

func TestOffset(t *testing.T) {
	t.Parallel()
	q := New("SELECT * FROM test_data")
	q.Offset(10).Flush(" ")
	mysql, err := q.MySQLString()
	if err != nil {
		t.Errorf("Unexpected error: %+v\n", err)
	}
	postgres, err := q.PostgreSQLString()
	if err != nil {
		t.Errorf("Unexpected error: %+v\n", err)
	}
	if err != nil {
		t.Errorf("Unexpected error: %+v\n", err)
	}
	if mysql != "SELECT * FROM test_data OFFSET ?;" {
		t.Errorf("Expected `%s`, got `%s`", "SELECT * FROM test_data OFFSET ?;", mysql)
	}
	if postgres != "SELECT * FROM test_data OFFSET $1;" {
		t.Errorf("Expected `%s`, got `%s`", "SELECT * FROM test_data OFFSET $1;", postgres)
	}
}

func BenchmarkMySQLString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := queryTests[b.N%len(queryTests)]
		b.StartTimer()
		test.Query.MySQLString()
	}
}

func BenchmarkPostgreSQLString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := queryTests[b.N%len(queryTests)]
		b.StartTimer()
		test.Query.PostgreSQLString()
	}
}

func BenchmarkQueryString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := queryTests[b.N%len(queryTests)]
		b.StartTimer()
		_ = test.Query.String()
	}
}
