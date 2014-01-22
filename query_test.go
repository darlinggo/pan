package pan

import (
	"fmt"
	"testing"
)

type queryTest struct {
	ExpectedResult string
	Query          *Query
}

var queryTests = []queryTest{
	queryTest{
		ExpectedResult: "This query expects $1 one arg;",
		Query: &Query{
			SQL:    "This query expects ? one arg",
			Args:   []interface{}{0},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "",
		Query: &Query{
			SQL:    "This query expects ? one arg but won't get it;",
			Args:   []interface{}{},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "",
		Query: &Query{
			SQL:    "This query expects no arguments but will get one;",
			Args:   []interface{}{0},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "",
		Query: &Query{
			SQL:    "This query expects ? two args ? but will get one;",
			Args:   []interface{}{0},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "",
		Query: &Query{
			SQL:    "This query expects ? ? two args but will get three;",
			Args:   []interface{}{0, 1, 2},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "Unicode test 世 $1;",
		Query: &Query{
			SQL:    "Unicode test 世 ?",
			Args:   []interface{}{0},
			Engine: POSTGRES,
		},
	},
	queryTest{
		ExpectedResult: "Unicode boundary test $1 " + string(rune(0x80)) + ";",
		Query: &Query{
			SQL:    "Unicode boundary test ? " + string(rune(0x80)),
			Args:   []interface{}{0},
			Engine: POSTGRES,
		},
	},
}

func init() {
	expected := "lots of args"
	SQL := "lots of args"
	args := []interface{}{}
	for i := 1; i < 1001; i++ {
		SQL += " ?"
		expected += fmt.Sprintf(" $%d", i)
		args = append(args, false)
		if i == 10 || i == 100 || i == 1000 {
			queryTests = append(queryTests, queryTest{
				ExpectedResult: expected + ";",
				Query: &Query{
					SQL:    SQL,
					Args:   args,
					Engine: POSTGRES,
				},
			})
		}
	}
}

func TestQueriesFromTable(t *testing.T) {
	for pos, test := range queryTests {
		result := test.Query.String()
		if result != test.ExpectedResult {
			t.Logf("Expected\n%d\ngot\n%d\n.", []byte(test.ExpectedResult), []byte(result))
			t.Errorf("Query test %d failed. Expected \"%s\", got \"%s\".", pos+1, test.ExpectedResult, result)
		}
	}
}

func TestWrongNumberArgsError(t *testing.T) {
	q := New(POSTGRES, "?")
	q.Args = append(q.Args, 1, 2, 3)
	err := q.checkCounts()
	if err == nil {
		t.Errorf("Expected error.")
	}
	if e, ok := err.(WrongNumberArgsError); !ok {
		t.Errorf("Error was not a WrongNumberArgsError.")
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

func TestIncludeIfNotNil(t *testing.T) {
	q := New(POSTGRES, "")
	q.IncludeIfNotNil("hello ?", "world")
	if q.Generate("") != " hello $1;" {
		t.Errorf("Expected `%s`, got `%s`", " hello $1;", q.Generate(""))
	}

	var val *testType
	q = New(POSTGRES, "")
	q.IncludeIfNotNil("hello ?", val)
	if q.Generate("") != ";" {
		t.Errorf("Expected `%s`, got `%s`", ";", q.Generate(""))
	}

	q = New(POSTGRES, "")
	q.IncludeIfNotNil("hello ?", New)
	if q.Generate("") != ";" {
		t.Errorf("Expected `%s`, got `%s`", ";", q.Generate(""))
	}
}

func TestRepeatedOrder(t *testing.T) {
	q := New(POSTGRES, "SELECT * FROM test_data")
	q.IncludeOrder("id DESC")
	q.IncludeOrder("date DESC")
	if q.Generate(" ") != "SELECT * FROM test_data ORDER BY id DESC;" {
		t.Errorf("Expected `%s`, got `%s`", "SELECT * FROM test_data ORDER BY id DESC;", q.Generate(" "))
	}
}

func TestRepeatedLimit(t *testing.T) {
	q := New(POSTGRES, "SELECT * FROM test_data")
	q.IncludeLimit(10)
	q.IncludeLimit(5)
	if q.Generate(" ") != "SELECT * FROM test_data LIMIT $1;" {
		t.Errorf("Expected `%s`, got `%s`", "SELECT * FROM test_data LIMIT $1;", q.Generate(" "))
	}
}

func BenchmarkQueriesFromTable(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		test := queryTests[b.N%len(queryTests)]
		b.StartTimer()
		result := test.Query.String()
		b.StopTimer()
		if result != test.ExpectedResult {
			b.Errorf("Query test %d failed. Expected \"%s\", got \"%s\".", (b.N%len(queryTests))+1, test.ExpectedResult, result)
		}
		b.StartTimer()
	}
}
