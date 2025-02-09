package isr

type name struct {
	value string
}

func Name(value string) name {
	return name{value: value}
}
