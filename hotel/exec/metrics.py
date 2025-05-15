#!/home/farzad/files/venv/bin/python3

import numpy as np
import argparse
from datetime import datetime

def main():
    parser = argparse.ArgumentParser(prog="metrics")
    parser.add_argument("--file", type=str, required=True)
    parser.add_argument("--window", type=float, default=0,
                        help="Window size in seconds for rate over time plot (e.g., 0.1 for 100ms)")
    args = parser.parse_args()

    metrics = {}

    with open(args.file, "r") as file:
        lines = file.readlines()
        for line in lines:
            try:
                # template: M# <sidecar name> <metric name> <connection type> <service>:<method> <value>
                _, timestamp, _, tag, service_name, metric_name, conn_type, rpc_path, value = line.strip().split(" ")
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
                metrics[service_name][metric_name][conn_type][rpc_path]["timestamps"].append(timestamp)
    
    for service_name, data in metrics.items():
        print(rf"/////////////////////////////////////////////  {service_name}  \\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\\")
        for metric_name, content in data.items():
            for conn_type, rpc_data in content.items():
                for rpc_path, rpc_content in rpc_data.items():
                    print_info(rpc_content["values"], rpc_content["timestamps"], f"{metric_name}  {conn_type}  {rpc_path}", rate_window=args.window)

def print_info(data, timestamps, name, bins=10, bar_char='█', width=40, rate_window=0):
    print(f"###########   {name}   ###########")

    p50 = np.percentile(data, 50)
    p95 = np.percentile(data, 95)
    p99 = np.percentile(data, 99)
    print(f"Count: {len(data)}")
    print(f"50th: {p50:.2f} us")
    print(f"95th: {p95:.2f} us")
    print(f"99th: {p99:.2f} us")

    # Calculate overall rate using the timestamps
    dt_list = [datetime.strptime(ts, "%H:%M:%S.%f") for ts in timestamps]
    start_time = min(dt_list)
    end_time = max(dt_list)
    duration = (end_time - start_time).total_seconds()
    if duration > 0:
        rate = len(timestamps) / duration
    else:
        rate = len(timestamps)
    print(f"Rate: {rate:.2f} events per second")

    # Plot rate over time (relative to the beginning) if a window size is provided
    if rate_window > 0:
        print("\nRate over time (relative to beginning):")
        # Compute time offsets from start_time (in seconds)
        time_offsets = np.array([(dt - start_time).total_seconds() for dt in dt_list])
        # Create bins based on the specified window
        bins_arr = np.arange(0, duration + rate_window, rate_window)
        counts, _ = np.histogram(time_offsets, bins=bins_arr)
        window_rates = counts / rate_window
        max_rate_window = max(window_rates) if len(window_rates) > 0 else 1
        for i in range(len(window_rates)):
            win_start = bins_arr[i]
            win_end = bins_arr[i+1]
            rate_val = window_rates[i]
            bar_len = int((rate_val / max_rate_window) * width)
            bar = bar_char * bar_len
            print(f"{win_start:6.2f}-{win_end:6.2f} s: {bar} ({rate_val:.2f} events/s)")

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

if __name__ == "__main__":
    main()