package util

import (
	"goswitch/internal/config"
	"regexp"
)

// ApplyRegexMapper 应用正则映射规则
func ApplyRegexMapper(name string, mappers []config.RegexMapper) string {
	result := name
	for _, m := range mappers {
		re, err := regexp.Compile(m.FromPattern)
		if err != nil {
			continue
		}
		result = re.ReplaceAllString(result, m.ToValue)
	}
	return result
}
