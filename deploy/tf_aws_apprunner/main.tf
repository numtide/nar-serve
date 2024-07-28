# Make a copy of the docker image in that repository
resource "aws_ecr_repository" "nar_serve" {
  name                 = var.name
  image_tag_mutability = "MUTABLE"
  tags                 = var.tags
}

resource "aws_iam_role" "nar_serve_access_role" {
  name               = "${var.name}-access-role"
  assume_role_policy = <<ASSUME_ROLE
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "build.apprunner.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
ASSUME_ROLE
  managed_policy_arns = [
    "arn:aws:iam::aws:policy/service-role/AWSAppRunnerServicePolicyForECRAccess",
  ]
}

resource "aws_apprunner_service" "nar_serve" {
  service_name = var.name
  tags         = var.tags

  source_configuration {
    auto_deployments_enabled = false

    authentication_configuration {
      access_role_arn = aws_iam_role.nar_serve_access_role.arn
    }

    image_repository {
      image_configuration {
        port = "8383"
        runtime_environment_variables = {
          NIX_CACHE_URL = var.cache_url
        }
      }
      image_identifier      = "${aws_ecr_repository.nar_serve.repository_url}:${var.image_tag}"
      image_repository_type = "ECR"
    }
  }

  health_check_configuration {
    healthy_threshold   = 1
    interval            = 5
    path                = "/healthz"
    protocol            = "HTTP"
    timeout             = 1
    unhealthy_threshold = 3
  }
}
