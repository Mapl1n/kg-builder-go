package relation

import (
	"fmt"
	"regexp"
	"strings"

	"kg-builder-go/internal/model"

	"github.com/google/uuid"
)

// Extractor — 从文本中抽取实体间关系
// 基于规则匹配中文关系模式
type Extractor struct {
	patterns []relationPattern
}

type relationPattern struct {
	predicate  string
	subjectType string
	objectType  string
	pattern    *regexp.Regexp
}

func New() *Extractor {
	patterns := []relationPattern{
		{
			// 法定代表人: X是Y的法定代表人
			predicate: "法定代表人", subjectType: "person", objectType: "org",
			pattern: regexp.MustCompile(`([^\s，。,\.]{2,4})\s*(?:为|是|担任)\s*([^\s，。,\.]{2,30}(?:有限公司|集团|公司))的?(?:法定代表人|负责人|总经理|法人)`),
		},
		{
			// 合同签订: X与Y签订合同
			predicate: "签订", subjectType: "org", objectType: "contract",
			pattern: regexp.MustCompile(`([^\s，。,\.]{2,30}(?:有限公司|集团|公司|企业))\s*(?:与|和|同)\s*([^\s，。,\.]{2,30}(?:有限公司|集团|公司|企业)?)\s*(?:签订|签署|订立)`),
		},
		{
			// 雇佣关系
			predicate: "受雇于", subjectType: "person", objectType: "org",
			pattern: regexp.MustCompile(`([^\s，。,\.]{2,4})\s*(?:就职于|受雇于|任职于|在)\s*([^\s，。,\.]{2,30}(?:有限公司|公司|集团|部门))`),
		},
		{
			// 金额关联: X合同金额为Y
			predicate: "金额", subjectType: "contract", objectType: "money",
			pattern: regexp.MustCompile(`((?:合同|协议|约定))(?:金额|总价|价款|价款总额)\s*(?:为|是)?\s*((?:人民币|￥|¥)?\d+(?:,\d{3})*(?:\.\d{1,2})?\s*(?:元|万元|美元))`),
		},
		{
			// 地点关联
			predicate: "位于", subjectType: "org", objectType: "location",
			pattern: regexp.MustCompile(`([^\s，。,\.]{2,30}(?:有限公司|公司|企业))\s*(?:位于|地址在|住址在)\s*([^\s，。,\.]{3,50}(?:省|市|区|县|路|街|号))`),
		},
	}

	return &Extractor{patterns: patterns}
}

// Extract 从文本中抽取关系
func (e *Extractor) Extract(docID string, entities []model.Entity, text string) []model.Relation {
	var relations []model.Relation

	// 先建立实体名称索引
	entityByName := make(map[string][]model.Entity)
	for _, ent := range entities {
		entityByName[ent.Type] = append(entityByName[ent.Type], ent)
	}

	for _, pp := range e.patterns {
		for _, m := range pp.pattern.FindAllStringSubmatch(text, -1) {
			if len(m) < 2 {
				continue
			}

			subName := cleanEntityName(m[1])
			objName := cleanEntityName(m[len(m)-1])

			// 查找匹配的实体
			subID := findEntity(entityByName[pp.subjectType], subName)
			objID := findEntity(entityByName[pp.objectType], objName)

			if subID != "" && objID != "" && subID != objID {
				relations = append(relations, model.Relation{
					ID:        uuid.New().String(),
					Subject:   subID,
					Predicate: pp.predicate,
					Object:    objID,
					DocID:     docID,
					Evidence:  m[0],
				})
			}
		}
	}

	return relations
}

func findEntity(entities []model.Entity, name string) string {
	for _, e := range entities {
		if strings.Contains(e.Name, name) || strings.Contains(name, e.Name) {
			return e.ID
		}
	}
	return ""
}

func cleanEntityName(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimRight(s, "，,。.！!；;：:")
	return s
}

func init() { _ = fmt.Sprintf }
