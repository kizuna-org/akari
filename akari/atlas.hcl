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
  src = "ent://ent/schema"
  dev = "docker://postgres/17/dev?search_path=public"
  url = format(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    local.envfile["POSTGRES_USER"],
    local.envfile["POSTGRES_PASSWORD"],
    local.envfile["POSTGRES_HOST"],
    local.envfile["POSTGRES_PORT"],
    local.envfile["POSTGRES_DB"],
    local.envfile["POSTGRES_SSLMODE"]
  )
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
  src = "ent://ent/schema"
  dev = "docker://postgres/17/dev?search_path=public"
  url = format(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    local.envfile_test["POSTGRES_USER"],
    local.envfile_test["POSTGRES_PASSWORD"],
    local.envfile_test["POSTGRES_HOST"],
    local.envfile_test["POSTGRES_PORT"],
    local.envfile_test["POSTGRES_DB"],
    local.envfile_test["POSTGRES_SSLMODE"]
  )
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
  src = "ent://ent/schema"
  url = format(
    "postgres://%s:%s@%s:%s/%s?sslmode=%s",
    getenv("POSTGRES_USER"),
    getenv("POSTGRES_PASSWORD"),
    getenv("POSTGRES_HOST"),
    getenv("POSTGRES_PORT"),
    getenv("POSTGRES_DB"),
    getenv("POSTGRES_SSLMODE")
  )
  migration {
    dir = "file://internal/database/migrations"
  }
}
