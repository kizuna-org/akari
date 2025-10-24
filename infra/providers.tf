terraform {
  required_version = ">= 1.13.4, < 2.0.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "7.8.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.5.3"
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
