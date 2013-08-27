package pan

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type Query struct {
	SQL string
	Args []interface{}
	IncludesWhere bool
	IncludesOrder bool
	IncludesLimit bool
}

func New() *Query{
	return &Query {
		SQL: "",
		Args: []interface{}{},
	}
}

type WrongNumberArgsError struct {
	NumExpected int
	NumFound int
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

func (q *Query) String() string {
	if err := q.checkCounts(); err != nil {
		return ""
	}
	var pos, width, outputPos int
	var r rune
	var count = 1
	replacementRune, _ := utf8.DecodeRune([]byte("?"))
	toReplace := strings.Count(q.SQL, "?")
	bytesNeeded := len(q.SQL) + (toReplace/10) + 1 // we're adding an extra character, we need to buffer for it
	output := make([]byte, bytesNeeded)
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
			continue
		}
		used := utf8.EncodeRune(buffer[0:], r)
		for b := 0; b < used; b++ {
			output[outputPos] = buffer[b]
			outputPos += 1
		}
	}
	return string(output)+";"
}

func (q *Query) IncludeWhere() {
	q.SQL = strings.TrimSpace(q.SQL)
	q.SQL += " "
	if q.IncludesWhere {
		return
	}
	q.SQL += "WHERE "
	q.IncludesWhere = true
}

func (q *Query) IncludeOrder() {
	q.SQL = strings.TrimSpace(q.SQL)
	q.SQL += " "
	if q.IncludesOrder {
		return
	}
	q.SQL += "ORDER BY "
	q.IncludesOrder = true
}

func (q *Query) IncludeLimit(limit int) {
	q.SQL = strings.TrimSpace(q.SQL)
	if q.IncludesLimit {
		return
	}
	q.SQL += " LIMIT $1"
	q.Args = append(q.Args, limit)
	q.IncludesLimit = true
}
