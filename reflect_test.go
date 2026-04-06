package pan

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

type testType struct {
	myInt          int
	MyTaggedInt    int `sql_column:"tagged_int"`
	MyString       string
	myTaggedString string `sql_column:"tagged_string"` //nolint:revive // purposefully tagging unexported field to make sure it's not surfaced
	OmittedColumn  string `sql_column:"-"`
}

func (testType) GetSQLTableName() string {
	return "test_types"
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
		if val, ok := values[pos].(int); column == "test_types.tagged_int" && (!ok || val != 2) {
			t.Errorf("Expected tagged_int to be %d, got %v", 2, values[pos])
		}
		if val, ok := values[pos].(string); column == "test_types.my_string" && (!ok || val != "hello") {
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

func (invalidSQLFieldReflector) GetSQLTableName() string {
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

	var namer SQLTableNamer
	namer = testType{}
	columns = Columns(namer)
	if len(columns) != 2 {
		t.Errorf("Expected %d columns, but got %v", len(columns), columns)
	}
	values = ColumnValues(namer)
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}

	namer = &testType{}
	columns = Columns(namer)
	if len(columns) != 2 {
		t.Errorf("Expected %d columns, but got %v", len(columns), columns)
	}
	values = ColumnValues(namer)
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
	t.Parallel()

	ctx := t.Context()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	t.Cleanup(func() {
		err := os.Remove(dbPath)
		if err != nil {
			t.Log("Error cleaning up after test:", err)
		}
	})

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		closeErr := conn.Close()
		if closeErr != nil {
			t.Log("error closing connection:", closeErr)
		}
	}()

	dummy := testType{
		myInt:          1,
		MyTaggedInt:    12,
		MyString:       "test",
		myTaggedString: "tagged",
		OmittedColumn:  "hide",
	}
	expectation := testType{}
	_, err = conn.ExecContext(ctx, "create table test_types (tagged_int integer, my_string varchar);")
	if err != nil {
		t.Error(err)
	}
	query := Insert(dummy)
	mysql, err := query.SQLiteString()
	if err != nil {
		t.Error(err)
	}
	_, err = conn.ExecContext(ctx, mysql, query.Args()...)
	if err != nil {
		t.Log(query.String())
		t.Error(err)
	}
	selectQuery := New("SELECT " + Columns(dummy).String() + " FROM " + Table(dummy))
	selectString, err := selectQuery.SQLiteString()
	if err != nil {
		t.Log(query.String())
		t.Error(err)
	}
	rows, err := conn.QueryContext(ctx, selectString)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		closeErr := rows.Close()
		if closeErr != nil {
			t.Error(closeErr)
		}
	}()
	for rows.Next() {
		err = Unmarshal(rows, &expectation)
		if err != nil {
			t.Error(err)
		}
	}
	if err := rows.Err(); err != nil {
		t.Error(err)
	}
	if expectation.MyTaggedInt != dummy.MyTaggedInt {
		t.Errorf("Expected MyTaggedInt to be %d, was %d.", dummy.MyTaggedInt, expectation.MyTaggedInt)
	}
	if expectation.MyString != dummy.MyString {
		t.Errorf("Expected MyString to be %s, was %s.", dummy.MyString, expectation.MyString)
	}
}
