//go:build android

package platform

func isAndroid() bool {
	return true
}

func hasGUI() bool {
	return false
}

func detectDistro() DistroType {
	return DistroAndroid
}

func detectContainer() bool {
	return false
}
