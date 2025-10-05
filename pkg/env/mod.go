package env

type Mode int

func (m Mode) String() string {
	if m < 0 || int(m) >= len(modeName) {
		return "unknown"
	}
	return modeName[m]
}

func (m Mode) IsActive(mode Mode) bool {
	return m >= mode
}
