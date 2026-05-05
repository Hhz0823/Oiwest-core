//go:build darwin

package platform

import (
	"os"
)

func isAndroid() bool {
	return false
}

func hasGUI() bool {
	if os.Getenv("DISPLAY") != "" {
		return true
	}
	return true
}

func detectDistro() DistroType {
	return DistroUnknown
}

func detectContainer() bool {
	return false
}
