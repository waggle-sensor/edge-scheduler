package datatype

import "sync"

type Queue struct {
	mu       sync.Mutex
	entities []*Plugin
	index    int
}

func (q *Queue) ResetIter() {
	q.mu.Lock()
	q.index = 0
	q.mu.Unlock()
}

func (q *Queue) More() bool {
	return q.index < len(q.entities)
}

func (q *Queue) Next() *Plugin {
	q.mu.Lock()
	if q.index > len(q.entities) {
		return nil
	}
	p := q.entities[q.index]
	q.index += 1
	q.mu.Unlock()
	return p
}

func (q *Queue) GetPluginNames() (list []string) {
	q.ResetIter()
	for q.More() {
		plugin := q.Next()
		list = append(list, plugin.Name)
	}
	return
}

func (q *Queue) GetGoalIDs() (list map[string]bool) {
	list = make(map[string]bool)
	q.ResetIter()
	for q.More() {
		plugin := q.Next()
		list[plugin.GoalID] = true
	}
	return
}

func (q *Queue) Length() int {
	return len(q.entities)
}

func (q *Queue) Push(p *Plugin) {
	q.mu.Lock()
	q.entities = append(q.entities, p)
	q.index += 1
	q.mu.Unlock()
}

func (q *Queue) Pop(p *Plugin) *Plugin {
	q.mu.Lock()
	var found *Plugin
	for i, _p := range q.entities {
		if _p.Name == p.Name {
			q.entities = append(q.entities[:i], q.entities[i+1:]...)
			found = _p
			q.index -= 1
			break
		}
	}
	q.mu.Unlock()
	return found
}

func (q *Queue) PopFirst() *Plugin {
	if q.Length() > 0 {
		return q.Pop(q.entities[0])
	} else {
		return nil
	}
}
