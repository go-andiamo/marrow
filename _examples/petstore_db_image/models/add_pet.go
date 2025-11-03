package models

type AddPet struct {
	Name     string `json:"name"`
	Dob      string `json:"dob"`
	Category struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"category"`
}
