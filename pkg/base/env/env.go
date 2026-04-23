package env

import (
	"sync/atomic"
)

var (
	ModeRelease Mode = 0
	ModeDevelop Mode = 1
	ModeDebug   Mode = 2
)

var modeName = []string{
	"release",
	"develop",
	"debug",
}

var nameState atomic.Pointer[string]
var envState atomic.Pointer[string]
var modeState atomic.Pointer[Mode]

func init() {
	SetName("")
	SetMode(ModeRelease)
}

func SetName(name string) {
	nameState.Store(&name)
}

func GetName() string {
	return *nameState.Load()
}

func SetEnv(env string) {
	envState.Store(&env)
}

func GetEnv() string {
	p := envState.Load()
	if p == nil {
		return ""
	}
	return *p
}

func ParseMode(name string) Mode {
	switch name {
	case "debug":
		return ModeDebug
	case "develop":
		return ModeDevelop
	case "release":
		return ModeRelease
	default:
		return ModeRelease
	}
}

func SetMode(mode Mode) {
	modeState.Store(&mode)
}

func GetMode() Mode {
	return *modeState.Load()
}

func IsModeActive(mode Mode) bool {
	return (*modeState.Load()).IsActive(mode)
}
