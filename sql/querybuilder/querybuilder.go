package querybuilder

import (
	"fmt"
	"strings"
)

// DBType defines database type: mysql/postgresql
type DBType int
type Param struct {
	Key   string
	Value interface{}
}

const (
	// DBPostgres represents postgresql
	DBPostgres DBType = 1

	// DBMySQL represents mysql
	DBMySQL DBType = 2
)

// Builder represents simple query builder, mainly to prevent SQL injection
type Builder struct {
	sb       *strings.Builder
	counter  int
	args     []interface{}
	bindType DBType
}

// New creates new query builder.
//
// ? is used as placeholder and will be replaced with $ version if it is
// postgresql. It supports multiple placeholder, and the number of placeholders must
// match with the number of the data
func New(dbType DBType, baseQuery string, data ...interface{}) *Builder {
	var sb strings.Builder

	b := &Builder{
		sb:       &sb,
		bindType: dbType,
	}
	b.addQuery(baseQuery, data...)
	return b
}

// AddQuery append a query with the given format string and data.
//
// ? is used as placeholder and will be replaced with $ version if it is
// postgresql. It supports multiple placeholder, and the number of placeholders must
// match with the number of the data
//
// A space is prepended before the query
func (b *Builder) AddQuery(format string, data ...interface{}) {
	b.sb.WriteString(" ")
	b.addQuery(format, data...)
}

func (b *Builder) addQuery(format string, data ...interface{}) {
	b.args = append(b.args, data...)

	if b.bindType == DBPostgres {
		// TODO : compare with https://godoc.org/github.com/jmoiron/sqlx#Rebind
		// - see if we can do it faster with that package
		// - or improve our package to do that kind of optimization
		for i := 0; i < len(data); i++ {
			b.counter++
			format = strings.Replace(format, "?", fmt.Sprintf("$%d", b.counter), 1)
		}
	}
	b.sb.WriteString(format)
}

// AddString add raw string to the query.
//
// A space is prepended before the query
func (b *Builder) AddString(str string) {
	b.sb.WriteString(" " + str)
}

// Query returns the safe query with placeholder replaced as necessary
func (b *Builder) Query() string {
	return b.sb.String()
}

// Args returns the data arguments for the query
func (b *Builder) Args() []interface{} {
	return b.args
}

func GenerateQuery(query string, fn func() []Param) *Builder {
	args := fn()

	where := "WHERE "
	queryBuilder := New(DBPostgres, query)
	for i := 0; i < len(args); i++ {
		if i != 0 {
			queryBuilder.AddQuery(" AND "+args[i].Key+" = ?", args[i].Value)
			continue
		}

		queryBuilder.AddQuery(where+args[i].Key+" = ?", args[i].Value)
	}

	return queryBuilder
}
