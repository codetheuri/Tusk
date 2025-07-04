package model

type Todo struct {
	ID  int `json:"id"`
	Title string `json:"title" validate:"required,min=3"`
	Completed  bool  `json:"completed"`
}