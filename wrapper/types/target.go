package types

type Targets []string

func (ts Targets) IsTarget(target string) bool {
	for _, t := range ts {
		if t == target {
			return true
		}
	}

	return false
}
