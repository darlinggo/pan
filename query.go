package pan

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"unicode/utf8"
)

type Query struct {
	SQL           string
	Args          []interface{}
	Expressions   []string
	IncludesWhere bool
	IncludesOrder bool
	IncludesLimit bool
}

func New() *Query {
	return &Query{
		SQL:  "",
		Args: []interface{}{},
	}
}

type WrongNumberArgsError struct {
	NumExpected int
	NumFound    int
}

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

func (q *Query) Generate(join string) string {
	if len(q.Expressions) > 0 {
		q.FlushExpressions(join)
	}
	return q.String()
}

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

func (q *Query) FlushExpressions(join string) {
	q.SQL = strings.TrimSpace(q.SQL) + " "
	q.SQL += strings.TrimSpace(strings.Join(q.Expressions, join))
	q.Expressions = q.Expressions[0:0]
}

func (q *Query) IncludeIfNotNil(key string, value interface{}) {
	val := reflect.ValueOf(value)
	kind := val.Kind()
	if kind != reflect.Map && kind != reflect.Ptr && kind != reflect.Slice {
		return
	}
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, value)
}

func (q *Query) IncludeIfNotEmpty(key string, value interface{}) {
	if reflect.DeepEqual(value, reflect.Zero(reflect.TypeOf(value)).Interface()) {
		return
	}
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, value)
}

func (q *Query) Include(key string, values ...interface{}) {
	q.Expressions = append(q.Expressions, key)
	q.Args = append(q.Args, values...)
}

func (q *Query) IncludeWhere() {
	if q.IncludesWhere {
		return
	}
	q.Expressions = append(q.Expressions, "WHERE")
	q.FlushExpressions(" ")
	q.IncludesWhere = true
}

func (q *Query) IncludeOrder(orderClause string) {
	if q.IncludesOrder {
		return
	}
	q.Expressions = append(q.Expressions, "ORDER BY ")
	q.IncludesOrder = true
}

func (q *Query) IncludeLimit(limit int) {
	if q.IncludesLimit {
		return
	}
	q.Expressions = append(q.Expressions, " LIMIT ?")
	q.Args = append(q.Args, limit)
	q.IncludesLimit = true
}
