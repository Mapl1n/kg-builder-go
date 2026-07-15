package ner

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"kg-builder-go/internal/model"

	"github.com/google/uuid"
)

type Recognizer struct {
	personPatterns   []*regexp.Regexp
	orgPatterns      []*regexp.Regexp
	contractPatterns []*regexp.Regexp
	datePattern      *regexp.Regexp
	moneyPattern     *regexp.Regexp
	idCardPattern    *regexp.Regexp
	phonePattern     *regexp.Regexp
}

func New() *Recognizer {
	return &Recognizer{
		personPatterns: compilePatterns(
			`(法定代表人|负责人|联系人|经理|董事长|总经理|股东|委托人|代理人)[：:为是任]*\s*([^\s，。,\.]{2,4})`,
			`([^\s，。,\.]{2,4})\s*(先生|女士|同志)`,
			`([^\s，。,\.]{2,4})\s*(为|是)\s*法定代表人`,
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
		datePattern:   regexp.MustCompile(`(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日号]?)`),
		moneyPattern:  regexp.MustCompile(`(?:人民币|￥|¥|USD|美元)?\s*(\d+(?:,\d{3})*(?:\.\d{1,2})?\s*(?:元|万元|美元|USD))`),
		idCardPattern: regexp.MustCompile(`(\d{17}[\dXx])`),
		phonePattern:  regexp.MustCompile(`(1[3-9]\d{9})`),
	}
}

func compilePatterns(patterns ...string) []*regexp.Regexp {
	r := make([]*regexp.Regexp, len(patterns))
	for i, p := range patterns {
		r[i] = regexp.MustCompile(p)
	}
	return r
}

func (r *Recognizer) Extract(docID, docName, text string) []model.Entity {
	var entities []model.Entity
	seen := make(map[string]bool)

	add := func(t, name string, meta map[string]string) bool {
		if isStopWord(name) {
			return false
		}
		// org-specific validation
		if t == "org" && !strings.Contains(name, "公司") && !strings.Contains(name, "集团") &&
			!strings.Contains(name, "企业") && !strings.Contains(name, "银行") &&
			!strings.Contains(name, "学校") && !strings.Contains(name, "医院") {
			return false
		}
		key := t + ":" + name + ":" + docID
		if seen[key] {
			return false
		}
		seen[key] = true
		entities = append(entities, model.Entity{
			ID: uuid.New().String(), Name: name, Type: t,
			DocID: docID, DocName: docName, Metadata: meta, CreatedAt: time.Now(),
		})
		return true
	}

	// Person
	for _, p := range r.personPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanName(m[j]); len([]rune(name)) >= 2 {
					add("person", name, nil)
				}
			}
		}
	}

	// Org
	for _, p := range r.orgPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanName(m[j]); len([]rune(name)) >= 2 {
					add("org", name, nil)
				}
			}
		}
	}

	// Contract
	for _, p := range r.contractPatterns {
		for _, m := range p.FindAllStringSubmatch(text, -1) {
			for j := 1; j < len(m); j++ {
				if name := cleanName(m[j]); name != "" {
					add("contract", name, nil)
				}
			}
		}
	}

	// Date
	for _, m := range r.datePattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("date", m[1], nil)
		}
	}

	// Money
	for _, m := range r.moneyPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("money", m[1], map[string]string{"value": m[1]})
		}
	}

	// ID Card
	for _, m := range r.idCardPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("person", "SFZ-"+m[1][:3]+"***", nil)
		}
	}

	// Phone
	for _, m := range r.phonePattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("person", "TEL-"+m[1], nil)
		}
	}

	return entities
}

var stopWords = map[string]bool{
	"法定代表人": true, "负责人": true, "联系人": true, "经理": true, "董事长": true, "总经理": true,
	"股东": true, "委托人": true, "代理人": true, "先生": true, "女士": true, "同志": true,
	"甲方": true, "乙方": true, "合同编号": true, "协议编号": true,
}

func isStopWord(s string) bool { return stopWords[s] }

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "，,。.！!；;：:")
	s = strings.TrimLeft(s, "，,。.！!；;：:为是任与被在与同和")
	// Remove known prefix patterns like "乙方为"
	for _, prefix := range []string{"甲方为", "乙方为", "甲方是", "乙方是"} {
		if strings.HasPrefix(s, prefix) {
			s = strings.TrimPrefix(s, prefix)
			break
		}
	}
	if len([]rune(s)) < 2 {
		return ""
	}
	return s
}

func init() { _ = fmt.Sprintf }
