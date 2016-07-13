q := pan.New("SELECT "+append(pan.Columns(post), "COUNT(*) FROM blah")+" FROM "+pan.Table(post))
pan.New("SELECT "+strings.Join(pan.Columns(post), ", ")+" FROM "+pan.Table(post))
q.Where().Comparison(post, "ID", "=", post.ID).Expression(" OR ").Comparison(post, "Name = ?", post.Name)
q.Expression(" OR ")
q.In(post, "Name", names)
q.OrderBy(post, "Name").OrderByDesc(post, "ID").Limit(maxResults)
--------
pan.Insert(post, ...values)
q := pan.NewExpression("INSERT flkfs into whaosh").Values(post, post, post)
.Values("klsdjglsjdg", 0, "alkghdskjghs")

func New(expression string) *Query {
}

func Insert(obj interface{}, values ...interface{}) *Query {
}

func Table(obj interface{}) string {}

func Column(obj interface{}, property string) string {}

func Columns(obj interface{}) Columns {}

type Columns []string
func (c Columns) String() string {
	return strings.Join(c, ",")
}

func (q *Query) Where() *Query {
}

func (q *Query) Comparison(obj interface{}, property, operator string, value interface{}) *Query {
}

func (q *Query) Expression(expression string, args ...interface{}) *Query {
}

func (q *Query) In(obj interface{}, property string, values interface{}) *Query {
}

func (q *Query) OrderBy(obj interface{}, property string) *Query {
}

func (q *Query) OrderByDesc(obj interface{}, property string) *Query {
}

func (q *Query) Limit(max int) *Query {
}

func (q *Query) Offset(max int) *Query {
}

func (q *Query) PostgreSQLString() string {
}

func (q *Query) MySQLString() string {
}

func (q *Query) Args() []interface{}

func (q *Query) String() string {
}
