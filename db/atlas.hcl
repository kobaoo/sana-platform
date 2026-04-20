env "local" {
  src = [
    "ent://../orgstructure/organizations/ent/schema",
    "ent://../orgstructure/users/ent/schema",
    "ent://../orgstructure/contracts_dzo/ent/schema",
  ]
  dev = "docker://postgres/16/dev?search_path=public"
  migration {
    dir = "file://migrations"
  }
}
