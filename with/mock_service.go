package with

import "github.com/go-andiamo/marrow/mocks/service"

func MockService(name string) With {
	return &mockService{
		name: name,
	}
}

type mockService struct {
	name string
	svc  service.MockedService
}

var _ With = (*mockService)(nil)

func (m *mockService) Init(init SuiteInit) (err error) {
	m.svc = service.NewMockedService(m.name)
	if err = m.svc.Start(); err == nil {
		init.AddMockService(m.svc)
	}
	return err
}

func (m *mockService) Stage() Stage {
	return Supporting
}

func (m *mockService) Shutdown() (fn func()) {
	if m.svc != nil {
		fn = m.svc.Shutdown
	}
	return fn
}
