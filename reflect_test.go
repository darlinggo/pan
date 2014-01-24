package pan

import (
	"testing"
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
