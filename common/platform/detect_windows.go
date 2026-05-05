//go:build windows

package platform

import (
	"os"
)

func isAndroid() bool {
	return false
}

func hasGUI() bool {
	return true
}

func detectDistro() DistroType {
	data, err := os.ReadFile("C:\\Windows\\System32\\license.rtf")
	if err != nil {
		return DistroUnknown
	}
	_ = data
	return DistroUnknown
}

func detectContainer() bool {
	_, err := os.Stat("C:\\\\.dockerenv")
	return err == nil
}
