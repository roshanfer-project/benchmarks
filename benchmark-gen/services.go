package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func generateGenericServices(genConfig *GeneratedConfig, outputDir string) error {
	servicesDir := filepath.Join(outputDir, "services")
	if err := os.MkdirAll(servicesDir, 0755); err != nil {
		return err
	}

	// Generate generic frontend service
	if err := generateFrontendService(genConfig, servicesDir); err != nil {
		return err
	}

	// Only generate backend service if there are backend services defined
	hasBackends := false
	for _, svc := range genConfig.Services {
		if svc.Type == "backend" {
			hasBackends = true
			break
		}
	}

	if hasBackends {
		// Generate generic backend service
		if err := generateBackendService(genConfig, servicesDir); err != nil {
			return err
		}
	}

	return nil
}

func generateFrontendService(genConfig *GeneratedConfig, outputDir string) error {
	frontendDir := filepath.Join(outputDir, "frontend")
	if err := os.MkdirAll(frontendDir, 0755); err != nil {
		return err
	}

	// Build endpoint configuration from edges
	endpointConfigs := make(map[string][]EndpointCall)
	for _, edge := range genConfig.Edges {
		frontendName := edge.From
		if endpointConfigs[frontendName] == nil {
			endpointConfigs[frontendName] = []EndpointCall{}
		}
		// For now, map each HTTP endpoint to all its RPC calls
		// We'll need to enhance this based on actual requirements
	}

	// Generate client variables and initialization
	backendClients := make(map[string]bool)
	for _, edge := range genConfig.Edges {
		backendClients[edge.To] = true
	}

	var backends []string
	for backend := range backendClients {
		backends = append(backends, backend)
	}
	sort.Strings(backends)

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"net/http\"\n")
	sb.WriteString("\t\"strconv\"\n")
	sb.WriteString("\t\"strings\"\n")
	if len(backends) > 0 {
		sb.WriteString("\t\"test/test\"\n")
	}
	sb.WriteString("\t\"test/protobuf\"\n")
	sb.WriteString("\t\"test/test/utils\"\n\n")
	sb.WriteString("\t\"google.golang.org/grpc/metadata\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("var serviceName string\n")
	sb.WriteString("var listenPort int\n")
	sb.WriteString("var sidecar bool\n")
	sb.WriteString("var responseSize int\n")
	sb.WriteString("var preRepeat int\n")
	sb.WriteString("var postRepeat int\n")
	sb.WriteString("var log = utils.GetLogger(\"frontend\")\n\n")

	for _, backend := range backends {
		sb.WriteString(fmt.Sprintf("var client%s protobuf.%sClient\n", backend, backend))
	}

	sb.WriteString("\nfunc main() {\n")
	sb.WriteString("\thttp.HandleFunc(\"/\", func(w http.ResponseWriter, r *http.Request) {\n")
	sb.WriteString("\t\tendpoint := r.URL.Path\n")
	sb.WriteString("\t\thandleEndpoint(w, r, endpoint)\n")
	sb.WriteString("\t})\n")
	sb.WriteString("\thttp.ListenAndServe(fmt.Sprintf(\":%d\", listenPort), nil)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func getContextWithRpcId(r *http.Request) context.Context {\n")
	sb.WriteString("\tif sidecar {\n")
	sb.WriteString("\t\trpcId := r.Header.Get(\"rpc-id\")\n")
	sb.WriteString("\t\tif rpcId == \"\" {\n")
	sb.WriteString("\t\t\tlog.Error(\"rpc-id header is required\")\n")
	sb.WriteString("\t\t\treturn r.Context()\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tmd := metadata.New(map[string]string{\"rpc-id\": rpcId})\n")
	sb.WriteString("\t\treturn metadata.NewOutgoingContext(r.Context(), md)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn r.Context()\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func init() {\n")
	sb.WriteString("\tserviceName = utils.GetEnvVar(\"SERVICE_NAME\", true)\n")
	sb.WriteString("\tsidecar = utils.GetEnvVar(\"sidecar\", true) == \"true\"\n")
	sb.WriteString("\tlistenPort = utils.StrToInt(utils.GetEnvVar(\"LISTEN_PORT\", true))\n")
	sb.WriteString("\tresponseSize = utils.StrToInt(utils.GetEnvVar(\"RESPONSE_SIZE\", true))\n")
	sb.WriteString("\tpreRepeat = utils.StrToInt(utils.GetEnvVar(\"PRE_REPEAT\", true))\n")
	sb.WriteString("\tpostRepeat = utils.StrToInt(utils.GetEnvVar(\"POST_REPEAT\", true))\n\n")

	sb.WriteString("\tlog.Info(fmt.Sprintf(\"Initializing frontend service %s on port %d\", serviceName, listenPort))\n")
	sb.WriteString("\tlog.Info(fmt.Sprintf(\"Config: sidecar=%v, responseSize=%d, preRepeat=%d, postRepeat=%d\", sidecar, responseSize, preRepeat, postRepeat))\n\n")

	// Initialize clients based on configuration
	sb.WriteString("\t// Initialize gRPC clients\n")
	sb.WriteString("\tendpoints := strings.Split(utils.GetEnvVar(\"ENDPOINTS\", false), \",\")\n")
	sb.WriteString("\tfor _, endpoint := range endpoints {\n")
	sb.WriteString("\t\tif endpoint == \"\" {\n")
	sb.WriteString("\t\t\tcontinue\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tparts := strings.Split(endpoint, \"=\")\n")
	sb.WriteString("\t\tif len(parts) != 2 {\n")
	sb.WriteString("\t\t\tcontinue\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tbackendName := parts[0]\n")
	if len(backends) > 0 {
		sb.WriteString("\t\taddress := parts[1]\n")
	} else {
		sb.WriteString("\t\t_ = parts[1] // address not needed when no backends\n")
	}
	sb.WriteString("\t\tswitch backendName {\n")
	for _, backend := range backends {
		sb.WriteString(fmt.Sprintf("\t\tcase \"%s\":\n", backend))
		sb.WriteString("\t\t\tconn := test.GetConnBasic(address)\n")
		sb.WriteString(fmt.Sprintf("\t\t\tclient%s = protobuf.New%sClient(conn)\n", backend, backend))
	}
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func makebigString(size int) string {\n")
	sb.WriteString("\treturn strings.Repeat(\"a\", size)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func busyLoop(repeat int) {\n")
	sb.WriteString("\tfor range repeat {\n")
	sb.WriteString("\t\tfor range 10000 {\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func writeResponseWithoutchunkEncoding(w http.ResponseWriter, data string) {\n")
	sb.WriteString("\tresponseBytes := []byte(data)\n")
	sb.WriteString("\tw.Header().Set(\"Content-Length\", strconv.Itoa(len(responseBytes)))\n")
	sb.WriteString("\tw.Write(responseBytes)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func handleEndpoint(w http.ResponseWriter, r *http.Request, endpoint string) {\n")
	if len(backends) > 0 {
		sb.WriteString("\tctx := getContextWithRpcId(r)\n")
	} else {
		sb.WriteString("\t_ = getContextWithRpcId(r) // ctx not needed when no backends\n")
	}
	sb.WriteString("\t// Get endpoint configuration from environment\n")
	// Normalize endpoint to env var name
	sb.WriteString("\tenvName := strings.ReplaceAll(strings.ToUpper(endpoint), \"/\", \"_\")\n")
	sb.WriteString("\tenvName = strings.ReplaceAll(envName, \"__\", \"_\")\n")
	sb.WriteString("\tif strings.HasPrefix(envName, \"_\") {\n")
	sb.WriteString("\t\tenvName = envName[1:]\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tendpointConfig := utils.GetEnvVar(fmt.Sprintf(\"ENDPOINT_%s\", envName), false)\n")
	sb.WriteString("\tif endpointConfig == \"\" {\n")
	sb.WriteString("\t\thttp.Error(w, \"Unknown endpoint\", http.StatusNotFound)\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\t// Parse endpoint config: \"pre:N:method1:backend1:pre:N1:post:N2:method2:backend2:post:N\"\n")
	sb.WriteString("\tparts := strings.Split(endpointConfig, \":\")\n")
	sb.WriteString("\tif len(parts) < 3 {\n")
	sb.WriteString("\t\thttp.Error(w, \"Invalid endpoint config\", http.StatusInternalServerError)\n")
	sb.WriteString("\t\treturn\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\t// Parse initial pre-repeat\n")
	sb.WriteString("\tif parts[0] == \"pre\" && len(parts) > 1 {\n")
	sb.WriteString("\t\tif preVal := utils.StrToInt(parts[1]); preVal > 0 {\n")
	sb.WriteString("\t\t\tbusyLoop(preVal)\n")
	sb.WriteString("\t\t} else {\n")
	sb.WriteString("\t\t\tbusyLoop(preRepeat)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tparts = parts[2:]\n")
	sb.WriteString("\t} else {\n")
	sb.WriteString("\t\tbusyLoop(preRepeat)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tbigString := makebigString(responseSize)\n")
	sb.WriteString("\t// Execute RPC calls based on config\n")
	sb.WriteString("\ti := 0\n")
	sb.WriteString("\tfor i < len(parts) {\n")
	sb.WriteString("\t\tif i+1 >= len(parts) {\n")
	sb.WriteString("\t\t\tbreak\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t\tmethod := parts[i]\n")
	sb.WriteString("\t\tbackend := parts[i+1]\n")
	sb.WriteString("\t\ti += 2\n")
	sb.WriteString("\t\t// Check for per-RPC pre-repeat\n")
	sb.WriteString("\t\tif i < len(parts) && parts[i] == \"pre\" && i+1 < len(parts) {\n")
	sb.WriteString("\t\t\tif preVal := utils.StrToInt(parts[i+1]); preVal > 0 {\n")
	sb.WriteString("\t\t\t\tbusyLoop(preVal)\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\ti += 2\n")
	sb.WriteString("\t\t}\n")
	if len(backends) > 0 {
		sb.WriteString("\t\tvar resp *protobuf.Resp\n")
		sb.WriteString("\t\tvar err error\n")
		sb.WriteString("\t\tswitch backend {\n")
		for _, backend := range backends {
			sb.WriteString(fmt.Sprintf("\t\tcase \"%s\":\n", backend))
			sb.WriteString("\t\t\tresp, err = callBackendMethod(ctx, backend, method, bigString)\n")
		}
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\tif err != nil {\n")
		sb.WriteString("\t\t\thttp.Error(w, err.Error(), http.StatusInternalServerError)\n")
		sb.WriteString("\t\t\treturn\n")
		sb.WriteString("\t\t}\n")
		sb.WriteString("\t\tbigString = resp.Data\n")
	} else {
		sb.WriteString("\t\t_ = method // method not used when no backends\n")
		sb.WriteString("\t\t_ = backend // backend not used when no backends\n")
	}
	sb.WriteString("\t\t// Check for per-RPC post-repeat\n")
	sb.WriteString("\t\tif i < len(parts) && parts[i] == \"post\" && i+1 < len(parts) {\n")
	sb.WriteString("\t\t\tif postVal := utils.StrToInt(parts[i+1]); postVal > 0 {\n")
	sb.WriteString("\t\t\t\tbusyLoop(postVal)\n")
	sb.WriteString("\t\t\t} else {\n")
	sb.WriteString("\t\t\t\tbusyLoop(postRepeat)\n")
	sb.WriteString("\t\t\t}\n")
	sb.WriteString("\t\t\ti += 2\n")
	sb.WriteString("\t\t} else {\n")
	sb.WriteString("\t\t\tbusyLoop(postRepeat)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\t// Final post-repeat\n")
	sb.WriteString("\tif i < len(parts) && parts[i] == \"post\" && i+1 < len(parts) {\n")
	sb.WriteString("\t\tif postVal := utils.StrToInt(parts[i+1]); postVal > 0 {\n")
	sb.WriteString("\t\t\tbusyLoop(postVal)\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\twriteResponseWithoutchunkEncoding(w, bigString)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func callBackendMethod(ctx context.Context, backend, method string, data string) (*protobuf.Resp, error) {\n")
	if len(backends) > 0 {
		sb.WriteString("\targ := &protobuf.Arg{Data: data}\n")
	} else {
		sb.WriteString("\t_ = data // data not used when no backends\n")
		sb.WriteString("\tvar arg *protobuf.Arg\n")
		sb.WriteString("\t_ = arg\n")
	}
	sb.WriteString("\tswitch backend {\n")
	for _, backend := range backends {
		sb.WriteString(fmt.Sprintf("\tcase \"%s\":\n", backend))
		sb.WriteString("\t\tswitch method {\n")
		methods := genConfig.ProtoServices[backend]
		for _, method := range methods {
			sb.WriteString(fmt.Sprintf("\t\tcase \"%s\":\n", method))
			sb.WriteString(fmt.Sprintf("\t\t\treturn client%s.%s(ctx, arg)\n", backend, method))
		}
		sb.WriteString("\t\tdefault:\n")
		sb.WriteString("\t\t\treturn nil, fmt.Errorf(\"unknown method %s for backend %s\", method, backend)\n")
		sb.WriteString("\t\t}\n")
	}
	sb.WriteString("\tdefault:\n")
	sb.WriteString("\t\treturn nil, fmt.Errorf(\"unknown backend %s\", backend)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	mainFile := filepath.Join(frontendDir, "main.go")
	return os.WriteFile(mainFile, []byte(sb.String()), 0644)
}

func generateBackendService(genConfig *GeneratedConfig, outputDir string) error {
	backendDir := filepath.Join(outputDir, "backend")
	if err := os.MkdirAll(backendDir, 0755); err != nil {
		return err
	}

	var sb strings.Builder
	sb.WriteString("package main\n\n")
	sb.WriteString("import (\n")
	sb.WriteString("\t\"context\"\n")
	sb.WriteString("\t\"fmt\"\n")
	sb.WriteString("\t\"net\"\n")
	sb.WriteString("\t\"test/test\"\n")
	sb.WriteString("\t\"test/protobuf\"\n")
	sb.WriteString("\t\"test/test/utils\"\n\n")
	sb.WriteString("\t\"google.golang.org/grpc\"\n")
	sb.WriteString(")\n\n")

	sb.WriteString("var serviceName string\n")
	sb.WriteString("var listenPort int\n")
	sb.WriteString("var backendRepeat int\n")
	sb.WriteString("var log = utils.GetLogger(\"backend\")\n\n")

	// Generate service-specific implementations
	var backends []string
	for backend := range genConfig.ProtoServices {
		backends = append(backends, backend)
	}
	sort.Strings(backends)

	for _, backend := range backends {
		methods := genConfig.ProtoServices[backend]
		sb.WriteString(fmt.Sprintf("type %sImpl struct {\n", backend))
		sb.WriteString(fmt.Sprintf("\tprotobuf.Unimplemented%sServer\n", backend))
		sb.WriteString("}\n\n")

		for _, method := range methods {
			sb.WriteString(fmt.Sprintf("func (s *%sImpl) %s(ctx context.Context, req *protobuf.Arg) (*protobuf.Resp, error) {\n", backend, method))
			sb.WriteString("\tbusyLoop(backendRepeat)\n")
			sb.WriteString("\treturn &protobuf.Resp{\n")
			sb.WriteString("\t\tData: \"Hello, \" + req.Data,\n")
			sb.WriteString("\t}, nil\n")
			sb.WriteString("}\n\n")
		}
	}

	sb.WriteString("func busyLoop(repeat int) {\n")
	sb.WriteString("\tfor range repeat {\n")
	sb.WriteString("\t\tfor range 10000 {\n")
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func Run() error {\n")
	sb.WriteString("\topts := test.Opts\n")
	sb.WriteString("\tsrv := grpc.NewServer(opts...)\n")
	sb.WriteString("\t// Register services based on SERVICE_NAMES env var\n")
	sb.WriteString("\tserviceNames := utils.GetEnvVar(\"SERVICE_NAMES\", true)\n")
	sb.WriteString("\tnames := strings.Split(serviceNames, \",\")\n")
	sb.WriteString("\tfor _, name := range names {\n")
	sb.WriteString("\t\tswitch name {\n")
	for _, backend := range backends {
		sb.WriteString(fmt.Sprintf("\t\tcase \"%s\":\n", backend))
		sb.WriteString(fmt.Sprintf("\t\t\tprotobuf.Register%sServer(srv, &%sImpl{})\n", backend, backend))
	}
	sb.WriteString("\t\t}\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\tlis, err := net.Listen(\"tcp\", fmt.Sprintf(\":%%d\", listenPort))\n")
	sb.WriteString("\tif err != nil {\n")
	sb.WriteString("\t\treturn fmt.Errorf(\"failed to listen: %%w\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("\treturn srv.Serve(lis)\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func init() {\n")
	sb.WriteString("\tserviceName = utils.GetEnvVar(\"SERVICE_NAME\", true)\n")
	sb.WriteString("\tlistenPort = utils.StrToInt(utils.GetEnvVar(\"LISTEN_PORT\", true))\n")
	sb.WriteString("\tbackendRepeat = utils.StrToInt(utils.GetEnvVar(\"BACKEND_REPEAT\", true))\n")

	sb.WriteString("\tlog.Info(fmt.Sprintf(\"Initializing backend service %s on port %d\", serviceName, listenPort))\n")
	sb.WriteString("\tlog.Info(fmt.Sprintf(\"Config: backendRepeat=%d\", backendRepeat))\n")
	sb.WriteString("}\n\n")

	sb.WriteString("func main() {\n")
	sb.WriteString("\tlog.Info(fmt.Sprintf(\"Starting backend server %%s\", serviceName))\n")
	sb.WriteString("\tif err := Run(); err != nil {\n")
	sb.WriteString("\t\tlog.Error(\"main\", \"failed to run backend server\", err)\n")
	sb.WriteString("\t}\n")
	sb.WriteString("}\n")

	// Fix: add missing import
	content := strings.Replace(sb.String(), "import (\n\t\"context\"\n\t\"fmt\"\n\t\"net\"\n\t\"test/test\"\n\t\"test/protobuf\"\n\t\"test/test/utils\"\n\n\t\"google.golang.org/grpc\"\n)", "import (\n\t\"context\"\n\t\"fmt\"\n\t\"net\"\n\t\"strings\"\n\t\"test/test\"\n\t\"test/protobuf\"\n\t\"test/test/utils\"\n\n\t\"google.golang.org/grpc\"\n)", 1)

	mainFile := filepath.Join(backendDir, "main.go")
	return os.WriteFile(mainFile, []byte(content), 0644)
}

type EndpointCall struct {
	Method   string
	Backend  string
	Sequence int
}
