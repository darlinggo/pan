package pan

import (
	"testing"
	"time"
)

type testPost struct {
	ID       int
	Title    string
	Author   int `sql_column:"author_id'`
	Body     string
	Created  time.Time
	Modified *time.Time
}

func (t testPost) GetSQLTableName() string {
	return "test_data"
}

func init() {
	p := testPost{123, "my post", 1, "this is a test post", time.Now(), nil}
	fields, values := GetQuotedFields(p)
	sqlTable[New("INSERT").Include("INTO ?", GetTableName(p)).Include("("+VariableList(len(fields))+")", []interface{}(fields)...).Include("VALUES").Include("("+VariableList(len(values))+")", values...).Generate(" ")] = "INSERT INTO $1 ($2,$3,$4,$5,$6,$7) VALUES ($8,$9,$10,$11,$12,$13);"
	sqlTable[New("UPDATE "+GetTableName(p)).Include("SET").FlushExpressions(" ").IncludeIfNotEmpty(GetColumn(p, "Title")+" = ?", p.Title).IncludeIfNotNil(GetColumn(p, "Modified")+" = ?", p.Modified).Include(GetColumn(p, "Author")+" = ?", p.Author).FlushExpressions(", ").IncludeWhere().Include("? = ?", GetColumn(p, "ID"), p).Generate(" ")] = "UPDATE test_data SET `title` = $1, `author` = $2 WHERE $3 = $4;"
}

var sqlTable = map[string]string{
	New("INSERT").Include("INTO ?", GetTableName(testPost{})).Include("("+VariableList(4)+")", "a", "b", "c", "d").Include("VALUES").Include("("+VariableList(4)+")", 0, 1, 2, 3).Generate(" "): "INSERT INTO $1 ($2,$3,$4,$5) VALUES ($6,$7,$8,$9);",
}

func TestSQLTable(t *testing.T) {
	for output, expectation := range sqlTable {
		if output != expectation {
			if output == "" {
				t.Errorf("Expected %s, but there was an argument count error.", expectation)
			} else {
				t.Errorf("Expected '%s', got '%s'", expectation, output)
			}
		}
	}
}
