[![Build Status](https://travis-ci.org/secondbit/pan.png)](https://travis-ci.org/secondbit/pan)

# pan
    import "github.com/DramaFever/pan"




## Constants
``` go
const (
    MYSQL dbengine = iota
    POSTGRES
)
```
``` go
const TAG_NAME = "sql_column" // The tag that will be read

```


## func GetAbsoluteColumnName
``` go
func GetAbsoluteColumnName(s sqlTableNamer, property string) string
```
GetAbsoluteColumnName returns the field name associated with the specified property in the passed value.
Property must correspond exactly to the name of the property in the type, or this function will
panic.


## func GetAbsoluteFields
``` go
func GetAbsoluteFields(s sqlTableNamer) (fields []interface{}, values []interface{})
```
GetAbsoluteFields returns a slice of the fields in the passed type, with their names
drawn from tags or inferred from the property name (which will be lower-cased with underscores,
e.g. CamelCase => camel_case) and a corresponding slice of interface{}s containing the values for
those properties. Fields will be surrounded in \` marks and prefixed with their table name, as
determined by the passed type's GetSQLTableName. The format will be \`table_name\`.\`field_name\`.


## func GetColumn
``` go
func GetColumn(s interface{}, property string) string
```
GetColumn returns the field name associated with the specified property in the passed value.
Property must correspond exactly to the name of the property in the type, or this function will
panic.


## func GetM2MAbsoluteColumnName
``` go
func GetM2MAbsoluteColumnName(t sqlTableNamer, field string, t2 sqlTableNamer) string
```
GetM2MAbsoluteColumnName returns the column name for the supplied field in a many-to-many relationship table,
including the table name. The field belongs to the first sqlTableNamer, the second sqlTableNamer is the other
table in the many-to-many relationship.


## func GetM2MColumnName
``` go
func GetM2MColumnName(t sqlTableNamer, field string) string
```
GetM2MColumnName returns the column name for the supplied field in a many-to-many relationship table.


## func GetM2MFields
``` go
func GetM2MFields(t1 sqlTableNamer, field1 string, t2 sqlTableNamer, field2 string) (columns, values []interface{})
```
GetM2MFields returns a slice of the columns that should be in a table that maps the many-to-many relationship of
the types supplied, with their corresponding values. The field parameters specify the primary keys used in
the relationship table to map to that type.


## func GetM2MQuotedColumnName
``` go
func GetM2MQuotedColumnName(t sqlTableNamer, field string) string
```
GetM2MQuotedColumnName returns the column name for the supplied field in a many-to-many relationship table,
including the quote marks around the column name.


## func GetM2MQuotedFields
``` go
func GetM2MQuotedFields(t1 sqlTableNamer, field1 string, t2 sqlTableNamer, field2 string) (columns, values []interface{})
```
GetM2MQuotedFields wraps the fields returned by GetM2MFields in quotes.


## func GetM2MTableName
``` go
func GetM2MTableName(t1, t2 sqlTableNamer) string
```
GetM2MTableName returns a consistent table name for a many-to-many relationship between two tables. No
matter what order the fields are passed in, the resulting table name will always be consistent.


## func GetQuotedFields
``` go
func GetQuotedFields(s sqlTableNamer) (fields []interface{}, values []interface{})
```
GetQuotedFields returns a slice of the fields in the passed type, with their names
drawn from tags or inferred from the property name (which will be lower-cased with underscores,
e.g. CamelCase => camel_case) and a corresponding slice of interface{}s containing the values for
those properties. Fields will be surrounding in ` marks.


## func GetTableName
``` go
func GetTableName(t sqlTableNamer) string
```
GetTableName returns the table name for any type that implements the `GetSQLTableName() string`
method signature. The returned string will be used as the name of the table to store the data
for all instances of the type.


## func GetUnquotedColumn
``` go
func GetUnquotedColumn(s interface{}, property string) string
```

## func QueryList
``` go
func QueryList(fields []interface{}) string
```
QueryList joins the passed fields into a string that can be used when selecting the fields to return
or specifying fields in an update or insert.


## func Unmarshal
``` go
func Unmarshal(s Scannable, dst interface{}) error
```

## func VariableList
``` go
func VariableList(num int) string
```
VariableList returns a list of `num` variable placeholders for use in SQL queries involving slices
and arrays.



## type Query
``` go
type Query struct {
    SQL           string
    Args          []interface{}
    Expressions   []string
    IncludesWhere bool
    IncludesOrder bool
    IncludesLimit bool
    Engine        dbengine
}
```
Query contains the data needed to perform a single SQL query.









### func New
``` go
func New(engine dbengine, query string) *Query
```
New creates a new Query object. The passed engine is used to format variables. The passed string is used to prefix the query.




### func (\*Query) FlushExpressions
``` go
func (q *Query) FlushExpressions(join string) *Query
```
FlushExpressions joins the Query's Expressions with the join string, then concatenates them
to the Query's SQL. It then resets the Query's Expressions. This permits Expressions to be joined
by different strings within a single Query.



### func (\*Query) Generate
``` go
func (q *Query) Generate(join string) string
```
Generate creates a string from the Query, joining its SQL property and its Expressions. Expressions are joined
using the join string supplied.



### func (\*Query) Include
``` go
func (q *Query) Include(key string, values ...interface{}) *Query
```
Include adds the supplied key (which should be an expression) to the Query's Expressions and the value
to the Query's Args.



### func (\*Query) IncludeIfNotEmpty
``` go
func (q *Query) IncludeIfNotEmpty(key string, value interface{}) *Query
```
IncludeIfNotEmpty adds the supplied key (which should be an expression) to the Query's Expressions if
and only if the value parameter is not the empty value for its type. If the key is added to the Query's
Expressions, the value is added to the Query's Args.



### func (\*Query) IncludeIfNotNil
``` go
func (q *Query) IncludeIfNotNil(key string, value interface{}) *Query
```
IncludeIfNotNil adds the supplied key (which should be an expression) to the Query's Expressions if
and only if the value parameter is not a nil value. If the key is added to the Query's Expressions, the
value is added to the Query's Args.



### func (\*Query) IncludeLimit
``` go
func (q *Query) IncludeLimit(limit int64) *Query
```
IncludeLimit includes the LIMIT clause if the LIMIT clause has not already been included in the Query.
This cannot detect LIMIT clauses that are manually added to the Query's SQL; it only tracks IncludeLimit().
The passed int is used as the limit in the resulting query.



### func (\*Query) IncludeOffset
``` go
func (q *Query) IncludeOffset(offset int64) *Query
```


### func (\*Query) IncludeOrder
``` go
func (q *Query) IncludeOrder(orderClause string) *Query
```
IncludeOrder includes the ORDER BY clause if the ORDER BY clause has not already been included in the Query.
This cannot detect ORDER BY clauses that are manually added to the Query's SQL; it only tracks IncludeOrder().
The passed string is used as the expression to order by.



### func (\*Query) IncludeWhere
``` go
func (q *Query) IncludeWhere() *Query
```
IncludeWhere includes the WHERE clause if the WHERE clause has not already been included in the Query.
This cannot detect WHERE clauses that are manually added to the Query's SQL; it only tracks IncludeWhere().



### func (\*Query) InnerJoin
``` go
func (q *Query) InnerJoin(table, expression string) *Query
```


### func (\*Query) String
``` go
func (q *Query) String() string
```
String fulfills the String interface for Queries, and returns the generated SQL query after every instance of ?
has been replaced with a counter prefixed with $ (e.g., $1, $2, $3). There is no support for using ?, quoted or not,
within an expression. All instances of the ? character that are not meant to be substitutions should be as arguments
in prepared statements.



## type Scannable
``` go
type Scannable interface {
    Scan(dest ...interface{}) error
}
```










## type WrongNumberArgsError
``` go
type WrongNumberArgsError struct {
    NumExpected int
    NumFound    int
}
```
WrongNumberArgsError is thrown when a Query is evaluated whose Args does not match its Expressions.











### func (WrongNumberArgsError) Error
``` go
func (e WrongNumberArgsError) Error() string
```
Error fulfills the error interface, returning the expected number of arguments and the number supplied.









- - -
Generated by [godoc2md](http://godoc.org/github.com/davecheney/godoc2md)
