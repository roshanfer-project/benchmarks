variable "REGISTRY" {
  default = "farzad1132"
}
variable "TAG" {
  default = "latest"
}
variable "BENCH" {
  default = "alibaba-large"
}

group "default" {
  targets = ["ms-12657", "ms-14758", "ms-18750", "ms-19439", "ms-21298", "ms-25781", "ms-25806", "ms-2687", "ms-33572", "ms-38190", "ms-40087", "ms-41667", "ms-43032", "ms-43754", "ms-44246", "ms-45067", "ms-51783", "ms-51787", "ms-53792", "ms-56113", "ms-5720", "ms-58796", "ms-62039", "ms-64512", "ms-66921", "ms-67465", "ms-70124", "ms-7103", "ms-9105", "ms-64512-grpc", "rajomon-client"]
}

target "ms-12657" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-12657"
  tags = ["${REGISTRY}/${BENCH}-ms-12657:${TAG}"]
}

target "ms-14758" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-14758"
  tags = ["${REGISTRY}/${BENCH}-ms-14758:${TAG}"]
}

target "ms-18750" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-18750"
  tags = ["${REGISTRY}/${BENCH}-ms-18750:${TAG}"]
}

target "ms-19439" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-19439"
  tags = ["${REGISTRY}/${BENCH}-ms-19439:${TAG}"]
}

target "ms-21298" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-21298"
  tags = ["${REGISTRY}/${BENCH}-ms-21298:${TAG}"]
}

target "ms-25781" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-25781"
  tags = ["${REGISTRY}/${BENCH}-ms-25781:${TAG}"]
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

target "ms-33572" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-33572"
  tags = ["${REGISTRY}/${BENCH}-ms-33572:${TAG}"]
}

target "ms-38190" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-38190"
  tags = ["${REGISTRY}/${BENCH}-ms-38190:${TAG}"]
}

target "ms-40087" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-40087"
  tags = ["${REGISTRY}/${BENCH}-ms-40087:${TAG}"]
}

target "ms-41667" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-41667"
  tags = ["${REGISTRY}/${BENCH}-ms-41667:${TAG}"]
}

target "ms-43032" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-43032"
  tags = ["${REGISTRY}/${BENCH}-ms-43032:${TAG}"]
}

target "ms-43754" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-43754"
  tags = ["${REGISTRY}/${BENCH}-ms-43754:${TAG}"]
}

target "ms-44246" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-44246"
  tags = ["${REGISTRY}/${BENCH}-ms-44246:${TAG}"]
}

target "ms-45067" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-45067"
  tags = ["${REGISTRY}/${BENCH}-ms-45067:${TAG}"]
}

target "ms-51783" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-51783"
  tags = ["${REGISTRY}/${BENCH}-ms-51783:${TAG}"]
}

target "ms-51787" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-51787"
  tags = ["${REGISTRY}/${BENCH}-ms-51787:${TAG}"]
}

target "ms-53792" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-53792"
  tags = ["${REGISTRY}/${BENCH}-ms-53792:${TAG}"]
}

target "ms-56113" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-56113"
  tags = ["${REGISTRY}/${BENCH}-ms-56113:${TAG}"]
}

target "ms-5720" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-5720"
  tags = ["${REGISTRY}/${BENCH}-ms-5720:${TAG}"]
}

target "ms-58796" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-58796"
  tags = ["${REGISTRY}/${BENCH}-ms-58796:${TAG}"]
}

target "ms-62039" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-62039"
  tags = ["${REGISTRY}/${BENCH}-ms-62039:${TAG}"]
}

target "ms-64512" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-64512"
  tags = ["${REGISTRY}/${BENCH}-ms-64512:${TAG}"]
}

target "ms-66921" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-66921"
  tags = ["${REGISTRY}/${BENCH}-ms-66921:${TAG}"]
}

target "ms-67465" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-67465"
  tags = ["${REGISTRY}/${BENCH}-ms-67465:${TAG}"]
}

target "ms-70124" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-70124"
  tags = ["${REGISTRY}/${BENCH}-ms-70124:${TAG}"]
}

target "ms-7103" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-7103"
  tags = ["${REGISTRY}/${BENCH}-ms-7103:${TAG}"]
}

target "ms-9105" {
  context = "."
  dockerfile = "Dockerfile"
  target = "svc-ms-9105"
  tags = ["${REGISTRY}/${BENCH}-ms-9105:${TAG}"]
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

