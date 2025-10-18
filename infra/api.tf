resource "google_project_service" "aiplatform" {
  project = local.project_id
  service = "aiplatform.googleapis.com"
}
