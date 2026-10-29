variable "REPO" {
  default = "dubml"
}

variable "TAG" {
  default = "debug"
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
}

images = [
  { name = "ad", dockerfile = "docker/Dockerfile.go", target = "runtime" },
  { name = "cart", dockerfile = "docker/Dockerfile.python", target = "runtime" },
  { name = "checkout", dockerfile = "docker/Dockerfile.go", target = "runtime" },
  { name = "currency", dockerfile = "docker/Dockerfile.rust", target = "runtime" },
  { name = "email", dockerfile = "docker/Dockerfile.php", target = "runtime" },
  { name = "frontend", dockerfile = "docker/Dockerfile.go", target = "frontend" },
  { name = "payment", dockerfile = "docker/Dockerfile.php", target = "runtime" },
  { name = "product-catalogs", dockerfile = "docker/Dockerfile.rust", target = "runtime" },
  { name = "recommendation", dockerfile = "docker/Dockerfile.go", target = "runtime" },
  { name = "shipping", dockerfile = "docker/Dockerfile.php", target = "runtime" },
]

target "default" {
  matrix = {
    service = images
  }
  name       = service.name
  context    = "."
  dockerfile = service.dockerfile
  target     = service.target
  tags       = ["${REPO}/online-boutique-${service.name}:${TAG}"]
  platforms  = split(",", PLATFORMS)
  args = {
    SERVICE = service.name
  }
}
