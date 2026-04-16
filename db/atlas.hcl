env "local" {
  src = "ent://ent/schema"
  dev = "postgres://postgres:dev@localhost:54320/atlas_dev?sslmode=disable"
  url = "postgres://postgres:dev@localhost:54320/postgres?sslmode=disable"
  migration {
    dir = "file://migrations"
  }
}