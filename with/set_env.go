package with

import "os"

// SetEnv initialises a marrow.Suite with an environment variable set
func SetEnv(key, value string) With {
	return &setEnv{
		key:   key,
		value: value,
	}
}

type setEnv struct {
	key   string
	value string
}

var _ With = (*setEnv)(nil)

func (s *setEnv) Init(init SuiteInit) error {
	return os.Setenv(s.key, s.value)
}

func (s *setEnv) Stage() Stage {
	return Initial
}

func (s *setEnv) Shutdown() func() {
	return nil
}
