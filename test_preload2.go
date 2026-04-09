package main

import (
	"fmt"
	"github.com/Astervia/wacraft-core/src/workspace/entity"
	"gorm.io/gorm/schema"
	"sync"
)

type WorkspaceMemberWithPolicies struct {
	workspace_entity.WorkspaceMember
	Policies []workspace_entity.WorkspaceMemberPolicy `gorm:"foreignKey:WorkspaceMemberID"`
}

func main() {
    s, err := schema.Parse(&WorkspaceMemberWithPolicies{}, &sync.Map{}, schema.NamingStrategy{})
    if err != nil {
        fmt.Println("Error:", err)
        return
    }
    rel, ok := s.Relationships.Relations["Policies"]
    if !ok {
        fmt.Println("No Policies relation found")
    } else {
        fmt.Println("Found relation:", rel.Name, rel.Type)
    }
}
