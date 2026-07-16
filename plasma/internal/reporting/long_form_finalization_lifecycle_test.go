package reporting

import "testing"

func TestRecoverLongFormFinalizationHint(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		text string
		ok   bool
	}{
		{"root trailing comma", " {\r\n\t\"front_matter\":\"# 제목, {범위}\\\\\",\"closing\":\"## 끝 — 확인\",\r\n } ", true},
		{"valid json", `{"front_matter":"a","closing":"b"}`, false},
		{"two commas", `{"front_matter":"a","closing":"b",,}`, false},
		{"nested comma", `{"front_matter":{"x":"a",},"closing":"b",}`, false},
		{"array", `["a",]`, false},
		{"fence", "```json\n{\"front_matter\":\"a\",\"closing\":\"b\",}\n```", false},
		{"prefix", `answer {"front_matter":"a","closing":"b",}`, false},
		{"suffix", `{"front_matter":"a","closing":"b",} answer`, false},
		{"second value", `{"front_matter":"a","closing":"b",}{}`, false},
		{"unknown", `{"front_matter":"a","closing":"b","extra":"c",}`, false},
		{"missing", `{"front_matter":"a",}`, false},
		{"duplicate", `{"front_matter":"a","front_matter":"x","closing":"b",}`, false},
		{"non string", `{"front_matter":1,"closing":"b",}`, false},
		{"truncated", `{"front_matter":"a","closing":"b",`, false},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			hint := RecoverLongFormFinalizationHint(test.text)
			if hint.Available != test.ok {
				t.Fatalf("available=%v, want %v: %#v", hint.Available, test.ok, hint)
			}
		})
	}
}

func FuzzRecoverLongFormFinalizationHint(f *testing.F) {
	f.Add(`{"front_matter":"# A","closing":"## B",}`)
	f.Add(`{"front_matter":"quoted \", comma, and }","closing":"slash \\\",}`)
	f.Fuzz(func(t *testing.T, text string) {
		hint := RecoverLongFormFinalizationHint(text)
		if hint.Available && (hint.OpeningMarkdown == "" && hint.ClosingMarkdown == "") {
			t.Fatal("available hint decoded neither field")
		}
	})
}
