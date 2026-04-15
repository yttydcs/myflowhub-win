// 本文件覆盖 VarPool 绑定助手与持久化规则的行为。

package main

import "testing"

func TestParseVarPoolSubPrefs_Empty(t *testing.T) {
	if out := parseVarPoolSubPrefs(""); out != nil {
		t.Fatalf("expected nil got %+v", out)
	}
}

func TestParseVarPoolSubPrefs_Invalid(t *testing.T) {
	if out := parseVarPoolSubPrefs("{"); out != nil {
		t.Fatalf("expected nil got %+v", out)
	}
}

func TestNormalizeVarPoolSubPrefs_TrimsFiltersAndDedupes(t *testing.T) {
	out := normalizeVarPoolSubPrefs([]VarPoolSubPref{
		{Name: " a ", Owner: 1, Subscribed: true},
		{Name: "", Owner: 1, Subscribed: true},
		{Name: "b", Owner: 0, Subscribed: true},
		{Name: "a", Owner: 1, Subscribed: false},
	})
	if len(out) != 1 {
		t.Fatalf("expected 1 pref got %d", len(out))
	}
	if out[0].Name != "a" || out[0].Owner != 1 || out[0].Subscribed != false {
		t.Fatalf("unexpected normalized pref: %+v", out[0])
	}
}
