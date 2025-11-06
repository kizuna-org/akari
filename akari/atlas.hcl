locals {
  envfile = {
    for line in split("\n", file(".env")) : split("=", line)[0] => regex("=(.*)", line)[0]
    if !startswith(line, "#") && length(split("=", line)) > 1
  }
  envfile_test = {
    for line in split("\n", file(".env.test")) : split("=", line)[0] => regex("=(.*)", line)[0]
    if !startswith(line, "#") && length(split("=", line)) > 1
  }
}

env "local" {
  src = "ent://internal/ent/schema"
  dev = "docker://postgres/15/dev?search_path=public"
  url = local.envfile["DATABASE_URL"]
  migration {
    dir = "file://internal/database/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "test" {
  src = "ent://internal/ent/schema"
  dev = "docker://postgres/15/dev?search_path=public"
  url = local.envfile_test["DATABASE_URL"]
  migration {
    dir = "file://internal/database/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "dev" {
  src = "ent://internal/ent/schema"
  url = getenv("DATABASE_URL")
  migration {
    dir = "file://internal/database/migrations"
  }
}
