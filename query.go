package pan

import (
	"fmt"
	"strconv"
	"strings"
)

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
		query.Expression("("+Placeholders(len(columnValues))+")", columnValues...)
	}
	return query.Flush(" ")
}

type ErrWrongNumberArgs struct {
	NumExpected int
	NumFound    int
}

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
	var argPos int
	var res string
	toCheck := q.sql
	for i := strings.Index(toCheck, "?"); i >= 0; argPos++ {
		var arg interface{}
		arg = "!{MISSING}"
		if len(q.args) > argPos {
			arg = q.args[argPos]
		}
		res += toCheck[:i]
		res += fmt.Sprintf("%v", arg)
		toCheck = toCheck[i+1:]
		i = strings.Index(toCheck, "?")
	}
	res += toCheck
	return res
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
	count := 1
	var res string
	toCheck := q.sql
	for i := strings.Index(toCheck, "?"); i >= 0; count++ {
		res += toCheck[:i]
		res += "$" + strconv.Itoa(count)
		toCheck = toCheck[i+1:]
		i = strings.Index(toCheck, "?")
	}
	res += toCheck
	return res + ";", nil
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
	return q.Expression(Column(obj, property)+" IN("+Placeholders(len(values))+")", values...)
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
