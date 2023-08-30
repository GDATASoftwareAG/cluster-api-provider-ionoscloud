package ionos

type Create func(user, password, token, host string) IONOSClient

func NewClientFactory(create Create) *ClientFactory {
	return &ClientFactory{create}
}

type ClientFactory struct {
	create func(user, password, token, host string) IONOSClient
}

func (f *ClientFactory) GetClient(user, password, token, host string) IONOSClient {
	return f.create(user, password, token, host)
}
