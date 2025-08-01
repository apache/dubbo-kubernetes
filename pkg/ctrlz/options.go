package ctrlz

type Options struct {
	Port    uint16
	Address string
}

const DefaultControlZPort = 9876

func DefaultOptions() *Options {
	return &Options{
		Port:    DefaultControlZPort,
		Address: "localhost",
	}
}
