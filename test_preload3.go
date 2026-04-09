package main

import (
	"fmt"
	"github.com/Astervia/wacraft-core/src/workspace/entity"
	"gorm.io/gorm/schema"
	"sync"
)

func main() {
    s, err := schema.Parse(&workspace_entity.WorkspaceMember{}, &sync.Map{}, schema.NamingStrategy{})
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    for _, rel := range s.Relationships.Relations {
        fmt.Println("Found relation:", rel.Name, rel.Type)
    }
}
