package relation

import (
	"regexp"
	"strings"

	"kg-builder-go/internal/model"

	"github.com/google/uuid"
)

type Extractor struct {
	patterns []relationPattern
}

type relationPattern struct {
	predicate   string
	subjectType string
	objectType  string
	pattern     *regexp.Regexp
}

func New() *Extractor {
	return &Extractor{patterns: []relationPattern{
		{predicate: "签约", subjectType: "org", objectType: "contract",
			pattern: regexp.MustCompile(`(甲方|乙方)[：:是为]*\s*([^\s，。,\.]{2,30}(?:公司|有限|集团)?)\s*[,，]*.*?(?:签订|签署|订立).*?([A-Z]{2,4}-\d{4}-\d+)`)},
		{predicate: "涉及金额", subjectType: "contract", objectType: "money",
			pattern: regexp.MustCompile(`(?:合同|协议)(?:总)?金额[：:是为]*\s*(人民币?\s*\d+万?\s*(?:元|美元)?)`)},
		{predicate: "签署日期", subjectType: "contract", objectType: "date",
			pattern: regexp.MustCompile(`(?:签订|签署)(?:日期|时间|于)?[：:是为]*\s*(\d{4}[-/年]\d{1,2}[-/月]\d{1,2}[日号]?)`)},
	}}
}

// Extract 从文本中抽取关系
func (e *Extractor) Extract(docID string, entities []model.Entity, text string) []model.Relation {
	var relations []model.Relation
	seen := make(map[string]bool)

	entityByName := make(map[string][]model.Entity)
	entityByType := make(map[string][]model.Entity)
	for _, ent := range entities {
		entityByName[ent.Name] = append(entityByName[ent.Name], ent)
		entityByType[ent.Type] = append(entityByType[ent.Type], ent)
	}

	for _, pp := range e.patterns {
		for _, m := range pp.pattern.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}
			subID := findEntityInText(entityByType[pp.subjectType], m)
			objID := findEntityInText(entityByType[pp.objectType], m)

			if subID == "" || objID == "" || subID == objID {
				continue
			}
			key := subID + "|" + pp.predicate + "|" + objID
			if seen[key] {
				continue
			}
			seen[key] = true
			relations = append(relations, model.Relation{
				ID: uuid.New().String(), Subject: subID, Predicate: pp.predicate,
				Object: objID, DocID: docID, Evidence: m[0],
			})
		}
	}

	// Fallback: co-occurrence relations
	// Person linked to the only org in the same doc
	if len(relations) == 0 && len(entityByType["person"]) > 0 && len(entityByType["org"]) > 0 {
		for _, p := range entityByType["person"] {
			for _, o := range entityByType["org"] {
				key := p.ID + "|" + o.ID
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, model.Relation{
					ID: uuid.New().String(), Subject: p.ID, Predicate: "关联",
					Object: o.ID, DocID: docID, Evidence: p.Name + " - " + o.Name,
				})
			}
		}
	}

	// Contract linked to money entity
	if len(entityByType["contract"]) > 0 && len(entityByType["money"]) > 0 {
		for _, c := range entityByType["contract"] {
			for _, m := range entityByType["money"] {
				key := c.ID + "|金额|" + m.ID
				if seen[key] {
					continue
				}
				seen[key] = true
				relations = append(relations, model.Relation{
					ID: uuid.New().String(), Subject: c.ID, Predicate: "金额",
					Object: m.ID, DocID: docID, Evidence: m.Name,
				})
			}
		}
	}

	return relations
}

// findEntityInText searches regex match groups for a matching entity name
func findEntityInText(entities []model.Entity, matchGroups []string) string {
	if len(entities) == 0 {
		return ""
	}
	// Try each match group as a partial name
	for j := 1; j < len(matchGroups); j++ {
		name := cleanName(matchGroups[j])
		if name == "" || len([]rune(name)) < 2 {
			continue
		}
		// Direct match
		for _, e := range entities {
			if e.Name == name {
				return e.ID
			}
		}
		// Substring match
		for _, e := range entities {
			if strings.Contains(e.Name, name) || strings.Contains(name, e.Name) {
				return e.ID
			}
		}
	}
	return ""
}

func cleanName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "，,。.！!；;：:")
	s = strings.TrimLeft(s, "，,。.！!；;：:是任为")
	return s
}
