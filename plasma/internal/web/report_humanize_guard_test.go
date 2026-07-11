package web

import "testing"

func TestValidateHumanizedMarkdownRejectsProtectedTokenDrift(t *testing.T) {
	original := "# Report\n\nGemma 4는 `KV cache`를 사용한다.\n\n출처: https://example.com/report\n\n인용: \"모델 카드는 128K context를 명시한다.\"\n"
	humanized := "# Report\n\nGemma 4는 `KV 캐시`를 사용한다.\n\n출처: https://example.com/report\n\n인용: \"모델 카드는 128K context를 명시한다.\"\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected inline code drift to fail")
	}

	humanized = "# Report\n\nGemma 4는 `KV cache`를 사용한다.\n\n참고: https://example.com/report\n\n인용: \"모델 카드는 128K context를 명시한다.\"\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected source-bearing line drift to fail")
	}

	humanized = "# Report\n\nGemma 4는 `KV cache`를 사용한다.\n\n출처: https://example.com/report\n\n인용: \"모델 카드는 긴 context를 명시한다.\"\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected quote and number drift to fail")
	}

	humanized = "# Report\n\nGemma four는 `KV cache`를 사용한다.\n\n출처: https://example.com/report\n\n인용: \"모델 카드는 128K context를 명시한다.\"\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected latin technical token drift to fail")
	}

	original = "# Report\n\n본문입니다.\n\n## 출처\n\n- 자료 제목, 내부 스모크 fixture.\n"
	humanized = "# Report\n\n본문입니다.\n\n## 출처\n\n- 자료명, 내부 스모크 fixture.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected source section line drift to fail")
	}
}

func TestValidateHumanizedMarkdownRejectsSameBlockContentDrop(t *testing.T) {
	original := "# Report\n\n- 첫 항목입니다.\n- 둘째 항목입니다.\n\n한 문장입니다. 둘째 문장입니다.\n"
	humanized := "# Report\n\n- 첫 항목입니다.\n\n한 문장입니다. 둘째 문장입니다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected list item drop to fail")
	}

	humanized = "# Report\n\n- 첫 항목입니다.\n- 둘째 항목입니다.\n\n한 문장입니다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected same-block sentence drop to fail")
	}

	original = "# Report\n\n핵심 판단은 검증 가능한 원문에 근거해야 하며, 결론은 자료의 한계를 함께 설명해야 한다. 보고서는 출처와 인용을 보존하고, 독자가 같은 자료를 다시 확인할 수 있게 해야 한다. 이 문장은 그대로 유지된다.\n"
	humanized = "# Report\n\n전혀 다른 결론을 제시하지만 문장 수는 그대로 유지하고, 원래 자료와 무관한 판단을 덧붙인다. 보고서는 검증보다 인상을 우선하며, 독자가 같은 자료를 확인할 필요는 없다고 말한다. 이 문장은 그대로 유지된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected broad same-shape prose rewrite to fail")
	}

	original = "# Report\n\n첫 문장은 보존되어야 한다.\n둘째 문장도 보존되어야 한다.\n셋째 문장도 보존되어야 한다.\n넷째 문장도 보존되어야 한다.\n다섯째 문장도 보존되어야 한다.\n"
	humanized = "# Report\n\n첫 문장은 다르게 바뀌어야 한다.\n둘째 문장도 다르게 바뀌어야 한다.\n셋째 문장도 다르게 바뀌어야 한다.\n넷째 문장도 다르게 바뀌어야 한다.\n다섯째 문장도 다르게 바뀌어야 한다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short report full-body rewrite to fail")
	}

	original = "# Report\n\n핵심 판단은 원문에 근거해야 한다.\n"
	humanized = "# Report\n\n핵심 판단은 인상에 맞춰 바꿔도 된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short single-paragraph semantic rewrite to fail")
	}

	original = "# Report\n\n핵심 판단은 반드시 원문에 근거해야 한다.\n"
	humanized = "# Report\n\n핵심 판단은 반드시 인상에 근거해야 한다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short content-word replacement to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 불가능하다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short polarity replacement to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 가능하지 않다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 안 된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short standalone negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 권장된다.\n"
	humanized = "# Report\n\n이 접근은 안 권장된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short adverbial negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 안된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short joined negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 안돼.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short contracted negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 권장된다.\n"
	humanized = "# Report\n\n이 접근은 안한다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short joined action negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 가능하다.\n"
	humanized = "# Report\n\n이 접근은 안됨.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short memo-style negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 권장된다.\n"
	humanized = "# Report\n\n이 접근은 안함.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short abbreviated action negation rewrite to fail")
	}

	original = "# Report\n\n이 접근은 권장된다.\n"
	humanized = "# Report\n\n이 접근은 안해도 된다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected short contracted action negation rewrite to fail")
	}

	original = "# Report\n\n- 상위 항목\n  - 하위 항목\n"
	humanized = "# Report\n\n- 상위 항목\n- 하위 항목\n"
	if err := validateHumanizedMarkdown(original, humanized); err == nil {
		t.Fatal("expected nested list flattening to fail")
	}
}

func TestValidateHumanizedMarkdownAllowsConservativeKoreanToneEdit(t *testing.T) {
	original := "# Report\n\n이 작업은 수행되어야 한다.\n\n## Details\n\n추가 검토가 필요할 수 있다.\n"
	humanized := "# Report\n\n이 작업은 수행해야 한다.\n\n## Details\n\n추가 검토가 필요할 수 있다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		t.Fatalf("expected conservative Korean tone edit to pass, got %v", err)
	}

	original = "# Report\n\n- 이 작업은 수행되어야 한다.\n- 추가 검토가 필요할 수 있다.\n"
	humanized = "# Report\n\n- 이 작업은 수행해야 한다.\n- 추가 검토가 필요할 수 있다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		t.Fatalf("expected conservative list item tone edit to pass, got %v", err)
	}

	original = "# Report\n\n출처 참조 보존은 보고서 검증 가능성과 직접 연결된다.\n\n출처: https://example.com/report\n"
	humanized = "# Report\n\n출처 참조 보존은 보고서를 검증할 수 있게 하는 조건과 직접 연결된다.\n\n출처: https://example.com/report\n"
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		t.Fatalf("expected conservative prose edit mentioning sources to pass, got %v", err)
	}

	original = "# Report\n\n이 제안은 안전 방안을 설명한다.\n"
	humanized = "# Report\n\n이 제안은 안전한 방안을 설명한다.\n"
	if err := validateHumanizedMarkdown(original, humanized); err != nil {
		t.Fatalf("expected Korean words containing 안 to pass, got %v", err)
	}
}
