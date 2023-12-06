package main

import (
	"context"
	"ent-grpc-prac/ent"
	"log"

	"time"

	"github.com/go-sql-driver/mysql"
)

func main() {
	entOptions := []ent.Option{}

	// 発行されるSQLをロギングするなら
	entOptions = append(entOptions, ent.Debug())

	// サンプルなのでここにハードコーディングしてます。
	mc := mysql.Config{
		User:                 "user",
		DBName:               "ent-grpc-prac-mysql",
		Passwd:               "password",
		Net:                  "tcp",
		Addr:                 "mysql:3306",
		AllowNativePasswords: true,
		ParseTime:            true,
	}

	var client *ent.Client
	var err error
	for i := 0; i < 10; i++ {
		client, err = ent.Open("mysql", mc.FormatDSN(), entOptions...)
		if err == nil {
			break
		}
		log.Printf("Error open mysql ent client: %v\n", err)
		time.Sleep(time.Second * 5)
	}
	if err != nil {
		log.Fatalf("Error open mysql ent client: %v\n", err)
	}

	defer client.Close()

	// Run the auto migration tool.
	if err := client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}
}
