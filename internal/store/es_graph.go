package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"kg-builder-go/internal/model"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
)

// GraphStore — ES 作为图存储
// 使用 entity 索引 + relation 索引 + 跨索引查询
type GraphStore struct {
	es          *elasticsearch.Client
	entityIndex string
	relIndex    string
}

func NewGraphStore(es *elasticsearch.Client, entityIndex string) *GraphStore {
	return &GraphStore{
		es:          es,
		entityIndex: entityIndex + "_entities",
		relIndex:    entityIndex + "_relations",
	}
}

// Init 创建索引
func (s *GraphStore) Init() error {
	for _, idx := range []string{s.entityIndex, s.relIndex} {
		req := esapi.IndicesCreateRequest{Index: idx}
		resp, _ := req.Do(context.Background(), s.es)
		if resp != nil {
			resp.Body.Close()
		}
	}
	log.Printf("[GRAPH] indices ready: %s, %s", s.entityIndex, s.relIndex)
	return nil
}

// SaveEntity 保存实体
func (s *GraphStore) SaveEntity(entity model.Entity) error {
	body, _ := json.Marshal(entity)
	req := esapi.IndexRequest{
		Index:      s.entityIndex,
		DocumentID: entity.ID,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}
	resp, err := req.Do(context.Background(), s.es)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SaveRelation 保存关系
func (s *GraphStore) SaveRelation(rel model.Relation) error {
	body, _ := json.Marshal(rel)
	req := esapi.IndexRequest{
		Index:      s.relIndex,
		DocumentID: rel.ID,
		Body:       bytes.NewReader(body),
		Refresh:    "true",
	}
	resp, err := req.Do(context.Background(), s.es)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

// SearchEntity ★ 按名称搜索实体（模糊匹配）
func (s *GraphStore) SearchEntity(name string, entityType string, docID string) ([]model.Entity, error) {
	must := []interface{}{}
	if name != "" {
		must = append(must, map[string]interface{}{
			"match": map[string]interface{}{"name": map[string]interface{}{"query": name, "fuzziness": "AUTO"}},
		})
	}
	if entityType != "" {
		must = append(must, map[string]interface{}{"term": map[string]string{"type": entityType}})
	}
	if docID != "" {
		must = append(must, map[string]interface{}{"term": map[string]string{"doc_id": docID}})
	}

	query := map[string]interface{}{"query": map[string]interface{}{"bool": map[string]interface{}{"must": must}}, "size": 50}
	body, _ := json.Marshal(query)

	resp, err := s.es.Search(
		s.es.Search.WithContext(context.Background()),
		s.es.Search.WithIndex(s.entityIndex),
		s.es.Search.WithBody(bytes.NewReader(body)),
	)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Hits struct{ Hits []struct{ Source model.Entity `json:"_source"` } } `json:"hits"`
	}
	json.NewDecoder(resp.Body).Decode(&result)

	entities := make([]model.Entity, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		entities = append(entities, h.Source)
	}
	return entities, nil
}

// GetRelationsByEntity 获取某实体的所有关系
func (s *GraphStore) GetRelationsByEntity(entityID string) ([]model.Relation, error) {
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"should": []interface{}{
					map[string]interface{}{"term": map[string]string{"subject": entityID}},
					map[string]interface{}{"term": map[string]string{"object": entityID}},
				},
			},
		},
		"size": 100,
	}
	body, _ := json.Marshal(query)

	resp, _ := s.es.Search(
		s.es.Search.WithContext(context.Background()),
		s.es.Search.WithIndex(s.relIndex),
		s.es.Search.WithBody(bytes.NewReader(body)),
	)
	defer resp.Body.Close()

	var result struct{ Hits struct{ Hits []struct{ Source model.Relation `json:"_source"` } } `json:"hits"` }
	json.NewDecoder(resp.Body).Decode(&result)

	relations := make([]model.Relation, 0, len(result.Hits.Hits))
	for _, h := range result.Hits.Hits {
		relations = append(relations, h.Source)
	}
	return relations, nil
}

// GetGraphData ★ 获取图谱可视化数据
func (s *GraphStore) GetGraphData(entityID string, depth int) (*model.GraphData, error) {
	if depth <= 0 {
		depth = 2
	}

	// BFS from entityID
	visited := make(map[string]bool)
	var nodes []model.GraphNode
	var edges []model.GraphEdge
	entityMap := make(map[string]model.Entity)

	queue := []string{entityID}
	visited[entityID] = true

	for d := 0; d < depth && len(queue) > 0; d++ {
		var next []string
		for _, eid := range queue {
			// Fetch entity
			ent, err := s.getEntity(eid)
			if err == nil && ent != nil {
				entityMap[eid] = *ent
				color := "#3b82f6"
				switch ent.Type {
				case "person": color = "#3b82f6"
				case "org": color = "#22c55e"
				case "contract": color = "#f59e0b"
				case "date": color = "#8b5cf6"
				case "money": color = "#ef4444"
				case "location": color = "#06b6d4"
				}
				nodes = append(nodes, model.GraphNode{
					ID: ent.ID, Name: ent.Name, Type: ent.Type,
					Group: entityTypeGroup[ent.Type],
					DocID: ent.DocID, Size: 30,
				})
				_ = color
			}

			// Get relations
			rels, _ := s.GetRelationsByEntity(eid)
			for _, rel := range rels {
				edges = append(edges, model.GraphEdge{
					Source: rel.Subject, Target: rel.Object, Label: rel.Predicate,
				})

				other := rel.Subject
				if other == eid {
					other = rel.Object
				}
				if !visited[other] {
					visited[other] = true
					next = append(next, other)
				}
			}
		}
		queue = next
	}

	return &model.GraphData{Nodes: nodes, Edges: edges}, nil
}

func (s *GraphStore) getEntity(id string) (*model.Entity, error) {
	resp, err := s.es.Get(s.entityIndex, id)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.IsError() {
		return nil, fmt.Errorf("not found")
	}
	var ent struct{ Source model.Entity `json:"_source"` }
	json.NewDecoder(resp.Body).Decode(&ent)
	return &ent.Source, nil
}

var entityTypeGroup = map[string]int{"person":0,"org":1,"contract":2,"date":3,"money":4,"location":5}

func init() { _ = strings.TrimSpace }
