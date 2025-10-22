package marrow

type BeforeAfter_ interface {
	When() When
	Runnable
}

type When int

const (
	Before When = iota
	After
)

type beforeAfter struct {
	when When
	do   Runnable
}

var _ BeforeAfter_ = (*beforeAfter)(nil)

func (b *beforeAfter) When() When {
	return b.when
}

func (b *beforeAfter) Run(ctx Context) error {
	return b.do.Run(ctx)
}

func (b *beforeAfter) Frame() *Frame {
	return b.do.Frame()
}

//go:noinline
func SetVar(when When, name string, value any) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &setVar{
			name:  name,
			value: value,
			frame: frame(0),
		},
	}
}

//go:noinline
func ClearVars(when When) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &clearVars{
			frame: frame(0),
		},
	}
}

//go:noinline
func DbInsert(when When, tableName string, row Columns) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbInsert{
			tableName: tableName,
			row:       row,
			frame:     frame(0),
		},
	}
}

//go:noinline
func DbExec(when When, query string, args ...any) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbExec{
			query: query,
			args:  args,
			frame: frame(0),
		},
	}
}

//go:noinline
func DbClearTable(when When, tableName string) BeforeAfter_ {
	return &beforeAfter{
		when: when,
		do: &dbClearTable{
			tableName: tableName,
			frame:     frame(0),
		},
	}
}
