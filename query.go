package pan

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"
)

// Query contains the data needed to perform a single SQL query.
type Query struct {
	SQL           string
	Args          []interface{}
	Expressions   []string
	IncludesWhere bool
	IncludesOrder bool
	IncludesLimit bool
}

// New creates a new Query object. The passed string is used to prefix the query.
func New(query string) *Query {
	return &Query{
		SQL:  query,
		Args: []interface{}{},
	}
}

// WrongNumberArgsError is thrown when a Query is evaluated whose Args does not match its Expressions.
type WrongNumberArgsError struct {
	NumExpected int
	NumFound    int
}

// Error fulfills the error interface, returning the expected number of arguments and the number supplied.
func (e WrongNumberArgsError) Error() string {
	return fmt.Sprintf("Expected %d arguments, got %d.", e.NumExpected, e.NumFound)
}

func (q *Query) checkCounts() error {
	placeholders := strings.Count(q.SQL, "?")
	args := len(q.Args)
	if placeholders != args {
		return WrongNumberArgsError{NumExpected: placeholders, NumFound: args}
	}
	return nil
}

// Generate creates a string from the Query, joining its SQL property and its Expressions. Expressions are joined
// using the join string supplied.
func (q *Query) Generate(join string) string {
	if len(q.Expressions) > 0 {
		q.FlushExpressions(join)
	}
	return q.String()
}

// String fulfills the String interface for Queries, and returns the generated SQL query after every instance of ?
// has been replaced with a counter prefixed with $ (e.g., $1, $2, $3). There is no support for using ?, quoted or not,
// within an expression. All instances of the ? character that are not meant to be substitutions should be as arguments
// in prepared statements.
func (q *Query) String() string {
	if err := q.checkCounts(); err != nil {
		return ""
	}
	var pos, width, outputPos int
	var r rune
	var count = 1
	replacementRune, _ := utf8.DecodeRune([]byte("?"))
	terminator := []byte(";")
	toReplace := float64(strings.Count(q.SQL, "?"))
	bytesNeeded := float64(len(q.SQL) + len(";"))
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
	for pos < len(q.SQL) {
		r, width = utf8.DecodeRuneInString(q.SQL[pos:])
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
	for i := 0; i < len(terminator); i++ {
		output[len(output)-(len(terminator)-i)] = terminator[i]
	}
	return string(output)
}

// FlushExpressions joins the Query's Expressions with the join string, then concatenates them
// to the Query's SQL. It then resets the Query's Expressions. This permits Expressions to be joined
// by different strings within a single Query.
func (q *Query) FlushExpressions(join string) *Query {
	q.SQL = strings.TrimSpace(q.SQL) + " "
	q.SQL += strings.TrimSpace(strings.Join(q.Expressions, join))
	q.Expressions = q.Expressions[0:0]
	return q
}

// IncludeIfNotNil adds the supplied key (which should be an expression) to the Query's Expressions if
// and only if the value parameter is not a nil value. If the key is added to the Query's Expressions, the
// value is added to the Query's Args.
func (q *Query) IncludeIfNotNil(key string, value interface{}) *Query {
	val := reflect.ValueOf(value)
	kind := val.Kind()
	if kind != reflect.Map && kind != reflect.Ptr && kind != reflect.Slice {
		return q
	}
	if val.IsNil() {
		return q
	}
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, value)
	return q
}

// IncludeIfNotEmpty adds the supplied key (which should be an expression) to the Query's Expressions if
// and only if the value parameter is not the empty value for its type. If the key is added to the Query's
// Expressions, the value is added to the Query's Args.
func (q *Query) IncludeIfNotEmpty(key string, value interface{}) *Query {
	if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return q
	}
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, value)
	return q
}

// Include adds the supplied key (which should be an expression) to the Query's Expressions and the value
// to the Query's Args.
func (q *Query) Include(key string, values ...interface{}) *Query {
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, values...)
	return q
}

// IncludeWhere includes the WHERE clause if the WHERE clause has not already been included in the Query.
// This cannot detect WHERE clauses that are manually added to the Query's SQL; it only tracks IncludeWhere().
func (q *Query) IncludeWhere() *Query {
	if q.IncludesWhere {
		return q
	}
	q.Expressions = append(q.Expressions, "WHERE")
	q.FlushExpressions(" ")
	q.IncludesWhere = true
	return q
}

// IncludeOrder includes the ORDER BY clause if the ORDER BY clause has not already been included in the Query.
// This cannot detect ORDER BY clauses that are manually added to the Query's SQL; it only tracks IncludeOrder().
// The passed string is used as the expression to order by.
func (q *Query) IncludeOrder(orderClause string) *Query {
	if q.IncludesOrder {
		return q
	}
	q.Expressions = append(q.Expressions, "ORDER BY ")
	q.IncludesOrder = true
	return q
}

// IncludeLimit includes the LIMIT clause if the LIMIT clause has not already been included in the Query.
// This cannot detect LIMIT clauses that are manually added to the Query's SQL; it only tracks IncludeLimit().
// The passed int is used as the limit in the resulting query.
func (q *Query) IncludeLimit(limit int) *Query {
	if q.IncludesLimit {
		return q
	}
	q.Expressions = append(q.Expressions, " LIMIT ?")
	q.Args = append(q.Args, limit)
	q.IncludesLimit = true
	return q
}
