# Importing pan
    import "darlinggo.co/pan"

# About pan

`pan` is an SQL query building and response unmarshalling library for Go. It is designed to be compatible with MySQL and PostgreSQL, but should be more or less agnostic. Please let us know if your favourite SQL flavour is not supported.

Pan is not designed to be an ORM, but it still eliminates much of the boilerplate code around writing queries and scanning over rows.

Pan’s design focuses on reducing repetition and hardcoded strings in your queries, but without limiting your ability to write any form of query you want. It is meant to be the smallest possible abstraction on top of SQL.

Docs can be found on [GoDoc.org](https://godoc.org/darlinggo.co/pan).

If you're using pan, we encourage you to join the [pan mailing list](https://groups.google.com/a/darlinggo.co/group/pan), which will be our main mode of communication.

# Using pan

Pan revolves around structs that fill the `SQLTableNamer` interface, by implementing the `GetSQLTableName() string` function, which just returns the name of the table that should store the data for that struct.

Let's say you have a `Person` in your code.

```go
type Person struct {
    ID    int     `sql_column:"person_id"`
    FName string  `sql_column:"fname"`
    LName string  `sql_column:"lname"`
    Age   int
}
```

And you have a corresponding `Person` table:

```
+-----------+-------------+------+-----+---------+-------+
| Field     | Type        | Null | Key | Default | Extra |
+-----------+-------------+------+-----+---------+-------+
| person_id | int         | NO   |     | NULL    |       |
| fname     | varchar(20) | NO   |     | ''      |       |
| lname     | varchar(20) | NO   |     | ''      |       |
| age       | int         | NO   |     | 0       |       |
+-----------+-------------+------+-----+---------+-------+
```

> **Note**: Unless you're using sql.NullString or equivalent, it's not recommended to allow `NULL` in your data. It may cause you trouble when unmarshaling.

To use that `Person` type with pan, you need it to fill the `SQLTableNamer` interface, letting pan know to use the `person` table in your database:

```go
func (p Person) GetSQLTableName() string {
    return "person"
}
```

## Creating a query

```go
// selects all rows
var p Person
query := pan.New(pan.MYSQL, "SELECT "+pan.Columns(p).String()+" FROM "+pan.Table(p))
```

or

```go
// selects one row
var p Person
query := pan.New(pan.MYSQL, "SELECT "+pan.Columns(p).String()+" FROM "+pan.Table(p)).Where()
query.Comparison(p, "ID", "=", 1)
query.Flush(" ")
```

That `Flush` command is important: pan works by creating a buffer of strings, and then joining them by some separator character. Flush takes the separator character (in this case, a space) and uses it to join all the buffered strings (in this case, the `WHERE` statement and the `person_id = ?` statement), and then adds the result to its query.

> It's safe to call `Flush` even if there are no buffered strings, so a good practice is to just call `Flush` after the entire query is built, just to make sure you don't leave anything buffered.

The `pan.Columns()` function returns the column names that a struct's properties correspond to. `pan.Columns().String()` joins them into a list of columns that can be passed right to the `SELECT` expression, making it easy to support reading only the columns you need, maintaining forward compatibility—your code will never choke on unexpected columns being added.

## Executing the query and reading results

```go
mysql, err := query.MySQLString() // could also be PostgreSQLString
if err != nil {
	// handle the error
}
rows, err := db.Query(mysql, query.Args...)
if err != nil {
	// handle the error
}
var people []Person
for rows.Next() {
	var p Person
        err := pan.Unmarshal(rows, &p) // put the results into the struct
        if err != nil {
        	// handle the error
        }
        people = append(people, p)
}
```

## How struct properties map to columns

There are a couple rules about how struct properties map to column names. First, only exported struct properties are used; unexported properties are ignored.

By default, a struct property's name is snake-cased, and that is used as the column name. For example, `Name` would become `name`, and `MyInt` would become `my_int`.

If you want more control or want to make columns explicit, the `sql_column` struct tag can be used to override this behaviour.

## Column flags

Sometimes, you need more than the base column name; you may need the full name (`table.column`) or you may be using special characters/need to quote the column name (`"column"` for Postgres, `\`column`\` for MySQL). To support these use cases, the `Column` and `Columns` functions take a variable number of flags (including none):

```go
Columns() // returns column format
Columns(FlagFull) // returns table.column format
Columns(FlagDoubleQuoted) // returns "column" format
Columns(FlagTicked) // returns `column` format
Columns(FlagFull, FlagDoubleQuoted) // returns "table"."column" format
Columns(FlagFull, FlagTicked) // returns `table`.`column` format
```

This behaviour is not exposed through the convenience functions built on top of `Column` and `Columns`; you'll need to use `Expression` to rebuild them by hand. Usually, this can be done simply; look at the source code for those convenience functions for examples.
