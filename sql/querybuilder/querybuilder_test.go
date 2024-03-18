package querybuilder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQueryBuilder(t *testing.T) {
	var (
		name      = "my_name"
		address   = "my_address"
		fullQuery = "select * from user where name=$1 and address=$2 order by name"
		args      = []interface{}{name, address}
	)
	qb := New(DBPostgres, "select * from user where")
	qb.AddQuery("name=?", name)
	qb.AddQuery("and address=?", address)
	qb.AddQuery("order by name")

	require.Equal(t, fullQuery, qb.Query())
	require.Equal(t, args, qb.Args())
}

func TestQueryBuilderNewWithArgs(t *testing.T) {
	var (
		age       = 10
		name      = "my_name"
		address   = "my_address"
		fullQuery = "select * from user where age=$1 and name=$2 and address=$3"
		args      = []interface{}{age, name, address}
	)
	qb := New(DBPostgres, "select * from user where age=?", age)
	qb.AddQuery("and name=?", name)
	qb.AddQuery("and address=?", address)

	require.Equal(t, fullQuery, qb.Query())
	require.Equal(t, args, qb.Args())
}
