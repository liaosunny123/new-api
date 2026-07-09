package ratio_setting

import "testing"

// TestGpt56CompletionRatio 验证 GPT-5.6 三个层级补全倍率默认 6（而非旧的 8）且不锁定（可在界面编辑）。
// 依据发布定价：输出均为输入的 6 倍（Sol 30/5、Terra 15/2.5、Luna 6/1）。
func TestGpt56CompletionRatio(t *testing.T) {
	for _, name := range []string{"gpt-5.6-sol", "gpt-5.6-terra", "gpt-5.6-luna", "gpt-5.6"} {
		info := GetCompletionRatioInfo(name)
		if info.Ratio != 6 {
			t.Errorf("%s completion ratio = %v, want 6", name, info.Ratio)
		}
		if info.Locked {
			t.Errorf("%s expected locked=false (可编辑)", name)
		}
	}

	// 回归保护：其它未特化的 gpt-5 仍保持旧的固定倍率 8 且锁定
	if info := GetCompletionRatioInfo("gpt-5.7-foo"); info.Ratio != 8 || !info.Locked {
		t.Errorf("gpt-5.7-foo = %+v, want {8 true}", info)
	}
}
