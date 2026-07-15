package ner

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"kg-builder-go/internal/model"

	"github.com/google/uuid"
)

// Recognizer — 中文命名实体识别（规则引擎 + 词典）
// 不依赖 NLP 库，纯 Go 实现，精准匹配中文实体模式
type Recognizer struct {
	personPatterns  []*regexp.Regexp
	orgPatterns     []*regexp.Regexp
	contractPatterns []*regexp.Regexp
	datePattern     *regexp.Regexp
	moneyPattern    *regexp.Regexp
	idCardPattern   *regexp.Regexp
	phonePattern    *regexp.Regexp
}

func New() *Recognizer {
	return &Recognizer{
		personPatterns: compilePatterns(
			`(法定代表人|负责人|联系人|经理|董事长|总经理|股东|甲方|乙方|委托人|代理人)[：:]\s*([^\s，。,\.]{2,4})`,
			`([^\s，。,\.]{2,4})\s*(先生|女士|同志)`,
		),
		orgPatterns: compilePatterns(
			`([^\s，。,\.]{2,30}(?:有限公司|股份有限公司|集团|有限责任|合伙企业|事务所|学校|医院|银行))`,
			`(甲方|乙方)[：:]?\s*([^\s，。,\.]{2,30})`,
		),
		contractPatterns: compilePatterns(
			`(《[^》]+》)`,
			`([A-Z]{2,4}-\d{4}-\d+)`,
			`(合同编号|协议编号|文件编号)[：:]\s*([A-Za-z0-9-]+)`,
		),
		datePattern:  regexp.MustCompile(`(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日号]?)`),
		moneyPattern: regexp.MustCompile(`(?:人民币|￥|¥|USD|美元)?\s*(\d+(?:,\d{3})*(?:\.\d{1,2})?\s*(?:元|万元|美元|USD))`),
		idCardPattern: regexp.MustCompile(`(\d{17}[\dXx])`),
		phonePattern: regexp.MustCompile(`(1[3-9]\d{9})`),
	}
}

func compilePatterns(patterns ...string) []*regexp.Regexp {
	var r []*regexp.Regexp
	for _, p := range patterns {
		r = append(r, regexp.MustCompile(p))
	}
	return r
}

// Extract ★ 从文本中提取所有实体
func (r *Recognizer) Extract(docID, docName, text string) []model.Entity {
	var entities []model.Entity
	seen := make(map[string]bool)

	extract := func(t, name string, metadata map[string]string) {
		key := fmt.Sprintf("%s:%s:%s", t, name, docID)
		if seen[key] {
			return
		}
		seen[key] = true
		entities = append(entities, model.Entity{
			ID:        uuid.New().String(),
			Name:      name,
			Type:      t,
			DocID:     docID,
			DocName:   docName,
			Metadata:  metadata,
			CreatedAt: time.Now(),
		})
	}

	// 人物提取
	for _, p := range r.personPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanEntity(m[j]); name != "" && len([]rune(name)) >= 2 {
					extract("person", name, nil)
				}
			}
		}
	}

	// 机构提取
	for _, p := range r.orgPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanEntity(m[j]); name != "" && len([]rune(name)) >= 2 {
					extract("org", name, nil)
				}
			}
		}
	}

	// 合同编号
	for _, p := range r.contractPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanEntity(m[j]); name != "" {
					extract("contract", name, nil)
				}
			}
		}
	}

	// 日期
	for _, m := range r.datePattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			extract("date", m[1], nil)
		}
	}

	// 金额
	for _, m := range r.moneyPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			extract("money", m[1], map[string]string{"value": m[1]})
		}
	}

	// 证号+手机号
	for _, m := range r.idCardPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			extract("person", "身份证-"+m[1][:3]+"****", nil)
		}
	}
	for _, m := range r.phonePattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			extract("person", "电话-"+m[1], nil)
		}
	}

	return entities
}

func cleanEntity(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "，,。.！!；;：:")
	if len([]rune(s)) < 2 {
		return ""
	}
	return s
}
