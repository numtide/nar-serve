output "url" {
  description = "Where the service is being deployed to"
  value       = aws_apprunner_service.nar_serve.service_url
}
