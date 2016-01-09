package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var TodoSvc *MockTodoService

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		log.Fatal("$PORT must be set")
	}

	TodoSvc = NewMockTodoService()
	mux := http.NewServeMux()

	mux.Handle("/todos", commonHandlers(todoHandler))
	mux.Handle("/todos/", commonHandlers(todoHandler))

	log.Fatal(http.ListenAndServe(":"+port, mux))
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
