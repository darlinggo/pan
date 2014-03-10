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
	foo := testType{
		myInt:          1,
		MyTaggedInt:    2,
		MyString:       "hello",
		myTaggedString: "world",
	}
	fields, values := GetAbsoluteFields(foo)
	if len(fields) != 2 {
		t.Errorf("Fields should have length %d, has length %d", 2, len(fields))
	}
	if len(values) != 2 {
		t.Errorf("Values should have length %d, has length %d", 2, len(values))
	}
	for pos, field := range fields {
		if field != "`test_types`.`tagged_int`" && field != "`test_types`.`my_string`" {
			t.Errorf("Unknown field found: % v'", field)
		}
		if field == "`test_types`.`tagged_int`" && values[pos].(int) != 2 {
			t.Errorf("Expected tagged_int to be %d, got %v", 2, values[pos])
		}
		if field == "`test_types`.`my_string`" && values[pos].(string) != "hello" {
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
	for input, expectedOutput := range camelToSnake {
		if expectedOutput != toSnake(input) {
			t.Errorf("Expected `%s` to be `%s`, was `%s`", input, expectedOutput, toSnake(input))
		}
	}
}

type invalidSqlFieldReflector string

func (i invalidSqlFieldReflector) GetSQLTableName() string {
	return "invalid_reflection_table"
}

func TestInvalidFieldReflection(t *testing.T) {
	fields, values := getFields(invalidSqlFieldReflector("test"), true)
	if len(fields) != 0 {
		t.Errorf("Expected %d fields, got %d.", 0, len(fields))
	}
	if len(values) != 0 {
		t.Errorf("Expected %d values, got %d.", 0, len(values))
	}
}

func TestInterfaceOrPointerFieldReflection(t *testing.T) {
	fields, values := getFields(&testType{}, false)
	if len(fields) != 2 {
		t.Errorf("Expected %d fields, but got %v", len(fields), fields)
	}
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}

	var i sqlTableNamer
	i = testType{}
	fields, values = getFields(i, false)
	if len(fields) != 2 {
		t.Errorf("Expected %d fields, but got %v", len(fields), fields)
	}
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}

	i = &testType{}
	fields, values = getFields(i, false)
	if len(fields) != 2 {
		t.Errorf("Expected %d fields, but got %v", len(fields), fields)
	}
	if len(values) != 2 {
		t.Errorf("Expected %d values, but got %v", len(values), values)
	}
}

func TestInvalidColumnTypes(t *testing.T) {
	result := GetColumn("", "test")
	if result != "" {
		t.Errorf("Expected column name to be `%s`, was `%s`.", "", result)
	}

	defer func() {
		t.Log(recover())
	}()
	result = GetColumn(&testType{}, "NotARealProperty")
	t.Errorf("Expected a panic, got `%s` instead.", result)
}

func TestOmittedColumn(t *testing.T) {
	fields, _ := GetQuotedFields(&testType{})
	for _, field := range fields {
		if field.(string) == "`omitted_column`" {
			t.Errorf("omitted_column should not have shown up, but it did.")
		}
	}
}

func TestGetM2MTableName(t *testing.T) {
	tableName := GetM2MTableName(testType{}, testType2{})
	if tableName != "more_tests_test_types" {
		t.Errorf("Expected `%s`, got `%s`", "more_tests_test_types", tableName)
	}
	tableName2 := GetM2MTableName(testType2{}, testType{})
	if tableName2 != "more_tests_test_types" {
		t.Errorf("Expected `%s`, got `%s`", "more_tests_test_types", tableName2)
	}
	if tableName != tableName2 {
		t.Errorf("`%s` is not equal to `%s`", tableName, tableName2)
	}
}

func TestGetAbsoluteColumnName(t *testing.T) {
	columnName := GetAbsoluteColumnName(testType{}, "MyString")
	if columnName != "`test_types`.`my_string`" {
		t.Errorf("Expected `%s`, got `%s`", "`test_types`.`my_string`", columnName)
	}
}

func TestGetM2MColumnName(t *testing.T) {
	columnName := GetM2MColumnName(testType{}, "MyString")
	if columnName != "test_types_my_string" {
		t.Errorf("Expected `%s`, got `%s`", "test_types_my_string", columnName)
	}
}

func TestGetM2MQuotedColumnName(t *testing.T) {
	columnName := GetM2MQuotedColumnName(testType{}, "MyString")
	if columnName != "`test_types_my_string`" {
		t.Errorf("Expected %s, got %s", "`test_types_my_string`", columnName)
	}
}

func TestGetM2MAbsoluteColumnName(t *testing.T) {
	columnName := GetM2MAbsoluteColumnName(testType{}, "MyString", testType2{})
	if columnName != "`more_tests_test_types`.`test_types_my_string`" {
		t.Errorf("Expected %s, got %s", "`more_tests_test_types`.`test_types_my_string`", columnName)
	}
}

func TestGetM2MFields(t *testing.T) {
	t1 := testType{MyString: "hello"}
	t2 := testType2{ID: "world"}
	columns, values := GetM2MFields(t1, "MyString", t2, "ID")
	columns2, values2 := GetM2MFields(t2, "ID", t1, "MyString")
	columns3, values3 := GetM2MFields(&t1, "MyString", &t2, "ID")
	columns4, values4 := GetM2MFields(sqlTableNamer(t1), "MyString", sqlTableNamer(t2), "ID")
	columns5, values5 := GetM2MFields(sqlTableNamer(&t1), "MyString", sqlTableNamer(&t2), "ID")
	columns6, _ := GetM2MQuotedFields(t1, "MyString", t2, "ID")
	if columns[0].(string) != "more_tests_id" {
		t.Errorf("Expected %s, got %s", "more_tests_id", columns[0])
	}
	if columns[1].(string) != "test_types_my_string" {
		t.Errorf("Expected %s, got %s", "test_types_my_string", columns[1])
	}
	if columns2[0].(string) != "more_tests_id" {
		t.Errorf("Expected %s, got %s", "more_tests_id", columns2[0])
	}
	if columns2[1].(string) != "test_types_my_string" {
		t.Errorf("Expected %s, got %s", "test_types_my_string", columns2[1])
	}
	if values[0].(string) != "world" {
		t.Errorf("Expected %s, got %s", "world", values[0].(string))
	}
	if values[1].(string) != "hello" {
		t.Errorf("Expected %s, got %s", "hello", values[1].(string))
	}
	if values2[0].(string) != "world" {
		t.Errorf("Expected %s, got %s", "world", values2[0].(string))
	}
	if values2[1].(string) != "hello" {
		t.Errorf("Expected %s, got %s", "hello", values2[1].(string))
	}
	if columns3[0].(string) != "more_tests_id" {
		t.Errorf("Expected %s, got %s", "more_tests_id", columns[0])
	}
	if columns3[1].(string) != "test_types_my_string" {
		t.Errorf("Expected %s, got %s", "test_types_my_string", columns[1])
	}
	if columns4[0].(string) != "more_tests_id" {
		t.Errorf("Expected %s, got %s", "more_tests_id", columns2[0])
	}
	if columns4[1].(string) != "test_types_my_string" {
		t.Errorf("Expected %s, got %s", "test_types_my_string", columns2[1])
	}
	if values3[0].(string) != "world" {
		t.Errorf("Expected %s, got %s", "world", values3[0].(string))
	}
	if values3[1].(string) != "hello" {
		t.Errorf("Expected %s, got %s", "hello", values3[1].(string))
	}
	if values4[0].(string) != "world" {
		t.Errorf("Expected %s, got %s", "world", values4[0].(string))
	}
	if values4[1].(string) != "hello" {
		t.Errorf("Expected %s, got %s", "hello", values4[1].(string))
	}
	if columns5[0].(string) != "more_tests_id" {
		t.Errorf("Expected %s, got %s", "more_tests_id", columns5[0])
	}
	if columns5[1].(string) != "test_types_my_string" {
		t.Errorf("Expected %s, got %s", "test_types_my_string", columns5[1])
	}
	if values5[0].(string) != "world" {
		t.Errorf("Expected %s, got %s", "world", values5[0].(string))
	}
	if values5[1].(string) != "hello" {
		t.Errorf("Expected %s, got %s", "hello", values5[1].(string))
	}
	if columns6[0].(string) != "`more_tests_id`" {
		t.Errorf("Expected %s, got %s", "`more_tests_id`", columns6[0].(string))
	}
	if columns6[1].(string) != "`test_types_my_string`" {
		t.Errorf("Expected %s, got %s", "`test_types_my_string`", columns6[1].(string))
	}
}

func TestInvalidM2MFieldTypes1(t *testing.T) {
	defer func() {
		t.Log(recover())
	}()
	fields, values := GetM2MFields(&testType{}, "NotARealProperty", &testType2{}, "ID")
	t.Errorf("Expected a panic, got `%v` and `%v` instead.", fields, values)
}

func TestInvalidM2MFieldTypes2(t *testing.T) {
	defer func() {
		t.Log(recover())
	}()
	fields, values := GetM2MFields(&testType{}, "MyString", &testType2{}, "NotARealProperty")
	t.Errorf("Expected a panic, got `%v` and `%v` instead.", fields, values)
}

func TestNonStructM2MFieldTypes1(t *testing.T) {
	defer func() {
		t.Log(recover())
	}()
	fields, values := GetM2MFields(invalidSqlFieldReflector("test"), "mystring", &testType2{}, "ID")
	t.Errorf("Expected a panic, got `%v` and `%v` instead.", fields, values)
}

func TestNonStructM2MFieldTypes2(t *testing.T) {
	defer func() {
		t.Log(recover())
	}()
	fields, values := GetM2MFields(&testType{}, "MyString", invalidSqlFieldReflector("test"), "ID")
	t.Errorf("Expected a panic, got `%v` and `%v` instead.", fields, values)
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
	fields, values := GetQuotedFields(dummy)
	q := New(MYSQL, "INSERT INTO "+GetTableName(dummy))
	q.Include("(" + QueryList(fields) + ")")
	q.Include("VALUES")
	q.Include("("+VariableList(len(values))+")", values...)
	q.FlushExpressions(" ")
	_, err = db.Exec(q.String(), q.Args...)
	if err != nil {
		t.Error(err)
	}
	row := db.QueryRow("SELECT * FROM test_types;")
	err = Unmarshal(row, &expectation)
	if err != nil {
		t.Error(err)
	}
	if expectation.MyTaggedInt != dummy.MyTaggedInt {
		t.Errorf("Expected MyTaggedInt to be %d, was %d.", dummy.MyTaggedInt, expectation.MyTaggedInt)
	}
	if expectation.MyString != dummy.MyString {
		t.Errorf("Expected MyString to be %s, was %s.", dummy.MyString, expectation.MyString)
	}
}
