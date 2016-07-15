package pan

import (
	"fmt"
	"reflect"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	TAG_NAME = "sql_column" // The tag that will be read
)

var (
	structReadCache = map[string][]string{}
	structReadMutex sync.RWMutex
)

func validTag(s string) bool {
	if s == "" {
		return false
	}
	for _, c := range s {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != rune([]byte("_")[0]) && c != rune([]byte(".")[0]) && c != rune([]byte("-")[0]) {
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
			prevWasLower = false
		}

		n := utf8.EncodeRune(buf, c)
		snake += string(buf[0:n])
	}
	return snake
}

func getFieldColumn(f reflect.StructField) string {
	// Get the SQL column name, from the tag or infer it
	field := f.Tag.Get(TAG_NAME)
	if field == "-" {
		return ""
	}
	if field == "" || !validTag(field) {
		field = toSnake(f.Name)
	}
	return field
}

// if needsValues is false, we'll attempt to use the cache and won't return any values
func readStruct(s SQLTableNamer, needsValues bool) (columns []string, values []interface{}) {
	typ := fmt.Sprintf("%T", s)
	structReadMutex.RLock()
	if cached, ok := structReadCache[typ]; !needsValues && ok {
		structReadMutex.RUnlock()
		return cached, nil
	}
	structReadMutex.RUnlock()
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)
	k := t.Kind()
	for k == reflect.Interface || k == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
		k = t.Kind()
	}
	if k != reflect.Struct {
		return
	}
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).PkgPath != "" {
			// skip unexported fields
			continue
		}
		field := getFieldColumn(t.Field(i))
		if field == "" {
			continue
		}
		field = s.GetSQLTableName() + "." + field
		columns = append(columns, field)

		if needsValues {
			// Get the value of the field
			value := v.Field(i).Interface()
			values = append(values, value)
		}
	}

	structReadMutex.Lock()
	structReadCache[typ] = columns
	structReadMutex.Unlock()
	return
}

func Columns(s SQLTableNamer) ColumnList {
	columns, _ := readStruct(s, false)
	return columns
}

// GetColumn returns the field name associated with the specified property in the passed value.
// Property must correspond exactly to the name of the property in the type, or this function will
// panic.
func Column(s SQLTableNamer, property string) string {
	t := reflect.TypeOf(s)
	k := t.Kind()
	for k == reflect.Interface || k == reflect.Ptr {
		t = reflect.ValueOf(s).Elem().Type()
		k = t.Kind()
	}
	if k != reflect.Struct {
		return ""
	}
	field, ok := t.FieldByName(property)
	if !ok {
		panic("Field not found in type: " + property)
	}
	column := s.GetSQLTableName() + "." + getFieldColumn(field)
	return column
}

func ColumnValues(s SQLTableNamer) []interface{} {
	_, values := readStruct(s, true)
	return values
}

type SQLTableNamer interface {
	GetSQLTableName() string
}

// GetTableName returns the table name for any type that implements the `GetSQLTableName() string`
// method signature. The returned string will be used as the name of the table to store the data
// for all instances of the type.
func Table(t SQLTableNamer) string {
	return t.GetSQLTableName()
}

// VariableList returns a list of `num` variable placeholders for use in SQL queries involving slices
// and arrays.
func VariableList(num int) string {
	placeholders := make([]string, num)
	for pos := 0; pos < num; pos++ {
		placeholders[pos] = "?"
	}
	return strings.Join(placeholders, ",")
}

// QueryList joins the passed fields into a string that can be used when selecting the fields to return
// or specifying fields in an update or insert.
func QueryList(fields []interface{}) string {
	strs := make([]string, len(fields))
	for pos, field := range fields {
		strs[pos] = field.(string)
	}
	return strings.Join(strs, ", ")
}

type Scannable interface {
	Scan(dest ...interface{}) error
}

func Unmarshal(s Scannable, dst interface{}) error {
	t := reflect.TypeOf(dst)
	v := reflect.ValueOf(dst)
	k := t.Kind()
	for k == reflect.Interface || k == reflect.Ptr {
		v = v.Elem()
		t = v.Type()
		k = t.Kind()
	}
	if k != reflect.Struct {
		return s.Scan(dst)
	}
	pointers := []interface{}{}
	for i := 0; i < t.NumField(); i++ {
		if t.Field(i).PkgPath != "" {
			// skip unexported fields
			continue
		}
		field := getFieldColumn(t.Field(i))
		if field == "" {
			continue
		}

		// Get the value of the field
		pointers = append(pointers, v.Field(i).Addr().Interface())
	}
	return s.Scan(pointers...)
}
