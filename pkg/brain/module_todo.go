package brain

import "sync"

type Todo struct {
	ID        int
	Title     string
	Completed bool
}

type TodoModule struct {
	todos []Todo
	mu    sync.RWMutex
}

func NewTodoModule() *TodoModule {
	return &TodoModule{
		todos: make([]Todo, 0, 128),
	}
}

func (m *TodoModule) Add(title string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := len(m.todos) + 1
	m.todos = append(m.todos, Todo{ID: id, Title: title})
	return id
}

func (m *TodoModule) Complete(id int) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, t := range m.todos {
		if t.ID == id {
			m.todos[i].Completed = true
			return true
		}
	}
	return false
}

func (m *TodoModule) List() []Todo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return append([]Todo(nil), m.todos...)
}
