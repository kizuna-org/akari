terraform {
  required_version = ">= 1.14.3, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "7.17.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.6.1"
    }
  }

  backend "gcs" {
    bucket = "kizuna-org-akari-tfstate"
    prefix = "terraform/state"
  }
}

provider "google" {
  project = local.project_id
  region  = local.region
  zone    = local.zone
}
