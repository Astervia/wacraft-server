package main

import (
	"fmt"
	"github.com/Astervia/wacraft-core/src/workspace/entity"
	"gorm.io/gorm/schema"
	"gorm.io/gorm"
	"gorm.io/driver/postgres"
)

type WorkspaceMemberWithPolicies struct {
	workspace_entity.WorkspaceMember
	Policies []workspace_entity.WorkspaceMemberPolicy `gorm:"foreignKey:WorkspaceMemberID"`
}

func main() {
    // Just parsing the schema
    db, err := gorm.Open(postgres.Open(""), &gorm.Config{})
    if err != nil {
        // ignore connection error for schema parse
    }
    s, err := schema.Parse(&WorkspaceMemberWithPolicies{}, &schema.syncMap{}, schema.NamingStrategy{})
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
