package db

import "encore.dev/storage/sqldb"

// DB — единая база данных приложения.
// Сервисы обращаются к ней через sqldb.Named("lms").
var DB = sqldb.NewDatabase("lms", sqldb.DatabaseConfig{
	Migrations: "./migrations",
})
