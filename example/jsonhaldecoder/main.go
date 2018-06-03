package main

import (
	"encoding/json"
	"fmt"

	"github.com/r0wbrt/riot/pkg/jsonhal"
)

type data struct {
	Links *jsonhal.Collection
}

func main() {
	s1 := data{}
	s1.Links = jsonhal.NewCollection()
	s1.Links.Values["self"] = []*jsonhal.CollectionValue{jsonhal.CreateLink("link1"), jsonhal.CreateLink("link2")}

	out, err := json.Marshal(s1)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(out))

	s2 := &data{}
	err = json.Unmarshal(out, s2)
	if err != nil {
		panic(err)
	}

	fmt.Print(s2.Links.Values["self"][0])
	fmt.Print(s2.Links.Values["self"][1])
}
