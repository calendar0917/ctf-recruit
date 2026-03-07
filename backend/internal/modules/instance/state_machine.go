package instance

var validTransitions = map[Status]map[Status]struct{}{
	StatusStarting: {
		StatusRunning: {},
		StatusFailed:  {},
	},
	StatusRunning: {
		StatusStopping: {},
		StatusExpired:  {},
		StatusFailed:   {},
	},
	StatusStopping: {
		StatusStopped: {},
		StatusExpired: {},
		StatusFailed:  {},
	},
	StatusStopped: {
		StatusCooldown: {},
	},
	StatusExpired: {
		StatusCooldown: {},
	},
	StatusFailed: {
		StatusCooldown: {},
	},
	StatusCooldown: {},
}

func CanTransition(from, to Status) bool {
	if from == to {
		return true
	}
	next, ok := validTransitions[from]
	if !ok {
		return false
	}
	_, ok = next[to]
	return ok
}
