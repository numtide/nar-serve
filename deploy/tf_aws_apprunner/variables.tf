variable "name" {
  description = "Resource name name"
}

variable "tags" {
  description = "Tag all the AWS resources with these"
  type        = map(string)
  default     = {}
}

# You will have to copy a release from
# https://hub.docker.com/repository/docker/numtide/nar-serve/tags
# into that ECR registry.
variable "image_tag" {
  description = "nar-serve docker tag"
}

variable "cache_url" {
  description = "URL of the cache to point nar-cache to"
  default     = "https://cache.nixos.org/"
}
