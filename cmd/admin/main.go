package main

import (
	"context"
	"flag"
	"log"
	"slices"

	"cloud.google.com/go/bigtable"
	"github.com/censys/scan-takehome/pkg/shared"
)

func main() {
	projectId := flag.String("project", "test-project", "GCP Project ID")
	instanceId := flag.String("instance", "test-instance", "GCP Bigtable Instance ID")

	ctx := context.Background()
	log.Printf("Started admin")
	client, errClient := bigtable.NewAdminClient(ctx, *projectId, *instanceId)
	if errClient != nil {
		panic(errClient)
	}

	defer client.Close()
	log.Printf("Admin Client created")
	tables, errTables := client.Tables(ctx)
	if errTables != nil {
		panic(errTables)
	}

	log.Printf("Tables: %v", tables)
	if !slices.Contains(tables, shared.TableName) {
		errTable := createTable(ctx, client)
		if errTable != nil {
			panic(errTable)
		}

		log.Printf("Table successfully created")
	}
}

func createTable(ctx context.Context, client *bigtable.AdminClient) error {
	log.Printf("Creating table")
	errTable := client.CreateTable(ctx, shared.TableName)
	if errTable != nil {
		return errTable
	}

	log.Printf("Creating column families")

	errTimestamp := client.CreateColumnFamily(ctx, shared.TableName, shared.TimestampColumn)
	if errTimestamp != nil {
		return errTimestamp
	}

	log.Printf("Created timestamp column family")

	errIp := client.CreateColumnFamily(ctx, shared.TableName, shared.IpColumn)
	if errIp != nil {
		return errIp
	}

	log.Printf("Created ip column family")

	errPort := client.CreateColumnFamily(ctx, shared.TableName, shared.PortColumn)
	if errPort != nil {
		return errPort
	}

	log.Printf("Created port column family")

	errService := client.CreateColumnFamily(ctx, shared.TableName, shared.ServiceColumn)
	if errService != nil {
		return errService
	}

	log.Printf("Created service column family")

	errData := client.CreateColumnFamily(ctx, shared.TableName, shared.DataColumn)
	if errData != nil {
		return errData
	}

	log.Printf("Created data column family")

	info, errTableInfo := client.TableInfo(ctx, shared.TableName)
	if errTableInfo == nil {
		log.Printf("table info %v", info)
	} else {
		log.Printf("FAILED: %v", errTableInfo)
	}

	return errTableInfo
}
