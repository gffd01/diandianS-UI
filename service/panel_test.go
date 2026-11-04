package service

import "testing"

func TestPanelReleaseAPIUsesStandaloneRepository(t *testing.T) {
	const want = "https://api.github.com/repos/gffd01/diandianS-UI/releases/latest"
	if panelReleaseAPI != want {
		t.Fatalf("面板更新地址错误：得到 %q，期望 %q", panelReleaseAPI, want)
	}
}

func TestUniversalVersionComparison(t *testing.T) {
	tests := []struct {
		name    string
		latest  string
		current string
		want    bool
	}{
		{name: "相同版本不提示", latest: "universal-24", current: "universal-24", want: false},
		{name: "发现新版本", latest: "universal-25", current: "universal-24", want: true},
		{name: "不把旧版本当更新", latest: "universal-23", current: "universal-24", want: false},
		{name: "兼容 v 前缀", latest: "vuniversal-25", current: "universal-24", want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNewerVersion(tt.latest, tt.current); got != tt.want {
				t.Fatalf("isNewerVersion(%q, %q) = %v，期望 %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}
