package services

import (
	"strings"
)

func normalizeSceneKey(location, time string) string {
	loc := strings.TrimSpace(location)
	tm := strings.TrimSpace(time)
	loc = strings.TrimPrefix(loc, "动漫风格")
	loc = strings.TrimPrefix(loc, "写实风格")
	loc = strings.Trim(loc, " ，,。；;\t")
	// Keep primary location chunk.
	if idx := strings.Index(loc, "·"); idx > 0 {
		loc = strings.TrimSpace(loc[:idx])
	}
	if idx := strings.Index(loc, "，"); idx > 0 {
		loc = strings.TrimSpace(loc[:idx])
	}
	if idx := strings.Index(loc, ","); idx > 0 {
		loc = strings.TrimSpace(loc[:idx])
	}
	if idx := strings.Index(tm, "·"); idx > 0 {
		tm = strings.TrimSpace(tm[:idx])
	}
	if idx := strings.Index(tm, "，"); idx > 0 {
		tm = strings.TrimSpace(tm[:idx])
	}
	if idx := strings.Index(tm, ","); idx > 0 {
		tm = strings.TrimSpace(tm[:idx])
	}
	return loc + "|" + tm
}

func sceneMatchScore(sbLocation, sbTime, sceneLocation, sceneTime string) int {
	locScore := 0
	if sbLocation != "" && sceneLocation != "" {
		if strings.Contains(sbLocation, sceneLocation) || strings.Contains(sceneLocation, sbLocation) {
			locScore += 3
		}
		// Loose keyword matching for common place descriptors.
		placeTokens := []string{"室内", "户外", "客厅", "卧室", "厨房", "餐厅", "走廊", "电梯", "楼道", "办公室", "公司", "医院", "学校", "街道", "小区", "公园", "车内", "天台"}
		for _, tok := range placeTokens {
			if strings.Contains(sbLocation, tok) && strings.Contains(sceneLocation, tok) {
				locScore += 1
			}
		}
	}
	timeScore := 0
	if sbTime != "" && sceneTime != "" {
		if strings.Contains(sbTime, sceneTime) || strings.Contains(sceneTime, sbTime) {
			timeScore += 2
		}
		timeTokens := []string{"清晨", "早晨", "上午", "中午", "下午", "傍晚", "黄昏", "夜晚", "深夜", "凌晨", "雨", "雪", "雾"}
		for _, tok := range timeTokens {
			if strings.Contains(sbTime, tok) && strings.Contains(sceneTime, tok) {
				timeScore += 1
			}
		}
	}
	return locScore + timeScore
}

func buildCharacterAliases(name string) []string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil
	}
	aliases := []string{trimmed}
	// Handle Chinese full-width parentheses aliases: "苏遥遥（遥遥）"
	if open := strings.Index(trimmed, "（"); open >= 0 {
		if close := strings.Index(trimmed[open+len("（"):], "）"); close >= 0 {
			main := strings.TrimSpace(trimmed[:open])
			alt := strings.TrimSpace(trimmed[open+len("（") : open+len("（")+close])
			if main != "" {
				aliases = append(aliases, main)
			}
			if alt != "" {
				aliases = append(aliases, alt)
			}
		}
	}
	// Deduplicate
	seen := make(map[string]struct{}, len(aliases))
	out := make([]string, 0, len(aliases))
	for _, a := range aliases {
		a = strings.TrimSpace(a)
		if a == "" {
			continue
		}
		if _, ok := seen[a]; ok {
			continue
		}
		seen[a] = struct{}{}
		out = append(out, a)
	}
	return out
}

func inferCharacterIDsFromText(text string, aliasToID map[string]uint) []uint {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	ids := make([]uint, 0, 4)
	seen := make(map[uint]struct{}, 4)
	for alias, id := range aliasToID {
		// Avoid too-short aliases to reduce false positives.
		if len([]rune(alias)) < 2 {
			continue
		}
		if strings.Contains(trimmed, alias) {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}
