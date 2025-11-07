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
  url = try(local.envfile["DATABASE_URL"], local.database_url)
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
  url = try(local.envfile_test["DATABASE_URL"], local.database_url_test)
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
  url = try(
    getenv("DATABASE_URL"),
    format(
      "postgres://%s:%s@%s:%s/%s?sslmode=%s",
      try(getenv("POSTGRES_USER"), "postgres"),
      getenv("POSTGRES_PASSWORD"),
      try(getenv("POSTGRES_HOST"), "localhost"),
      try(getenv("POSTGRES_PORT"), "5432"),
      getenv("POSTGRES_DB"),
      try(getenv("POSTGRES_SSLMODE"), "disable")
    )
  )
  migration {
    dir = "file://internal/database/migrations"
  }
}
