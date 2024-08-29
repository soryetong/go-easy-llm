package service

import (
	"regexp"
	"strings"
)

type MarkdownProcessor struct {
	inCodeBlock bool
}

func (self *MarkdownProcessor) Do(value string) string {
	lines := strings.Split(value, "\n")
	var result []string

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		if self.inCodeBlock {
			if strings.HasPrefix(trimmedLine, "```") {
				self.inCodeBlock = false
			}
			continue
		}

		if strings.HasPrefix(trimmedLine, "```") {
			self.inCodeBlock = true
			continue
		}

		if self.isMarkdown(trimmedLine) {
			continue
		}

		processedLine := self.removeLineCode(trimmedLine)
		if processedLine != "" {
			result = append(result, processedLine)
		}
	}

	return strings.Join(result, "\n")
}

// 判断是否为markdown语法
func (self *MarkdownProcessor) isMarkdown(value string) bool {
	for _, pattern := range self.getPatterns() {
		re := regexp.MustCompile(pattern)
		if re.MatchString(value) {
			return true
		}
	}

	return false
}

// 去除行内代码,包括markdown内容
func (self *MarkdownProcessor) removeLineCode(value string) string {
	re := regexp.MustCompile("`[^`]*`")
	return strings.TrimSpace(re.ReplaceAllString(value, ""))
}

// 正则匹配的规则
func (self *MarkdownProcessor) getPatterns() []string {
	return []string{
		"^#{1,6}\\s",           // 标题
		"^\\s*[-+*]\\s+",       // 无序列表
		"^\\d+\\.\\s+",         // 有序列表
		"^>\\s+",               // 引用
		"^!\\[.*?\\]\\(.*?\\)", // 图片
		"^\\[.*?\\]\\(.*?\\)",  // 链接
	}
}
