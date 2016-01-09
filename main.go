package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type Todo struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Order     int    `json:"order"`
	Url       string `json:"url"`
}

type TodoService interface {
	GetAll() ([]Todo, error)
	Get(id int) (*Todo, error)
	Save(todo *Todo) error
	DeleteAll() error
	Delete(id int) error
}

type MockTodoService struct {
	nextId  int
	apiRoot string
	Todos   []*Todo
}

func NewMockTodoService(apiRoot string) *MockTodoService {
	t := new(MockTodoService)
	t.Todos = make([]*Todo, 0)
	t.nextId = 1
	t.apiRoot = apiRoot
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
	return nil, fmt.Errorf("Todo %d was not found", id)
}

func (t *MockTodoService) Save(todo *Todo) error {
	if todo.Id == 0 {
		var mutex = &sync.Mutex{}
		mutex.Lock()
		todo.Id = t.nextId
		t.nextId++
		mutex.Unlock()
		todo.Url = t.apiRoot + strconv.Itoa(todo.Id)
	}

	for i, value := range t.Todos {
		if value.Id == todo.Id {
			t.Todos[i] = todo
			return nil
		}
	}

	t.Todos = append(t.Todos, todo)
	return nil
}

func (t *MockTodoService) DeleteAll() error {
	t.Todos = make([]*Todo, 0)
	return nil
}

func (t *MockTodoService) Delete(id int) error {
	for i, value := range t.Todos {
		if value.Id == id {
			t.Todos = append(t.Todos[:i], t.Todos[i+1:]...)
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
			return
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
var RootPath = "/todos"

func main() {
	TodoSvc = NewMockTodoService("http://localhost:8080" + RootPath + "/")
	mux := http.NewServeMux()

	mux.Handle(RootPath+"/", commonHandlers(todoHandler))
	mux.Handle(RootPath, commonHandlers(todoHandler))

	log.Fatal(http.ListenAndServe(":8080", mux))
}

func todoHandler(w http.ResponseWriter, r *http.Request) {
	var key string
	if len(r.URL.Path) > len(RootPath+"/") {
		key = r.URL.Path[len(RootPath+"/"):]
	} else {
		key = ""
	}

	switch r.Method {
	case "GET":
		if len(key) == 0 {
			todos, _ := TodoSvc.GetAll()
			json.NewEncoder(w).Encode(todos)
		} else {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			todo, _ := TodoSvc.Get(id)
			json.NewEncoder(w).Encode(todo)
		}
	case "POST", "PATCH":
		todo := Todo{
			Completed: false,
		}
		err := json.NewDecoder(r.Body).Decode(&todo)
		if err != nil {
			http.Error(w, err.Error(), 422)
			return
		}
		if todo.Id != 0 {
			id, err := strconv.Atoi(key)
			if err != nil {
				http.Error(w, "Invalid Id", http.StatusBadRequest)
				return
			}
			if id != todo.Id {
				http.Error(w, "Id in body must match URL parameter", http.StatusBadRequest)
				return
			}
		}
		TodoSvc.Save(&todo)
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
			TodoSvc.Delete(id)
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

}
