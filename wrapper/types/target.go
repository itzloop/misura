package types

type Strings []string

func (ts Strings) Exists(target string) bool {
	for _, t := range ts {
		if t == target {
			return true
		}
	}

	return false
}
