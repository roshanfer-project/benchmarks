variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "dynamic-large"
}

group "default" {
  targets = ["backend1", "backend2", "backend3", "backend4", "backend5", "backend6", "backend7", "backend8", "frontend", "frontend-grpc", "rajomon-client"]
}

target "backend1" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend1"
  tags = ["${REGISTRY}/${BENCH}-backend1:${TAG}"]
}

target "backend2" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend2"
  tags = ["${REGISTRY}/${BENCH}-backend2:${TAG}"]
}

target "backend3" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend3"
  tags = ["${REGISTRY}/${BENCH}-backend3:${TAG}"]
}

target "backend4" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend4"
  tags = ["${REGISTRY}/${BENCH}-backend4:${TAG}"]
}

target "backend5" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend5"
  tags = ["${REGISTRY}/${BENCH}-backend5:${TAG}"]
}

target "backend6" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend6"
  tags = ["${REGISTRY}/${BENCH}-backend6:${TAG}"]
}

target "backend7" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend7"
  tags = ["${REGISTRY}/${BENCH}-backend7:${TAG}"]
}

target "backend8" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-backend8"
  tags = ["${REGISTRY}/${BENCH}-backend8:${TAG}"]
}

target "frontend" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-frontend"
  tags = ["${REGISTRY}/${BENCH}-frontend:${TAG}"]
}

target "frontend-grpc" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-frontend-grpc"
  tags = ["${REGISTRY}/${BENCH}-frontend-grpc:${TAG}"]
}

target "rajomon-client" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-rajomon-client"
  tags = ["${REGISTRY}/${BENCH}-rajomon-client:${TAG}"]
}

