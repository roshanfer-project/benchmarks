Service: S_6624825
Trace id: T_20896218181

Generated benchmark from call graph. Regenerate with:
```bash
cd ../callgraph-framework && go run ./cmd/gen ../alibaba-large/callgraph.json -o ../alibaba-large
```

Deploy: ./build.sh [tag] && ./deploy.sh

Load tests: **`run.sh`** / **`run-plain.sh`** take **`PROTOCOL API OUTPUT_DIR`**; rates/durations come from **`RWG_RATES`** and **`RWG_DURATIONS`** (set by **`exec.runner`**). See **`callgraph-framework/README.md`**.