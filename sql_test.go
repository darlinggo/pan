package pan

import (
	"testing"
	"time"
)

type testPost struct {
	ID       int
	Title    string
	Author   int `sql_column:"author_id"`
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
	sqlTable[New(MYSQL, "INSERT").Include("INTO ?", GetTableName(p)).Include("("+VariableList(len(fields))+")", []interface{}(fields)...).Include("VALUES").Include("("+VariableList(len(values))+")", values...).Generate(" ")] = "INSERT INTO ? (?,?,?,?,?,?) VALUES (?,?,?,?,?,?);"
	sqlTable[New(POSTGRES, "INSERT").Include("INTO ?", GetTableName(p)).Include("("+VariableList(len(fields))+")", []interface{}(fields)...).Include("VALUES").Include("("+VariableList(len(values))+")", values...).Generate(" ")] = "INSERT INTO $1 ($2,$3,$4,$5,$6,$7) VALUES ($8,$9,$10,$11,$12,$13);"
	sqlTable[New(POSTGRES, "UPDATE "+GetTableName(p)).Include("SET").FlushExpressions(" ").IncludeIfNotEmpty(GetColumn(p, "Title")+" = ?", p.Title).IncludeIfNotNil(GetColumn(p, "Modified")+" = ?", p.Modified).Include(GetColumn(p, "Author")+" = ?", p.Author).FlushExpressions(", ").IncludeWhere().Include("? = ?", GetColumn(p, "ID"), p).Generate(" ")] = "UPDATE test_data SET `title` = $1, `author_id` = $2 WHERE $3 = $4;"
	sqlTable[New(MYSQL, "UPDATE "+GetTableName(p)).Include("SET").FlushExpressions(" ").IncludeIfNotEmpty(GetColumn(p, "Title")+" = ?", p.Title).IncludeIfNotNil(GetColumn(p, "Modified")+" = ?", p.Modified).Include(GetColumn(p, "Author")+" = ?", p.Author).FlushExpressions(", ").IncludeWhere().Include("? = ?", GetColumn(p, "ID"), p).Generate(" ")] = "UPDATE test_data SET `title` = ?, `author_id` = ? WHERE ? = ?;"
	p.Modified = &p.Created
	p.Title = ""
	fields, values = GetAbsoluteFields(p)
	sqlTable[New(POSTGRES, "UPDATE "+GetTableName(p)).Include("SET").FlushExpressions(" ").IncludeIfNotEmpty(GetColumn(p, "Title")+" = ?", p.Title).IncludeIfNotNil(GetColumn(p, "Modified")+" = ?", p.Modified).Include(GetColumn(p, "Author")+" = ?", p.Author).FlushExpressions(", ").IncludeWhere().Include("? = ?", GetColumn(p, "ID"), p).Generate(" ")] = "UPDATE test_data SET `modified` = $1, `author_id` = $2 WHERE $3 = $4;"
	sqlTable[New(MYSQL, "UPDATE "+GetTableName(p)).Include("SET").FlushExpressions(" ").IncludeIfNotEmpty(GetColumn(p, "Title")+" = ?", p.Title).IncludeIfNotNil(GetColumn(p, "Modified")+" = ?", p.Modified).Include(GetColumn(p, "Author")+" = ?", p.Author).FlushExpressions(", ").IncludeWhere().Include("? = ?", GetColumn(p, "ID"), p).Generate(" ")] = "UPDATE test_data SET `modified` = ?, `author_id` = ? WHERE ? = ?;"
	sqlTable[New(POSTGRES, "SELECT "+QueryList(fields)).Include("FROM ?", GetTableName(p)).IncludeWhere().Include(GetColumn(p, "Created")+" > (SELECT "+GetColumn(p, "Created")+" FROM `"+GetTableName(p)+"` WHERE "+GetColumn(p, "ID")+" = ?)", 123).IncludeWhere().IncludeOrder(GetColumn(p, "Created")+" DESC").IncludeLimit(19).Generate(" ")] = "SELECT `test_data`.`id`, `test_data`.`title`, `test_data`.`author_id`, `test_data`.`body`, `test_data`.`created`, `test_data`.`modified` FROM $1 WHERE `created` > (SELECT `created` FROM `test_data` WHERE `id` = $2) ORDER BY `created` DESC LIMIT $3;"
	sqlTable[New(MYSQL, "SELECT "+QueryList(fields)).Include("FROM ?", GetTableName(p)).IncludeWhere().Include(GetColumn(p, "Created")+" > (SELECT "+GetColumn(p, "Created")+" FROM `"+GetTableName(p)+"` WHERE "+GetColumn(p, "ID")+" = ?)", 123).IncludeWhere().IncludeOrder(GetColumn(p, "Created")+" DESC").IncludeLimit(19).Generate(" ")] = "SELECT `test_data`.`id`, `test_data`.`title`, `test_data`.`author_id`, `test_data`.`body`, `test_data`.`created`, `test_data`.`modified` FROM ? WHERE `created` > (SELECT `created` FROM `test_data` WHERE `id` = ?) ORDER BY `created` DESC LIMIT ?;"
}

var sqlTable = map[string]string{
	New(MYSQL, "INSERT").Include("INTO ?", GetTableName(testPost{})).Include("("+VariableList(4)+")", "a", "b", "c", "d").Include("VALUES").Include("("+VariableList(4)+")", 0, 1, 2, 3).Generate(" "):    "INSERT INTO ? (?,?,?,?) VALUES (?,?,?,?);",
	New(POSTGRES, "INSERT").Include("INTO ?", GetTableName(testPost{})).Include("("+VariableList(4)+")", "a", "b", "c", "d").Include("VALUES").Include("("+VariableList(4)+")", 0, 1, 2, 3).Generate(" "): "INSERT INTO $1 ($2,$3,$4,$5) VALUES ($6,$7,$8,$9);",
}

func TestSQLTable(t *testing.T) {
	for output, expectation := range sqlTable {
		if output != expectation {
			if output == "" {
				t.Errorf("Expected %s, but there was an argument count error.", expectation)
			} else {
				t.Errorf("Expected '%s' got '%s'", expectation, output)
			}
		}
	}
}
