package main

import ()

type Todo struct {
	Id        int    `json:"-"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
	Order     int    `json:"order"`
	Url       string `json:"url"`
}
