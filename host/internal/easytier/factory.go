package easytier

func NewDefaultAdapter() Adapter {
	adapter, err := newPlatformAdapter()
	if err == nil {
		return adapter
	}

	stub := NewStubAdapter()
	stub.setLastError(err)
	return stub
}
