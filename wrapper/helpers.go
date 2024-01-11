package wrapper

func genNameHelper(count int) func() string {
	start := -1
	if count < 1 {
		count = 1
	}

	alphabet := "abcdefghijklmnopqrstuvwxyz"

	return func() string {
		start++
		if start == len(alphabet) {
			start = 0
			count++
		}

		if start+count <= len(alphabet) {
			return string([]byte(alphabet)[start : start+count])
		}

		return string([]byte(alphabet)[start:start+count]) + string([]byte(alphabet)[0:start+count-len(alphabet)])
	}
}
