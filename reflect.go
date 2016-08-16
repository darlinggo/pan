package pan

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"
)

const (
	tagName = "sql_column" // The tag that will be read
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
	field := f.Tag.Get(tagName)
	if field == "-" {
		return ""
	}
	if field == "" || !validTag(field) {
		field = toSnake(f.Name)
	}
	return field
}

func hasFlags(list []Flag, passed ...Flag) bool {
	for _, candidate := range passed {
		var found bool
		for _, f := range list {
			if f == candidate {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func decorateColumns(columns []string, table string, flags ...Flag) []string {
	results := make([]string, 0, len(columns))
	for _, name := range columns {
		if hasFlags(flags, FlagTicked) {
			name = "`" + name + "`"
		} else if hasFlags(flags, FlagDoubleQuoted) {
			name = `"` + name + `"`
		}
		if hasFlags(flags, FlagFull, FlagTicked) {
			name = "`" + table + "`." + name
		} else if hasFlags(flags, FlagFull, FlagDoubleQuoted) {
			name = `"` + table + `".` + name
		} else if hasFlags(flags, FlagFull) {
			name = table + "." + name
		}
		results = append(results, name)
	}
	return results
}

// if needsValues is false, we'll attempt to use the cache and `values` will be nil
func readStruct(s SQLTableNamer, needsValues bool, flags ...Flag) (columns []string, values []interface{}) {
	typ := fmt.Sprintf("%T", s)
	structReadMutex.RLock()
	if cached, ok := structReadCache[typ]; !needsValues && ok {
		structReadMutex.RUnlock()
		return decorateColumns(cached, s.GetSQLTableName(), flags...), nil
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
	return decorateColumns(columns, s.GetSQLTableName(), flags...), values
}

// Columns returns a ColumnList containing the names of the columns
// in `s`.
func Columns(s SQLTableNamer, flags ...Flag) ColumnList {
	columns, _ := readStruct(s, false, flags...)
	return columns
}

// Column returns the name of the column that `property` maps to for `s`.
// `property` must be the exact name of a property on `s`, or Column will
// panic.
func Column(s SQLTableNamer, property string, flags ...Flag) string {
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
	columns := decorateColumns([]string{getFieldColumn(field)}, s.GetSQLTableName(), flags...)
	return columns[0]
}

// ColumnValues returns the values in `s` for each column in `s`, in the
// same order `Columns` returns the names.
func ColumnValues(s SQLTableNamer) []interface{} {
	_, values := readStruct(s, true)
	return values
}

// SQLTableNamer is used to represent a type that corresponds to an SQL
// table. It must define the GetSQLTableName method, returning the name
// of the SQL table to store data for that type in.
type SQLTableNamer interface {
	GetSQLTableName() string
}

// Table is a convenient shorthand wrapper for the GetSQLTableName method
// on `t`.
func Table(t SQLTableNamer) string {
	return t.GetSQLTableName()
}

// Placeholders returns a formatted string containing `num` placeholders.
// The placeholders will be comma-separated.
func Placeholders(num int) string {
	placeholders := make([]string, num)
	for pos := 0; pos < num; pos++ {
		placeholders[pos] = "?"
	}
	return strings.Join(placeholders, ", ")
}

// Scannable defines a type that can insert the results of a Query into
// the SQLTableNamer a Query was built from, and can list off the column
// names, in order, that those results represent.
type Scannable interface {
	Scan(dst ...interface{}) error
	Columns() ([]string, error)
}

type pointer struct {
	addr      interface{}
	column    string
	sortOrder int
}

type pointers []pointer

func (p pointers) Len() int { return len(p) }

func (p pointers) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p pointers) Less(i, j int) bool { return p[i].sortOrder < p[j].sortOrder }

func getColumnAddrs(s Scannable, in []pointer) ([]interface{}, error) {
	columns, err := s.Columns()
	if err != nil {
		return nil, err
	}
	var results pointers
	for _, pointer := range in {
		for pos, column := range columns {
			if column == pointer.column {
				pointer.sortOrder = pos
				results = append(results, pointer)
				break
			}
		}
	}
	sort.Sort(results)
	i := make([]interface{}, 0, len(results))
	for _, res := range results {
		i = append(i, res.addr)
	}
	return i, nil
}

// Unmarshal reads the Scannable `s` into the variable at `d`, and returns an
// error if it is unable to. If there are more values than `d` has properties
// associated with columns, `additional` can be supplied to catch the extra values.
// The variables in `additional` must be a compatible type with and be in the same
// order as the columns of `s`.
func Unmarshal(s Scannable, dst interface{}, additional ...interface{}) error {
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
	props := []pointer{}
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
		props = append(props, pointer{
			addr:   v.Field(i).Addr().Interface(),
			column: field,
		})
	}

	addrs, err := getColumnAddrs(s, props)
	if err != nil {
		return err
	}
	addrs = append(addrs, additional...)
	return s.Scan(addrs...)
}
