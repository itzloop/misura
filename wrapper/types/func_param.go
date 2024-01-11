package types

import (
	"fmt"
	"strings"
)

type FuncParam struct {
	Name string
	Type string
}

type FuncParams []FuncParam

func (f FuncParams) JoinNames() string {
	str := ""
	for i, p := range f {
		n := p.Name
		if strings.Contains(p.Type, "...") {
			n += "..."
		}
		if i == len(f)-1 {
			str += n
			continue
		}
		str += n + ", "
	}

	return str
}

func (f FuncParams) JoinTypes() string {
	str := ""
	for i, p := range f {
		if i == len(f)-1 {
			str += fmt.Sprintf("%s", p.Type)
			continue
		}
		str += fmt.Sprintf("%s, ", p.Type)
	}

	return str
}

func (f FuncParams) Join() string {
	str := ""
	for i, p := range f {
		if i == len(f)-1 {
			str += fmt.Sprintf("%s %s", p.Name, p.Type)
			continue
		}
		str += fmt.Sprintf("%s %s, ", p.Name, p.Type)
	}

	return str
}
