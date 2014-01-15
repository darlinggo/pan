package pan

import (
	"testing"
)

type testType struct {
	myInt int
	MyTaggedInt int `sql_column:"tagged_int"`
	MyString string
	myTaggedString string `sql_column:"tagged_string"`
}

func (t testType) GetSQLTableName() string {
	return "test_types"
}

func TestReflectedProperties(t *testing.T) {
	foo := testType{
		myInt: 1,
		MyTaggedInt: 2,
		MyString: "hello",
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
