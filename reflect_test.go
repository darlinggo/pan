package pan

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type testType struct {
	myInt          int
	MyTaggedInt    int `sql_column:"tagged_int"`
	MyString       string
	myTaggedString string `sql_column:"tagged_string"`
	OmittedColumn  string `sql_column:"-"`
}

func (t testType) GetSQLTableName() string {
	return "test_types"
}

type testType2 struct {
	ID string
}

func (t testType2) GetSQLTableName() string {
	return "more_tests"
}

func TestReflectedProperties(t *testing.T) {
	t.Parallel()
	foo := testType{
		myInt:          1,
		MyTaggedInt:    2,
		MyString:       "hello",
		myTaggedString: "world",
	}
	columns := Columns(foo, FlagFull)
	if len(columns) != 2 {
		t.Errorf("Columns should have length %d, has length %d", 2, len(columns))
	}
	values := ColumnValues(foo)
	if len(values) != 2 {
		t.Errorf("Values should have length %d, has length %d", 2, len(values))
	}
	for pos, column := range columns {
		if column != "test_types.tagged_int" && column != "test_types.my_string" {
			t.Errorf("Unknown column found: %v", column)
		}
		if column == "test_types.tagged_int" && values[pos].(int) != 2 {
			t.Errorf("Expected tagged_int to be %d, got %v", 2, values[pos])
		}
		if column == "test_types.my_string" && values[pos].(string) != "hello" {
			t.Errorf("Expected my_string to be %s, got %v", "hello", values[pos])
		}
	}
}

var tags = map[string]bool{
	"":          false,
	"my_data":   true,
	"my_data_☃": false,
	"my,data":   false,
}

func TestValidTag(t *testing.T) {
	t.Parallel()
	for input, validity := range tags {
		if validTag(input) != validity {
			expectedValidity := "valid"
			actualValidity := "valid"
			if !validity {
				actualValidity = "invalid"
			}
			if !validity {
				expectedValidity = "invalid"
			}
			t.Errorf("Expected `%s` to be %s, was %s.", input, expectedValidity, actualValidity)
		}
	}
}

var camelToSnake = map[string]string{
	"":          "",
	"myColumn":  "my_column",
	"MyColumn":  "my_column",
	"Mycolumn":  "mycolumn",
	"My☃Column": "my_column",
}

func TestCamelToSnake(t *testing.T) {
	t.Parallel()
	for input, expectedOutput := range camelToSnake {
		if expectedOutput != toSnake(input) {
			t.Errorf("Expected `%s` to be `%s`, was `%s`", input, expectedOutput, toSnake(input))
		}
	}
}

type invalidSQLFieldReflector string

func (i invalidSQLFieldReflector) GetSQLTableName() string {
	return "invalid_reflection_table"
}

func TestInvalidFieldReflection(t *testing.T) {
	t.Parallel()
	columns := Columns(invalidSQLFieldReflector("test"))
	values := ColumnValues(invalidSQLFieldReflector("test"))
	if len(columns) != 0 {
		t.Errorf("Expected %d columns, got %d.", 0, len(columns))
	}
	if len(values) != 0 {
		t.Errorf("Expected %d values, got %d.", 0, len(values))
	}
}

func TestInterfaceOrPointerFieldReflection(t *testing.T) {
	t.Parallel()
	columns := Columns(&testType{})
	if len(columns) != 2 {
		t.Errorf("Expected %d columns, but got %v", len(columns), columns)
	}
	values := ColumnValues(&testType{})
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}

	var i SQLTableNamer
	i = testType{}
	columns = Columns(i)
	if len(columns) != 2 {
		t.Errorf("Expected %d columns, but got %v", len(columns), columns)
	}
	values = ColumnValues(i)
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}

	i = &testType{}
	columns = Columns(i)
	if len(columns) != 2 {
		t.Errorf("Expected %d columns, but got %v", len(columns), columns)
	}
	values = ColumnValues(i)
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}
}

func TestInvalidColumnTypes(t *testing.T) {
	t.Parallel()
	defer func() {
		t.Log(recover())
	}()
	result := Column(&testType{}, "NotARealProperty")
	t.Errorf("Expected a panic, got `%s` instead.", result)
}

func TestOmittedColumn(t *testing.T) {
	t.Parallel()
	columns := Columns(&testType{})
	for _, column := range columns {
		if column == "omitted_column" {
			t.Errorf("omitted_column should not have shown up, but it did.")
		}
	}
}

func TestUnmarshal(t *testing.T) {
	os.Remove("./test.db")

	db, err := sql.Open("sqlite3", "./test.db")
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	dummy := testType{
		myInt:          1,
		MyTaggedInt:    12,
		MyString:       "test",
		myTaggedString: "tagged",
		OmittedColumn:  "hide",
	}
	expectation := testType{}
	_, err = db.Exec("create table test_types (tagged_int integer, my_string varchar);")
	if err != nil {
		t.Error(err)
	}
	q := Insert(dummy)
	mysql, err := q.SQLiteString()
	if err != nil {
		t.Error(err)
	}
	_, err = db.Exec(mysql, q.Args()...)
	if err != nil {
		t.Log(q.String())
		t.Error(err)
	}
	rows, err := db.Query("SELECT " + Columns(dummy).String() + " FROM test_types;")
	if err != nil {
		t.Error(err)
	}
	for rows.Next() {
		err = Unmarshal(rows, &expectation)
		if err != nil {
			t.Error(err)
		}
	}
	if expectation.MyTaggedInt != dummy.MyTaggedInt {
		t.Errorf("Expected MyTaggedInt to be %d, was %d.", dummy.MyTaggedInt, expectation.MyTaggedInt)
	}
	if expectation.MyString != dummy.MyString {
		t.Errorf("Expected MyString to be %s, was %s.", dummy.MyString, expectation.MyString)
	}
	os.Remove("./test.db")
}
