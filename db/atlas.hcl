env "local" {
  src = "ent://ent/schema"
  dev = "postgres://postgres:dev@localhost:54320/atlas_dev?sslmode=disable"
  migration {
    dir = "file://migrations?format=golang-migrate"
  }
}