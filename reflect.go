package pan

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"
	"unicode/utf8"
)

const TAG_NAME = "sql_column" // The tag that will be read

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
		// clear the buffer
		for i := 0; i < n; i++ {
			buf[i] = 0
		}
	}
	return snake
}

func getFieldColumn(f reflect.StructField, quote bool) string {
	// Get the SQL column name, from the tag or infer it
	field := f.Tag.Get(TAG_NAME)
	if field == "-" {
		return ""
	}
	if field == "" || !validTag(field) {
		field = toSnake(f.Name)
	}
	if quote {
		field = "`" + field + "`"
	}
	return field
}

func getFields(s sqlTableNamer, quoted, full bool) (fields []interface{}, values []interface{}) {
	t := reflect.TypeOf(s)
	v := reflect.ValueOf(s)
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
		field := getFieldColumn(t.Field(i), quoted)
		if field == "" {
			continue
		}
		if full {
			tablename := ""
			if quoted {
				tablename = "`"
			}
			tablename = tablename + s.GetSQLTableName()
			if quoted {
				tablename = tablename + "`"
			}
			tablename = tablename + "."
			field = tablename + field
		}

		// Get the value of the field
		value := v.Field(i).Interface()
		fields = append(fields, field)
		values = append(values, value)
	}
	return
}

// GetQuotedFields returns a slice of the fields in the passed type, with their names
// drawn from tags or inferred from the property name (which will be lower-cased with underscores,
// e.g. CamelCase => camel_case) and a corresponding slice of interface{}s containing the values for
// those properties. Fields will be surrounding in ` marks.
func GetQuotedFields(s sqlTableNamer) (fields []interface{}, values []interface{}) {
	return getFields(s, true, false)
}

func GetFields(s sqlTableNamer) (fields []interface{}, values []interface{}) {
	return getFields(s, false, false)
}

// GetAbsoluteFields returns a slice of the fields in the passed type, with their names
// drawn from tags or inferred from the property name (which will be lower-cased with underscores,
// e.g. CamelCase => camel_case) and a corresponding slice of interface{}s containing the values for
// those properties. Fields will be surrounded in \` marks and prefixed with their table name, as
// determined by the passed type's GetSQLTableName. The format will be \`table_name\`.\`field_name\`.
func GetAbsoluteFields(s sqlTableNamer) (fields []interface{}, values []interface{}) {
	return getFields(s, true, true)
}

func GetUnquotedAbsoluteFields(s sqlTableNamer) (fields []interface{}, values []interface{}) {
	return getFields(s, false, true)
}

// GetColumn returns the field name associated with the specified property in the passed value.
// Property must correspond exactly to the name of the property in the type, or this function will
// panic.
func GetColumn(s interface{}, property string) string {
	return getColumn(s, property, true)
}

// GetAbsoluteColumnName returns the field name associated with the specified property in the passed value.
// Property must correspond exactly to the name of the property in the type, or this function will
// panic.
func GetAbsoluteColumnName(s sqlTableNamer, property string) string {
	return fmt.Sprintf("`%s`.%s", GetTableName(s), GetColumn(s, property))
}

func GetUnquotedAbsoluteColumn(s sqlTableNamer, property string) string {
	return fmt.Sprintf("%s.%s", s.GetSQLTableName(), getColumn(s, property, false))
}

func getColumn(s interface{}, property string, quote bool) string {
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
	return getFieldColumn(field, quote)
}

func GetUnquotedColumn(s interface{}, property string) string {
	return getColumn(s, property, false)
}

type sqlTableNamer interface {
	GetSQLTableName() string
}

// GetTableName returns the table name for any type that implements the `GetSQLTableName() string`
// method signature. The returned string will be used as the name of the table to store the data
// for all instances of the type.
func GetTableName(t sqlTableNamer) string {
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

// GetM2MTableName returns a consistent table name for a many-to-many relationship between two tables. No
// matter what order the fields are passed in, the resulting table name will always be consistent.
func GetM2MTableName(t1, t2 sqlTableNamer) string {
	name1 := t1.GetSQLTableName()
	name2 := t2.GetSQLTableName()
	if name2 < name1 {
		name1, name2 = name2, name1
	}
	return fmt.Sprintf("%s_%s", name1, name2)
}

// GetM2MAbsoluteColumnName returns the column name for the supplied field in a many-to-many relationship table,
// including the table name. The field belongs to the first sqlTableNamer, the second sqlTableNamer is the other
// table in the many-to-many relationship.
func GetM2MAbsoluteColumnName(t sqlTableNamer, field string, t2 sqlTableNamer) string {
	return fmt.Sprintf("`%s`.%s", GetM2MTableName(t, t2), GetM2MQuotedColumnName(t, field))
}

// GetM2MColumnName returns the column name for the supplied field in a many-to-many relationship table.
func GetM2MColumnName(t sqlTableNamer, field string) string {
	return fmt.Sprintf("%s_%s", t.GetSQLTableName(), GetUnquotedColumn(t, field))
}

// GetM2MQuotedColumnName returns the column name for the supplied field in a many-to-many relationship table,
// including the quote marks around the column name.
func GetM2MQuotedColumnName(t sqlTableNamer, field string) string {
	return fmt.Sprintf("`%s`", GetM2MColumnName(t, field))
}

// GetM2MFields returns a slice of the columns that should be in a table that maps the many-to-many relationship of
// the types supplied, with their corresponding values. The field parameters specify the primary keys used in
// the relationship table to map to that type.
func GetM2MFields(t1 sqlTableNamer, field1 string, t2 sqlTableNamer, field2 string) (columns, values []interface{}) {
	type1 := reflect.TypeOf(t1)
	type2 := reflect.TypeOf(t2)
	value1 := reflect.ValueOf(t1)
	value2 := reflect.ValueOf(t2)
	kind1 := value1.Kind()
	kind2 := value2.Kind()
	for kind1 == reflect.Interface || kind1 == reflect.Ptr {
		value1 = value1.Elem()
		type1 = value1.Type()
		kind1 = value1.Kind()
	}
	for kind2 == reflect.Interface || kind2 == reflect.Ptr {
		value2 = value2.Elem()
		type2 = value2.Type()
		kind2 = value2.Kind()
	}
	if kind1 != reflect.Struct {
		panic("Can't get fields of " + type1.Name())
	}
	if kind2 != reflect.Struct {
		panic("Can't get fields of " + type2.Name())
	}
	v1 := value1.FieldByName(field1)
	v2 := value2.FieldByName(field2)
	if v1 == *new(reflect.Value) {
		panic(`No "` + field1 + `" field found in ` + type1.Name())
	}
	if v2 == *new(reflect.Value) {
		panic(`No "` + field2 + `" field found in ` + type2.Name())
	}
	column1 := GetM2MColumnName(t1, field1)
	column2 := GetM2MColumnName(t2, field2)
	if column2 < column1 {
		type1, type2 = type2, type1
		value1, value2 = value2, value1
		kind1, kind2 = kind2, kind1
		v1, v2 = v2, v1
		column1, column2 = column2, column1
	}
	columns = append(columns, column1, column2)
	values = append(values, v1.Interface(), v2.Interface())
	return
}

// GetM2MQuotedFields wraps the fields returned by GetM2MFields in quotes.
func GetM2MQuotedFields(t1 sqlTableNamer, field1 string, t2 sqlTableNamer, field2 string) (columns, values []interface{}) {
	columns, values = GetM2MFields(t1, field1, t2, field2)
	for pos, column := range columns {
		columns[pos] = "`" + column.(string) + "`"
	}
	return
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
		field := getFieldColumn(t.Field(i), true)
		if field == "" {
			continue
		}

		// Get the value of the field
		pointers = append(pointers, v.Field(i).Addr().Interface())
	}
	return s.Scan(pointers...)
}
