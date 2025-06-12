#!/home/farzad/files/venv/bin/python3

import numpy as np
import argparse
from datetime import datetime
import json
import os

def main():
    parser = argparse.ArgumentParser(prog="metrics")
    parser.add_argument("--file", type=str, required=True)
    parser.add_argument("--window", type=float, default=0,
                        help="Window size in seconds for rate over time plot (e.g., 0.1 for 100ms)")
    parser.add_argument("--no-print", action="store_true", help="Disable printing of additional output")
    args = parser.parse_args()

    metrics = {}

    with open(args.file, "r") as file:
        lines = file.readlines()
        for line in lines:
            try:
                # template: M# <sidecar name> <metric name> <connection type> <service>:<method> <value>
                _, timestamp, _, _, tag, service_name, metric_name, conn_type, rpc_path, value = line.strip().split(" ")
            except ValueError:
                continue

            if tag == "M#":
                if service_name not in metrics:
                    metrics[service_name] = {}
                if metric_name not in metrics[service_name]:
                    metrics[service_name][metric_name] = {}
                if conn_type not in metrics[service_name][metric_name]:
                    metrics[service_name][metric_name][conn_type] = {}
                if rpc_path not in metrics[service_name][metric_name][conn_type]:
                    metrics[service_name][metric_name][conn_type][rpc_path] = {"values": [], "timestamps": []}

                metrics[service_name][metric_name][conn_type][rpc_path]["values"].append(int(value))
                #metrics[service_name][metric_name][conn_type][rpc_path]["timestamps"].append(timestamp)
    
    export = {}
    for service_name, data in metrics.items():
        export[service_name] = {}
        print(rf"/////////////////////////////////////////////  {service_name}  \\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\")
        for metric_name, content in data.items():
            export[service_name][metric_name] = {}
            for conn_type, rpc_data in content.items():
                export[service_name][metric_name][conn_type] = {}
                for rpc_path, rpc_content in rpc_data.items():
                    name = f"{metric_name}  {conn_type}  {rpc_path}"
                    if not args.no_print:
                        print(f"###########   {name}   ###########")
                    res = percentiles(rpc_content["values"], [50, 95], no_print=args.no_print)
                    export[service_name][metric_name][conn_type][rpc_path] = res
                    #print_info(rpc_content["values"], rpc_content["timestamps"], rate_window=args.window)

        if not os.path.exists("metrics.json"):
            with open("metrics.json", "w") as tmp_file:
                tmp_file.write("{}")

        with open("metrics.json", "r+") as json_file:
            content = json_file.read()
            if content:
                json_dict = json.loads(content)
            else:
                json_dict = {}
            json_dict[service_name] = export[service_name]
            json_file.seek(0)
            json.dump(json_dict, json_file)
            json_file.truncate()

def print_info(data, timestamps, bins=10, bar_char='█', width=40, rate_window=0):
    # Print latency histogram as text output
    hist, bin_edges = np.histogram(data, bins=bins)
    max_count = max(hist)
    print("\nLatency Histogram (us):")
    for i in range(len(hist)):
        count = hist[i]
        bar_len = int((count / max_count) * width)
        bar = bar_char * bar_len
        bin_range = f"{int(bin_edges[i])}-{int(bin_edges[i+1])}".rjust(10)
        print(f"{bin_range}: {bar} ({count})")

def percentiles(data, percentiles, no_print=False):
    """Calculate the specified percentiles of the data."""
    result = {}
    if not no_print:
        print(f"Count: {len(data)}")
    for p in percentiles:
        result[p] = f"{np.percentile(data, p):.2f}"
        if not no_print:
            print(f"{p}th: {result[p]}")
    result["count"] = f"{len(data)}"
    return result

if __name__ == "__main__":
    main()