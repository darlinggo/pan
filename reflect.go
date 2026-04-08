package pan

import (
	"fmt"
	"reflect"
	"slices"
	"sort"
	"strings"
	"sync"
	"unicode"
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

func toSnake(input string) string {
	if input == "" {
		return ""
	}
	var snake strings.Builder
	prevWasLower := false
	for _, character := range input {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			continue
		}
		if unicode.IsLower(character) {
			prevWasLower = true
		} else if unicode.IsUpper(character) {
			character = unicode.ToLower(character)
			if prevWasLower {
				snake.WriteString("_")
			}
			prevWasLower = false
		}

		snake.WriteRune(character)
	}
	return snake.String()
}

// ColumnName returns the mapped column name for a field, given the field's
// name and the entire `sql_column:"foo"` tag. The tag may have other keys
// defined as well.
func ColumnName(fieldName, fieldTag string) string {
	columnName := reflect.StructTag(fieldTag).Get(tagName)
	if columnName == "-" {
		return ""
	}
	if columnName == "" || !validTag(columnName) {
		columnName = toSnake(fieldName)
	}
	return columnName
}

func hasFlags(list []Flag, passed ...Flag) bool {
	for _, candidate := range passed {
		if !slices.Contains(list, candidate) {
			return false
		}
	}
	return true
}

func decorateColumns(columns []string, table string, flags ...Flag) []string {
	results := make([]string, 0, len(columns))
	for _, name := range columns {
		results = append(results, DecorateColumn(name, table, flags...))
	}
	return results
}

// DecorateColumn quotes or qualifies the passed column that belongs to the
// passed table according to the passed [Flag]s.
func DecorateColumn(column string, table string, flags ...Flag) string {
	if hasFlags(flags, FlagTicked) {
		column = "`" + column + "`"
	} else if hasFlags(flags, FlagDoubleQuoted) {
		column = `"` + column + `"`
	}
	if hasFlags(flags, FlagFull, FlagTicked) {
		column = "`" + table + "`." + column
	} else if hasFlags(flags, FlagFull, FlagDoubleQuoted) {
		column = `"` + table + `".` + column
	} else if hasFlags(flags, FlagFull) {
		column = table + "." + column
	}
	return column
}

// if needsValues is false, we'll attempt to use the cache and `values` will be nil
func readStruct(namer SQLTableNamer, needsValues bool, flags ...Flag) (columns []string, values []any) {
	typ := fmt.Sprintf("%T", namer)
	structReadMutex.RLock()
	if cached, ok := structReadCache[typ]; !needsValues && ok {
		structReadMutex.RUnlock()
		return decorateColumns(cached, namer.GetSQLTableName(), flags...), nil
	}
	structReadMutex.RUnlock()
	namerValue := reflect.ValueOf(namer)
	namerType := reflect.TypeOf(namer)
	namerKind := namerType.Kind()
	for namerKind == reflect.Interface || namerKind == reflect.Pointer {
		namerValue = namerValue.Elem()
		namerType = namerValue.Type()
		namerKind = namerType.Kind()
	}
	if namerKind != reflect.Struct {
		return nil, nil
	}
	for fieldIndex := range namerType.NumField() {
		if namerType.Field(fieldIndex).PkgPath != "" {
			// skip unexported fields
			continue
		}
		field := ColumnName(namerType.Field(fieldIndex).Name, string(namerType.Field(fieldIndex).Tag))
		if field == "" {
			continue
		}
		columns = append(columns, field)

		if needsValues {
			// Get the value of the field
			value := namerValue.Field(fieldIndex).Interface()
			values = append(values, value)
		}
	}

	structReadMutex.Lock()
	structReadCache[typ] = columns
	structReadMutex.Unlock()
	return decorateColumns(columns, namer.GetSQLTableName(), flags...), values
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
func Column(namer SQLTableNamer, property string, flags ...Flag) string {
	namerType := reflect.TypeOf(namer)
	namerKind := namerType.Kind()
	for namerKind == reflect.Interface || namerKind == reflect.Pointer {
		namerType = reflect.ValueOf(namer).Elem().Type()
		namerKind = namerType.Kind()
	}
	if namerKind != reflect.Struct {
		return ""
	}
	field, ok := namerType.FieldByName(property)
	if !ok {
		panic("Field not found in type: " + property)
	}
	columns := decorateColumns([]string{ColumnName(field.Name, string(field.Tag))}, namer.GetSQLTableName(), flags...)
	return columns[0]
}

// ColumnValues returns the values in `s` for each column in `s`, in the
// same order `Columns` returns the names.
func ColumnValues(s SQLTableNamer) []any {
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
	for pos := range num {
		placeholders[pos] = "?"
	}
	return strings.Join(placeholders, ", ")
}

// Scannable defines a type that can insert the results of a Query into
// the SQLTableNamer a Query was built from, and can list off the column
// names, in order, that those results represent.
type Scannable interface {
	Scan(dst ...any) error
	Columns() ([]string, error)
}

type pointer struct {
	addr      any
	column    string
	sortOrder int
}

type pointers []pointer

func (p pointers) Len() int { return len(p) }

func (p pointers) Swap(i, j int) { p[i], p[j] = p[j], p[i] }

func (p pointers) Less(i, j int) bool { return p[i].sortOrder < p[j].sortOrder }

func getColumnAddrs(scannable Scannable, columnPointers []pointer) ([]any, error) {
	columns, err := scannable.Columns()
	if err != nil {
		return nil, err
	}
	var results pointers
	for _, pointer := range columnPointers {
		for pos, column := range columns {
			if column == pointer.column {
				pointer.sortOrder = pos
				results = append(results, pointer)
				break
			}
		}
	}
	sort.Sort(results)
	i := make([]any, 0, len(results))
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
func Unmarshal(scannable Scannable, dst any, additional ...any) error {
	dstType := reflect.TypeOf(dst)
	dstVal := reflect.ValueOf(dst)
	dstKind := dstType.Kind()
	for dstKind == reflect.Interface || dstKind == reflect.Pointer {
		dstVal = dstVal.Elem()
		dstType = dstVal.Type()
		dstKind = dstType.Kind()
	}
	if dstKind != reflect.Struct {
		return scannable.Scan(dst)
	}
	props := []pointer{}
	for fieldIndex := 0; fieldIndex < dstType.NumField(); fieldIndex++ {
		if dstType.Field(fieldIndex).PkgPath != "" {
			// skip unexported fields
			continue
		}
		field := ColumnName(dstType.Field(fieldIndex).Name, string(dstType.Field(fieldIndex).Tag))
		if field == "" {
			continue
		}

		// Get the value of the field
		props = append(props, pointer{
			addr:   dstVal.Field(fieldIndex).Addr().Interface(),
			column: field,
		})
	}

	addrs, err := getColumnAddrs(scannable, props)
	if err != nil {
		return err
	}
	addrs = append(addrs, additional...)
	return scannable.Scan(addrs...)
}
