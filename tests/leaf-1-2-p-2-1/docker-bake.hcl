variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "leaf-1-2-p-2-1"
}

group "default" {
  targets = ["frontend", "frontend-grpc", "rajomon-client"]
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

