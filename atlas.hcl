data "external_schema" "control" {
  program = ["go", "run", "./tools/control"]
}

data "external_schema" "tenant" {
  program = ["go", "run", "./tools/tenant"]
}

variable "dev_url" {
  type    = string
  default = getenv("ATLAS_DEV_URL")
}

locals {
  dev_url = var.dev_url != "" ? var.dev_url : "docker://postgres/18/dev?search_path=public"
}

env "control" {
  src = data.external_schema.control.url
  dev = local.dev_url
  migration {
    dir    = "file://internal/migrations/control"
    format = goose
  }
}

env "tenant" {
  src = data.external_schema.tenant.url
  dev = local.dev_url
  migration {
    dir    = "file://internal/migrations/tenant"
    format = goose
  }
}
