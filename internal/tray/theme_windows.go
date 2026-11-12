package tray

import "golang.org/x/sys/windows/registry"

func init() {
	readTaskbarTheme = func() (bool, error) {
		key, err := registry.OpenKey(registry.CURRENT_USER,
			`Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, registry.QUERY_VALUE)
		if err != nil {
			return false, err
		}
		defer key.Close()
		// The taskbar follows the system theme, not AppsUseLightTheme.
		light, _, err := key.GetIntegerValue("SystemUsesLightTheme")
		return light != 0, err
	}
}
