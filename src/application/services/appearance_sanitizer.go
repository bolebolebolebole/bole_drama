package services

import (
	"strings"
	"unicode/utf8"
)

// sanitizeCharacterAppearance tries to strip non-static, narrative details from an AI-generated appearance.
// It is intentionally conservative: keep stable, visible traits; drop emotions, plot, and time-based changes.
func sanitizeCharacterAppearance(input string) string {
	text := strings.TrimSpace(input)
	if text == "" {
		return input
	}

	// Normalize separators to simplify splitting.
	replacer := strings.NewReplacer(
		"\r\n", "，",
		"\n", "，",
		"\r", "，",
		"。", "，",
		"！", "，",
		"？", "，",
		"；", "，",
		";", "，",
		"、", "，",
		"\t", "，",
	)
	normalized := strings.TrimSpace(replacer.Replace(text))
	parts := strings.Split(normalized, "，")

	banned := []string{
		// emotions / mental states / judgements
		"疲惫", "憔悴", "隐忍", "绝望", "崩溃", "痛苦", "委屈", "无奈", "恐惧", "焦虑", "愤怒", "幸福", "开心", "难过", "可怜", "软弱", "怯懦", "坚定", "坚韧", "冷漠", "温柔",
		// expressions / dynamic facial states
		"眼神", "目光", "神情", "表情", "眼眶", "泪", "泪痕", "通红", "颤抖", "含泪",
		// story / time / before-after changes
		"原本", "曾经", "之前", "之后", "此刻", "现在", "当下", "后来", "变故", "经历", "背叛", "觉醒", "褪去", "燃起", "绝境",
		// causal narration markers
		"因为", "由于", "所以", "因此", "导致", "让她", "使她", "使得",
		// non-visual identity / life details
		"主妇", "家庭主妇", "操持家务", "照顾", "熬夜", "蒙在鼓里",
		// camera / scene language
		"特写", "镜头", "逆光", "暖光",
		// weak narrative phrasing
		"看起来", "显得", "透着", "带着",
	}

	trimConnectors := []string{"但", "却", "然而", "不过", "同时"}

	kept := make([]string, 0, len(parts))
	for _, raw := range parts {
		clause := strings.TrimSpace(raw)
		if clause == "" {
			continue
		}

		// If the clause contains narrative connectors, keep the left side when the right side is clearly narrative.
		for _, c := range trimConnectors {
			idx := strings.Index(clause, c)
			if idx > 0 {
				right := strings.TrimSpace(clause[idx+len(c):])
				if right != "" && containsAny(right, banned) {
					clause = strings.TrimSpace(clause[:idx])
					break
				}
			}
		}

		clause = strings.Trim(clause, " ，,。；;")
		if clause == "" {
			continue
		}
		if containsAny(clause, banned) {
			continue
		}
		kept = append(kept, clause)
	}

	out := strings.Trim(strings.Join(kept, "，"), " ，")
	out = collapseRepeatedComma(out)

	// If everything was filtered out, return the original (better than empty).
	if strings.TrimSpace(out) == "" {
		return text
	}

	// Keep the appearance reasonably short to reduce narrative drift.
	const maxRunes = 220
	if utf8.RuneCountInString(out) > maxRunes {
		runes := []rune(out)
		out = strings.TrimSpace(string(runes[:maxRunes]))
		out = strings.TrimRight(out, "，")
	}

	return out
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if n == "" {
			continue
		}
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func collapseRepeatedComma(s string) string {
	for strings.Contains(s, "，，") {
		s = strings.ReplaceAll(s, "，，", "，")
	}
	return strings.Trim(s, "，")
}
