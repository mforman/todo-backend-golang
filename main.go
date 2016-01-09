package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Todo struct {
	Id        int    `json:"-"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Order     int    `json:"order"`
	Url       string `json:"url"`
}

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

func optionsOk(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("access-control-allow-origin", "*")
		w.Header().Set("access-control-allow-methods", "GET, POST, PATCH, DELETE")
		w.Header().Set("access-control-allow-headers", "accept, content-type")
		if r.Method == "OPTIONS" {
			return // Preflight sets headers and we're done
		}
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func contentTypeJsonHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}

	return http.HandlerFunc(fn)
}

func commonHandlers(next http.HandlerFunc) http.Handler {
	return loggingHandler(contentTypeJsonHandler(optionsOk(next)))
}

var TodoSvc *MockTodoService

func main() {
	TodoSvc = NewMockTodoService()
	mux := http.NewServeMux()

	mux.Handle("/todos", commonHandlers(todoHandler))
	mux.Handle("/todos/", commonHandlers(todoHandler))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func addUrlToTodos(r *http.Request, todos ...*Todo) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseUrl := scheme + "://" + r.Host + "/todos/"

	for _, todo := range todos {
		todo.Url = baseUrl + strconv.Itoa(todo.Id)
	}
}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	key := ""
	if len(parts) > 2 {
		key = parts[2]
	}

	switch r.Method {
	case "GET":
		if len(key) == 0 {
			todos, err := TodoSvc.GetAll()
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			addUrlToTodos(r, todos...)
			json.NewEncoder(w).Encode(todos)
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			todo, err := TodoSvc.Get(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if todo == nil {
				http.NotFound(w, r)
				return
			}
			addUrlToTodos(r, todo)
			json.NewEncoder(w).Encode(todo)
		}
	case "POST":
		if len(key) > 0 {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		todo := Todo{
			Completed: false,
		}
		err := json.NewDecoder(r.Body).Decode(&todo)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		err = TodoSvc.Save(&todo)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		addUrlToTodos(r, &todo)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(todo)
	case "PATCH":
		id, err := strconv.Atoi(key)
		if err != nil {
			http.Error(w, "Invalid Id", http.StatusBadRequest)
			return
		}
		var todo Todo
		err = json.NewDecoder(r.Body).Decode(&todo)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		todo.Id = id

		err = TodoSvc.Save(&todo)
		if err != nil {
			if strings.ToLower(err.Error()) == "not found" {
				http.NotFound(w, r)
				return
			}
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		addUrlToTodos(r, &todo)
		json.NewEncoder(w).Encode(todo)
	case "DELETE":
		if len(key) == 0 {
			TodoSvc.DeleteAll()
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			err = TodoSvc.Delete(id)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
}
