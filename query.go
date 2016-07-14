package pan

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

// Query contains the data needed to perform a single SQL query.
type Query struct {
	sql           string
	args          []interface{}
	expressions   []string
	includesWhere bool
	includesOrder bool
}

type ColumnList []string

func (c ColumnList) String() string {
	return strings.Join(c, ", ")
}

// New creates a new Query object. The passed engine is used to format variables. The passed string is used to prefix the query.
func New(query string) *Query {
	return &Query{
		sql:  query,
		args: []interface{}{},
	}
}

func Insert(obj SQLTableNamer, values ...SQLTableNamer) *Query {
	inserts := make([]SQLTableNamer, 0, len(values)+1)
	inserts = append(inserts, obj)
	inserts = append(inserts, values...)
	columns := Columns(obj)
	query := New("INSERT INTO " + Table(obj) + " (" + columns.String() + ") VALUES")

	for _, v := range inserts {
		columnValues := ColumnValues(v)
		query.Expression("("+VariableList(len(columnValues))+")", columnValues...)
	}
	return query.Flush(" ")
}

// ErrWrongNumberArgs is thrown when a Query is evaluated whose Args does not match its Expressions.
type ErrWrongNumberArgs struct {
	NumExpected int
	NumFound    int
}

// Error fulfills the error interface, returning the expected number of arguments and the number supplied.
func (e ErrWrongNumberArgs) Error() string {
	return fmt.Sprintf("Expected %d arguments, got %d.", e.NumExpected, e.NumFound)
}

func (q *Query) checkCounts() error {
	placeholders := strings.Count(q.sql, "?")
	args := len(q.args)
	if placeholders != args {
		return ErrWrongNumberArgs{NumExpected: placeholders, NumFound: args}
	}
	return nil
}

func (q *Query) String() string {
	// TODO(paddy): return the query with values injected
	return ""
}

func (q *Query) MySQLString() (string, error) {
	if err := q.checkCounts(); err != nil {
		return "", err
	}
	return q.sql + ";", nil
}

func (q *Query) PostgreSQLString() (string, error) {
	if err := q.checkCounts(); err != nil {
		return "", err
	}
	var pos, width, outputPos int
	var r rune
	var count = 1
	replacementString := "?"
	replacementRune, _ := utf8.DecodeRune([]byte(replacementString))
	terminatorString := ";"
	terminatorBytes := []byte(terminatorString)
	toReplace := float64(strings.Count(q.sql, replacementString))
	bytesNeeded := float64(len(q.sql) + len(replacementString))
	powerCounter := float64(1)
	powerMax := math.Pow(10, powerCounter) - 1
	prevMax := float64(0)
	for powerMax < toReplace {
		bytesNeeded += ((powerMax - prevMax) * powerCounter)
		prevMax = powerMax
		powerCounter += 1
		powerMax = math.Pow(10, powerCounter) - 1
	}
	bytesNeeded += ((toReplace - prevMax) * powerCounter)
	output := make([]byte, int(bytesNeeded))
	buffer := make([]byte, utf8.UTFMax)
	for pos < len(q.sql) {
		r, width = utf8.DecodeRuneInString(q.sql[pos:])
		pos += width
		if r == replacementRune {
			newText := []byte(fmt.Sprintf("$%d", count))
			for _, b := range newText {
				output[outputPos] = b
				outputPos += 1
			}
			count += 1
			continue
		}
		used := utf8.EncodeRune(buffer[0:], r)
		for b := 0; b < used; b++ {
			output[outputPos] = buffer[b]
			outputPos += 1
		}
	}
	for i := 0; i < len(terminatorBytes); i++ {
		output[len(output)-(len(terminatorBytes)-i)] = terminatorBytes[i]
	}
	return string(output), nil
}

func (q *Query) Flush(join string) *Query {
	q.sql = strings.TrimSpace(q.sql) + " "
	q.sql += strings.TrimSpace(strings.Join(q.expressions, join))
	q.expressions = q.expressions[0:0]
	return q
}

func (q *Query) Expression(key string, values ...interface{}) *Query {
	q.expressions = append(q.expressions, key)
	q.args = append(q.args, values...)
	return q
}

func (q *Query) Where() *Query {
	if q.includesWhere {
		return q
	}
	q.Expression("WHERE")
	q.Flush(" ")
	q.includesWhere = true
	return q
}

func (q *Query) Comparison(obj SQLTableNamer, property, operator string, value interface{}) *Query {
	return q.Expression(Column(obj, property)+" "+operator+" ?", value)
}

func (q *Query) In(obj SQLTableNamer, property string, values ...interface{}) *Query {
	return q.Expression(Column(obj, property)+" IN("+VariableList(len(values))+")", values...)
}

func (q *Query) Assign(obj SQLTableNamer, property string, value interface{}) *Query {
	return q.Expression(Column(obj, property)+" = ?", value)
}

func (q *Query) orderBy(orderClause, dir string) *Query {
	exp := ", "
	if !q.includesOrder {
		exp = "ORDER BY "
		q.includesOrder = true
	}
	q.Expression(exp + orderClause + dir)
	return q
}

func (q *Query) OrderBy(column string) *Query {
	return q.orderBy(column, "")
}

func (q *Query) OrderByDesc(column string) *Query {
	return q.orderBy(column, " DESC")
}

func (q *Query) Limit(limit int64) *Query {
	return q.Expression("LIMIT ?", limit)
}

func (q *Query) Offset(offset int64) *Query {
	return q.Expression("OFFSET ?", offset)
}

func (q *Query) Args() []interface{} {
	return q.args
}
