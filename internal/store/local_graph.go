package store

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"kg-builder-go/internal/model"
)

// LocalGraphStore — 内存图谱存储，替代 ES
type LocalGraphStore struct {
	mu        sync.RWMutex
	entities  map[string]model.Entity
	relations map[string][]model.Relation // entityID → relations
}

func NewLocalGraphStore() *LocalGraphStore {
	return &LocalGraphStore{
		entities:  make(map[string]model.Entity),
		relations: make(map[string][]model.Relation),
	}
}

func (s *LocalGraphStore) SaveEntity(e model.Entity) {
	s.mu.Lock()
	s.entities[e.ID] = e
	s.mu.Unlock()
}

func (s *LocalGraphStore) SaveRelation(r model.Relation) {
	s.mu.Lock()
	s.relations[r.Subject] = append(s.relations[r.Subject], r)
	s.relations[r.Object] = append(s.relations[r.Object], r)
	s.mu.Unlock()
}

func (s *LocalGraphStore) SearchEntity(name, entityType, docID string) []model.Entity {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []model.Entity
	for _, e := range s.entities {
		if name != "" && !strings.Contains(strings.ToLower(e.Name), strings.ToLower(name)) {
			continue
		}
		if entityType != "" && e.Type != entityType {
			continue
		}
		if docID != "" && e.DocID != docID {
			continue
		}
		result = append(result, e)
	}
	// Sort by name match quality
	sort.Slice(result, func(i, j int) bool {
		return len(result[i].Name) < len(result[j].Name)
	})
	return result
}

func (s *LocalGraphStore) GetRelations(entityID string) []model.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.relations[entityID]
}

func (s *LocalGraphStore) GetGraphData(entityID string, depth int) (*model.GraphData, error) {
	if depth <= 0 {
		depth = 2
	}

	visited := make(map[string]bool)
	var nodes []model.GraphNode
	var edges []model.GraphEdge
	seenEdges := make(map[string]bool)

	groupMap := map[string]int{"person": 0, "org": 1, "contract": 2, "date": 3, "money": 4, "location": 5}

	queue := []string{entityID}
	visited[entityID] = true

	for d := 0; d < depth && len(queue) > 0; d++ {
		var next []string
		for _, eid := range queue {
			s.mu.RLock()
			ent, ok := s.entities[eid]
			s.mu.RUnlock()
			if ok {
				nodes = append(nodes, model.GraphNode{
					ID: ent.ID, Name: ent.Name, Type: ent.Type,
					Group: groupMap[ent.Type], DocID: ent.DocID, Size: 20 + 10*(len(s.GetRelations(eid))),
				})
			}
			for _, rel := range s.GetRelations(eid) {
				edgeKey := fmt.Sprintf("%s-%s-%s", rel.Subject, rel.Predicate, rel.Object)
				if !seenEdges[edgeKey] {
					seenEdges[edgeKey] = true
					edges = append(edges, model.GraphEdge{
						Source: rel.Subject, Target: rel.Object, Label: rel.Predicate,
					})
				}
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

func init() { _ = strings.TrimSpace }
