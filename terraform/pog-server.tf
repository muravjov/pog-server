resource "google_cloud_run_v2_service" "pogserver" {
  name     = "pog-server"
  location = "us-east4"
  client   = "terraform"

  template {
    containers {
      image = "us-central1-docker.pkg.dev/PROJECT/docker-repo/pog-server:dev"

      env {
        name  = "POG_AUTH_ITEM1"
        value = <<EOVALUE
{"name":"pog-client","hash":"$2a$10$oiV27ssmxy3ihPYA4w.rIOAH2eUQOnwCXoHL4PKXSZz2goKvL.Nwq","exp_date":"2035-11-12T05:22:05Z"}
EOVALUE

      }

      resources {
        limits = {
          cpu    = "1000m"
          memory = "1Gi"
        }
      }

      ports {
        name           = "h2c"
        container_port = 8080
      }

    }

    scaling {
      max_instance_count = 10

    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "noauth" {
  location = google_cloud_run_v2_service.pogserver.location
  name     = google_cloud_run_v2_service.pogserver.name
  role     = "roles/run.invoker"
  member   = "allUsers"
}
