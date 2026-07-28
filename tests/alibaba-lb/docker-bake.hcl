variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "alibaba-lb"
}

group "default" {
  targets = ["ms-25806", "ms-2687", "ms-40087", "ms-44246", "ms-51787", "ms-56113", "ms-64512", "ms-70124", "ms-64512-grpc", "rajomon-client"]
}

target "ms-25806" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-25806"
  tags = ["${REGISTRY}/${BENCH}-ms-25806:${TAG}"]
}

target "ms-2687" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-2687"
  tags = ["${REGISTRY}/${BENCH}-ms-2687:${TAG}"]
}

target "ms-40087" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-40087"
  tags = ["${REGISTRY}/${BENCH}-ms-40087:${TAG}"]
}

target "ms-44246" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-44246"
  tags = ["${REGISTRY}/${BENCH}-ms-44246:${TAG}"]
}

target "ms-51787" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-51787"
  tags = ["${REGISTRY}/${BENCH}-ms-51787:${TAG}"]
}

target "ms-56113" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-56113"
  tags = ["${REGISTRY}/${BENCH}-ms-56113:${TAG}"]
}

target "ms-64512" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-64512"
  tags = ["${REGISTRY}/${BENCH}-ms-64512:${TAG}"]
}

target "ms-70124" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-70124"
  tags = ["${REGISTRY}/${BENCH}-ms-70124:${TAG}"]
}

target "ms-64512-grpc" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-64512-grpc"
  tags = ["${REGISTRY}/${BENCH}-ms-64512-grpc:${TAG}"]
}

target "rajomon-client" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-rajomon-client"
  tags = ["${REGISTRY}/${BENCH}-rajomon-client:${TAG}"]
}

