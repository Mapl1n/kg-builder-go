package ner

import (
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
			// Chinese
			`(法定代表人|负责人|联系人|经理|董事长|总经理|股东|委托人|代理人)[：:为是任]*\s*([^\s，。,\.]{2,4})`,
			`([^\s，。,\.]{2,4})\s*(先生|女士|同志)`,
			`([^\s，。,\.]{2,4})\s*(为|是)\s*法定代表人`,
			// English — with context (title prefix) or standalone names
			`(?:legal rep|representative|contact|manager|CEO|director|Mr\.?|Ms\.?)[：: ]*([A-Z][a-z]+ [A-Z][a-z]+)`,
			`([A-Z][a-z]+ [A-Z][a-z]+)`,
		),
		orgPatterns: compilePatterns(
			// Chinese
			`([^\s，。,\.]{2,30}(?:有限公司|股份有限公司|集团|有限责任|合伙企业|事务所|学校|医院|银行))`,
			`(甲方|乙方)[：:]?\s*([^\s，。,\.]{2,30})`,
			// English
			`([A-Z][a-zA-Z]+ (?:Tech|Technology|Software|Inc|Ltd|LLC|Corp|Co)\b[^\s，。,]*)`,
			`([A-Z][a-z]+ [A-Z][a-z]+ (?:Ltd|Inc|LLC|Corp|Co))`,
		),
		contractPatterns: compilePatterns(
			`(《[^》]+》)`,
			`([A-Z]{2,4}-\d{4}-\d+)`,
			`(合同编号|协议编号|文件编号)[：:]\s*([A-Za-z0-9-]+)`,
		),
		datePattern:   regexp.MustCompile(`(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日号]?)`),
		moneyPattern:  regexp.MustCompile(`(?:人民币|RMB|￥|¥|USD|美元)?\s*(\d[\d,.]*\s*(?:万|元|万元|美元|USD|M|K|RMB))`),
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
		if t == "org" && !strings.Contains(name, "公司") && !strings.Contains(name, "集团") &&
			!strings.Contains(name, "企业") && !strings.Contains(name, "银行") &&
			!strings.Contains(name, "学校") && !strings.Contains(name, "医院") &&
			!strings.Contains(name, "Ltd") && !strings.Contains(name, "Inc") &&
			!strings.Contains(name, "LLC") && !strings.Contains(name, "Corp") &&
			!strings.Contains(name, "Tech") && !strings.Contains(name, "Co") {
			return false
		}
		// Exclude person names that are actually locations or orgs
		if t == "person" && (strings.HasSuffix(name, " Tech") || strings.HasSuffix(name, " Ltd") ||
			strings.HasSuffix(name, " Inc") || strings.HasSuffix(name, " Co") ||
			strings.HasSuffix(name, " LLC") || strings.HasSuffix(name, " Corp") ||
			strings.Contains(name, " Tech ") || strings.Contains(name, " Ltd ") ||
			strings.HasSuffix(name, " District") || strings.HasSuffix(name, " Road") ||
			strings.HasSuffix(name, " Street") || strings.HasSuffix(name, " Avenue") ||
			strings.HasSuffix(name, " City") || strings.HasSuffix(name, " Province")) {
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

	matchPatterns := func(patterns []*regexp.Regexp, etype string) {
		for _, p := range patterns {
			for _, m := range p.FindAllStringSubmatch(text, -1) {
				for j := 1; j < len(m); j++ {
					if name := cleanName(m[j]); len([]rune(name)) >= 2 {
						if etype == "contract" {
							add(etype, name, nil)
						} else {
							add(etype, name, nil)
						}
					}
				}
			}
		}
	}

	matchPatterns(r.personPatterns, "person")
	matchPatterns(r.orgPatterns, "org")
	matchPatterns(r.contractPatterns, "contract")

	for _, m := range r.datePattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("date", m[1], nil)
		}
	}
	for _, m := range r.moneyPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("money", m[1], map[string]string{"value": m[1]})
		}
	}
	for _, m := range r.idCardPattern.FindAllStringSubmatch(text, -1) {
		if len(m) > 1 {
			add("person", "SFZ-"+m[1][:3]+"***", nil)
		}
	}
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
