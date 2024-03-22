package datatype

import "sync"

type Queue struct {
	mu       sync.Mutex
	entities []*PluginRuntime
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

func (q *Queue) Next() *PluginRuntime {
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
		pr := q.Next()
		list = append(list, pr.Plugin.Name)
	}
	return
}

func (q *Queue) GetGoalIDs() (list map[string]bool) {
	list = make(map[string]bool)
	q.ResetIter()
	for q.More() {
		pr := q.Next()
		list[pr.Plugin.GoalID] = true
	}
	return
}

func (q *Queue) Length() int {
	return len(q.entities)
}

func (q *Queue) Push(p *PluginRuntime) {
	q.mu.Lock()
	q.entities = append(q.entities, p)
	q.index += 1
	q.mu.Unlock()
}

func (q *Queue) IsExist(pr *PluginRuntime) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, _pr := range q.entities {
		if pr.Equal(_pr) {
			return true
		}
	}
	return false
}

func (q *Queue) Pop(pr *PluginRuntime) *PluginRuntime {
	q.mu.Lock()
	var found *PluginRuntime
	for i, _pr := range q.entities {
		if _pr.Plugin.Name == pr.Plugin.Name {
			q.entities = append(q.entities[:i], q.entities[i+1:]...)
			found = _pr
			q.index -= 1
			break
		}
	}
	q.mu.Unlock()
	return found
}

func (q *Queue) PopFirst() *PluginRuntime {
	if q.Length() > 0 {
		return q.Pop(q.entities[0])
	} else {
		return nil
	}
}
