package app

import (
	"testing"
)

func TestToolkitLabel(t *testing.T) {
	cases := []struct {
		path  string
		label string
	}{
		{`plugins\2_FileSystem_Toolkit.ps1`, "FileSystem"},
		{`plugins\3_System_Toolkit.ps1`, "System"},
		{`plugins\4_Git_Toolkit.ps1`, "Git"},
		{`plugins\Browser_Toolkit.ps1`, "Browser"},
		{`plugins\Docker_Toolkit.ps1`, "Docker"},
		{`plugins\Help_Toolkit.ps1`, "Help"},
		{`plugins\Start_Dev_Toolkit.ps1`, "Start Dev"},
		{`plugins\STIBS\STIBS_Docker_Toolkit.ps1`, "STIBS Docker"},
		{`plugins\STIBS\STIBS_DB_Toolkit.ps1`, "STIBS DB"},
		{`plugins\M365\KVP_Star_Site_Toolkit.ps1`, "KVP Star Site"},
		{`plugins\M365\Star_IBS_Applications_Toolkit.ps1`, "Star IBS Applications"},
	}
	for _, tc := range cases {
		key := toolkitGroupKey(tc.path)
		got := toolkitLabel(key)
		if got != tc.label {
			t.Errorf("toolkitLabel(%q) = %q, want %q", tc.path, got, tc.label)
		}
	}
}
