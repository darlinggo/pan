package pan

import (
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const TAG_NAME = "sql_column"

func validTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func toSnake(s string) string {
	if s == "" {
		return ""
	}
	snake := ""
	prevWasLower := false
	buf := make([]byte, 4)
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			continue
		}
		if unicode.IsLower(c) {
			prevWasLower = true
		} else if unicode.IsUpper(c) {
			c = unicode.ToLower(c)
			if prevWasLower {
				snake += "_"
			}
		}

		n := utf8.EncodeRune(buf, c)
		snake += string(buf)
		// clear the buffer
		for i := 0; i < n; i++ {
			buf[i] = 0
		}
	}
	return snake
}

func getFieldColumn(f reflect.StructField) string {
	// Get the SQL column name, from the tag or infer it
	field := f.Tag.Get(TAG_NAME)
	if field == "" {
		field = toSnake(f.Name)
	}
	field = "`" + field + "`"
	return field
}

func getFields(s sqlTableNamer, full bool) (fields []string, values []interface{}) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
	k := t.Kind()
	if k != reflect.Interface && k != reflect.Ptr && k != reflect.Struct {
		return
	}
	if k == reflect.Ptr {
		v = v.Elem()
	}
	for i := 0; i < t.NumField(); i++ {
		field := getFieldColumn(t.Field(i))
		if full {
			field = "`" + s.GetSQLTableName() + "`." + field
		}
		// Get the value of the field
		value := v.Field(i).Interface()
		fields = append(fields, field)
		values = append(values, value)
	}
	return
}

func GetQuotedFields(s sqlTableNamer) (fields []string, values []interface{}) {
	return getFields(s, false)
}

func GetAbsoluteFields(s sqlTableNamer) (fields []string, values []interface{}) {
	return getFields(s, true)
}

func GetColumn(s interface{}, property string) string {
	t := reflect.TypeOf(s)
	k := t.Kind()
	if k != reflect.Interface && k != reflect.Ptr && k != reflect.Struct {
		return ""
	}
	field, ok := t.FieldByName(property)
	if !ok {
		panic("Field not found in type: " + property)
	}
	return getFieldColumn(field)
}

type sqlTableNamer interface {
	GetSQLTableName() string
}

func GetTableName(t sqlTableNamer) string {
	return t.GetSQLTableName()
}

func VariableList(num int) string {
	placeholders := make([]string, num)
	for pos := 0; pos < num; pos++ {
		placeholders[pos] = "?"
	}
	return strings.Join(placeholders, ",")
}

func QueryList(fields []string) string {
	return strings.Join(fields, ", ")
}
