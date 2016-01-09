package main

import (
	"fmt"
	"sync"
)

// Define an interface for the data methods to support different storage types
type TodoService interface {
	GetAll() ([]Todo, error)
	Get(id int) (*Todo, error)
	Save(todo *Todo) error
	DeleteAll() error
	Delete(id int) error
}

// MockTodoService uses a concurrent array for basic testing
type MockTodoService struct {
	m      sync.Mutex
	nextId int
	Todos  []*Todo
}

func NewMockTodoService() *MockTodoService {
	t := new(MockTodoService)
	t.m.Lock()
	t.Todos = make([]*Todo, 0)
	t.nextId = 1 // Start at 1 so we can distinguish from unspecified (0)
	t.m.Unlock()
	return t
}

func (t *MockTodoService) GetAll() ([]*Todo, error) {
	return t.Todos, nil
}

func (t *MockTodoService) Get(id int) (*Todo, error) {
	for _, value := range t.Todos {
		if value.Id == id {
			return value, nil
		}
	}
	return nil, nil
}

func (t *MockTodoService) Save(todo *Todo) error {
	if todo.Id == 0 { // Insert
		t.m.Lock()
		todo.Id = t.nextId
		t.nextId++
		t.m.Unlock()

		t.m.Lock()
		t.Todos = append(t.Todos, todo)
		t.m.Unlock()
		return nil
	}

	// Update existing
	for i, value := range t.Todos {
		if value.Id == todo.Id {
			t.Todos[i] = todo
			return nil
		}
	}

	return fmt.Errorf("Not Found")
}

func (t *MockTodoService) DeleteAll() error {
	t.m.Lock()
	t.Todos = make([]*Todo, 0)
	t.m.Unlock()
	return nil
}

func (t *MockTodoService) Delete(id int) error {
	for i, value := range t.Todos {
		if value.Id == id {
			t.m.Lock()
			t.Todos = append(t.Todos[:i], t.Todos[i+1:]...)
			t.m.Unlock()
			return nil
		}
	}
	return nil
}
