variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "fan-out-dynamic-0-9"
}

group "default" {
  targets = ["backend1", "backend2", "frontend", "frontend-grpc", "rajomon-client"]
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

