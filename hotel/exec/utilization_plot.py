#!/home/farzad/files/venv/bin/python3

import json
import matplotlib.pyplot as plt
import numpy as np
import argparse
import os
import sys
from datetime import datetime

# Add the experiments directory to the path to import canvas
sys.path.append('/home/farzad/files/ppm/experiments')
from canvas.canvas import create_canvas, marker_list, color_list

def main():
    parser = argparse.ArgumentParser(prog="utilization_plot")
    parser.add_argument("--file", type=str, default=".log/utilization.json", 
                        help="Path to utilization.json file")
    parser.add_argument("--output", type=str, default="utilization_plots",
                        help="Output directory for plots")
    parser.add_argument("--format", type=str, default="png", choices=["png", "pdf", "svg"],
                        help="Output format for plots")
    parser.add_argument("--show", action="store_true", help="Show plots interactively")
    args = parser.parse_args()

    if not os.path.exists(args.file):
        print(f"Error: {args.file} not found. Run metrics.py first to generate utilization data.")
        return

    # Create output directory
    if not os.path.exists(args.output):
        os.makedirs(args.output)

    # Load utilization data
    with open(args.file, "r") as f:
        utilization_data = json.load(f)

    if not utilization_data:
        print("No utilization data found.")
        return

    # Plot consolidated figure with all services
    #plot_consolidated_figure(utilization_data, args.output, args.format)

    # Plot individual plot types with all services consolidated
    plot_consolidated_box_plots(utilization_data, args.output, "pdf")
    plot_consolidated_time_series(utilization_data, args.output, "pdf")
    plot_consolidated_summary_stats(utilization_data, args.output, "pdf")
    plot_consolidated_histograms(utilization_data, args.output, "pdf")

    if args.show:
        plt.show()

    print(f"Plots saved to {args.output}/")

def plot_box_plots(data, output_dir, file_format):
    """Create box plots showing Average Req In distribution for each sidecar/service combination."""
    fig, axes = plt.subplots(figsize=(15, 10))
    
    all_values = []
    all_labels = []
    colors = plt.cm.Set3(np.linspace(0, 1, len(data)))
    
    for i, (sidecar_name, services) in enumerate(data.items()):
        for service_name, service_data in services.items():
            all_values.append(service_data["values"])
            all_labels.append(f"{sidecar_name}\n{service_name}")
    
    if all_values:
        box_plot = axes.boxplot(all_values, labels=all_labels, patch_artist=True)
        
        # Color the boxes
        for patch, color in zip(box_plot['boxes'], colors):
            patch.set_facecolor(color)
        
        axes.set_title("Average Req In Distribution by Sidecar and Service", fontsize=16, fontweight='bold')
        axes.set_ylabel("Utilization", fontsize=12)
        axes.set_xlabel("Sidecar / Service", fontsize=12)
        axes.grid(True, alpha=0.3)
        
        # Rotate labels for better readability
        plt.xticks(rotation=45, ha='right')
        plt.tight_layout()
        
        plt.savefig(f"{output_dir}/utilization_boxplot.{file_format}", dpi=300, bbox_inches='tight')
        plt.close()

def plot_time_series(data, output_dir, file_format):
    """Create time series plots for each sidecar."""
    for sidecar_name, services in data.items():
        if not services:
            continue
            
        fig, ax = plt.subplots(figsize=(12, 6))
        
        for service_name, service_data in services.items():
            values = service_data["values"]
            # Create a simple time index since we have values
            time_points = range(len(values))
            ax.plot(time_points, values, marker='o', markersize=5, label=service_name, linewidth=1.5)
        
        ax.set_title(f"Utilization Over Time - {sidecar_name}", fontsize=14, fontweight='bold')
        ax.set_xlabel("Measurement Index", fontsize=12)
        ax.set_ylabel("Utilization", fontsize=12)
        ax.legend()
        ax.grid(True, alpha=0.3)
        
        plt.tight_layout()
        plt.savefig(f"{output_dir}/utilization_timeseries_{sidecar_name}.{file_format}", dpi=300, bbox_inches='tight')
        plt.close()

def plot_summary_stats(data, output_dir, file_format):
    """Create bar charts showing mean, min, max utilization for each service."""
    fig, (ax1, ax2, ax3) = plt.subplots(1, 3, figsize=(18, 6))
    
    services = []
    means = []
    mins = []
    maxs = []
    colors = []
    
    color_map = plt.cm.Set2(np.linspace(0, 1, len(data)))
    sidecar_colors = {sidecar: color for sidecar, color in zip(data.keys(), color_map)}
    
    for sidecar_name, sidecar_services in data.items():
        for service_name, service_data in sidecar_services.items():
            services.append(f"{sidecar_name}\n{service_name}")
            means.append(service_data["mean"])
            mins.append(service_data["min"])
            maxs.append(service_data["max"])
            colors.append(sidecar_colors[sidecar_name])
    
    # Mean utilization
    bars1 = ax1.bar(range(len(services)), means, color=colors, alpha=0.7)
    ax1.set_title("Mean Utilization", fontweight='bold')
    ax1.set_ylabel("Utilization")
    ax1.set_xticks(range(len(services)))
    ax1.set_xticklabels(services, rotation=45, ha='right')
    ax1.grid(True, alpha=0.3)
    
    # Min utilization
    bars2 = ax2.bar(range(len(services)), mins, color=colors, alpha=0.7)
    ax2.set_title("Minimum Utilization", fontweight='bold')
    ax2.set_ylabel("Utilization")
    ax2.set_xticks(range(len(services)))
    ax2.set_xticklabels(services, rotation=45, ha='right')
    ax2.grid(True, alpha=0.3)
    
    # Max utilization
    bars3 = ax3.bar(range(len(services)), maxs, color=colors, alpha=0.7)
    ax3.set_title("Maximum Utilization", fontweight='bold')
    ax3.set_ylabel("Utilization")
    ax3.set_xticks(range(len(services)))
    ax3.set_xticklabels(services, rotation=45, ha='right')
    ax3.grid(True, alpha=0.3)
    
    plt.tight_layout()
    plt.savefig(f"{output_dir}/utilization_summary_stats.{file_format}", dpi=300, bbox_inches='tight')
    plt.close()

def plot_histograms(data, output_dir, file_format):
    """Create histograms showing Average Req In distribution for each service."""
    for sidecar_name, services in data.items():
        if not services:
            continue
            
        n_services = len(services)
        if n_services == 0:
            continue
            
        # Calculate subplot dimensions
        n_cols = min(3, n_services)
        n_rows = (n_services + n_cols - 1) // n_cols
        
        fig, axes = plt.subplots(n_rows, n_cols, figsize=(5*n_cols, 4*n_rows))
        if n_services == 1:
            axes = [axes]
        elif n_rows == 1:
            axes = axes.reshape(1, -1)
        
        fig.suptitle(f"Utilization Histograms - {sidecar_name}", fontsize=16, fontweight='bold')
        
        for idx, (service_name, service_data) in enumerate(services.items()):
            row = idx // n_cols
            col = idx % n_cols
            ax = axes[row, col] if n_rows > 1 else axes[col]
            
            values = service_data["values"]
            ax.hist(values, bins=20, alpha=0.7, edgecolor='black', linewidth=0.5)
            ax.set_title(f"{service_name}", fontweight='bold')
            ax.set_xlabel("Utilization")
            ax.set_ylabel("Frequency")
            ax.grid(True, alpha=0.3)
            
            # Add statistics text
            stats_text = f"Mean: {service_data['mean']:.3f}\nStd: {service_data['std']:.3f}\nCount: {service_data['count']}"
            ax.text(0.95, 0.95, stats_text, transform=ax.transAxes, 
                   verticalalignment='top', horizontalalignment='right',
                   bbox=dict(boxstyle='round', facecolor='white', alpha=0.8))
        
        # Hide empty subplots
        for idx in range(n_services, n_rows * n_cols):
            if n_rows > 1:
                row = idx // n_cols
                col = idx % n_cols
                axes[row, col].set_visible(False)
            elif n_cols > 1:
                axes[idx].set_visible(False)
        
        plt.tight_layout()
        plt.savefig(f"{output_dir}/utilization_histograms_{sidecar_name}.{file_format}", dpi=300, bbox_inches='tight')
        plt.close()

def plot_consolidated_figure(data, output_dir, file_format):
    """Create a consolidated figure with all services in subplots."""
    # Collect all sidecar-service combinations
    all_combinations = []
    for sidecar_name, services in data.items():
        for service_name, service_data in services.items():
            all_combinations.append((sidecar_name, service_name, service_data))
    
    if not all_combinations:
        print("No utilization data to plot.")
        return
    
    # Use canvas for the main comprehensive figure
    width = max(16, len(all_combinations) * 2)
    fig = plt.figure(figsize=(width, 12), dpi=300)
    
    # Apply canvas styling (fix deprecated seaborn style)
    try:
        plt.style.use("seaborn-v0_8-paper")
    except:
        plt.style.use("default")  # Fallback to default if seaborn style not available
    
    # Create a grid layout: 3 rows, with time series on top, box plots in middle, and stats on bottom
    gs = fig.add_gridspec(3, len(all_combinations), height_ratios=[2, 1, 1], hspace=0.3, wspace=0.3)
    
    # Use canvas colors
    sidecar_names = list(set([combo[0] for combo in all_combinations]))
    sidecar_colors = {sidecar: color_list[i % len(color_list)] for i, sidecar in enumerate(sidecar_names)}
    
    # Row 1: Time series plots for each service
    for i, (sidecar_name, service_name, service_data) in enumerate(all_combinations):
        ax = fig.add_subplot(gs[0, i])
        
        values = service_data["values"]
        time_points = range(len(values))
        
        ax.plot(time_points, values, color=sidecar_colors[sidecar_name], 
               linewidth=1.5, alpha=0.8, marker=marker_list[i % len(marker_list)], 
               markersize=4)
        ax.fill_between(time_points, values, alpha=0.3, color=sidecar_colors[sidecar_name])
        
        # Format service name for display
        display_name = service_name.split('/')[-1] if '/' in service_name else service_name
        ax.set_title(f"{sidecar_name}\n{display_name}", fontweight='bold')
        ax.set_xlabel("Time Index")
        ax.set_ylabel("Average Req In")
        ax.grid(True, alpha=0.3)
        
        # Add statistics text
        stats_text = f"μ={service_data['mean']:.2f}\nσ={service_data['std']:.2f}"
        ax.text(0.02, 0.98, stats_text, transform=ax.transAxes, 
               verticalalignment='top', horizontalalignment='left',
               bbox=dict(boxstyle='round,pad=0.3', facecolor='white', alpha=0.8),
               fontsize=8)
    
    # Row 2: Box plots for each service
    ax_box = fig.add_subplot(gs[1, :])
    
    all_values = []
    all_labels = []
    box_colors = []
    
    for sidecar_name, service_name, service_data in all_combinations:
        all_values.append(service_data["values"])
        display_name = service_name.split('/')[-1] if '/' in service_name else service_name
        all_labels.append(f"{sidecar_name}\n{display_name}")
        box_colors.append(sidecar_colors[sidecar_name])
    
    box_plot = ax_box.boxplot(all_values, labels=all_labels, patch_artist=True)
    
    # Color the boxes using canvas colors
    for patch, color in zip(box_plot['boxes'], box_colors):
        patch.set_facecolor(color)
        patch.set_alpha(0.7)
    
    ax_box.set_title("Average Req In Distribution Comparison", fontweight='bold')
    ax_box.set_ylabel("Average Req In")
    ax_box.grid(True, alpha=0.3)
    plt.setp(ax_box.get_xticklabels(), rotation=45, ha='right', fontsize=8)
    
    # Row 3: Summary statistics bar chart
    ax_stats = fig.add_subplot(gs[2, :])
    
    services = []
    means = []
    stds = []
    bar_colors = []
    
    for sidecar_name, service_name, service_data in all_combinations:
        display_name = service_name.split('/')[-1] if '/' in service_name else service_name
        services.append(f"{sidecar_name}\n{display_name}")
        means.append(service_data["mean"])
        stds.append(service_data["std"])
        bar_colors.append(sidecar_colors[sidecar_name])
    
    x_pos = range(len(services))
    bars = ax_stats.bar(x_pos, means, yerr=stds, capsize=5, 
                       color=bar_colors, alpha=0.7, edgecolor='black', linewidth=0.5)
    
    ax_stats.set_title("Mean Utilization with Standard Deviation", fontweight='bold')
    ax_stats.set_ylabel("Utilization")
    ax_stats.set_xticks(x_pos)
    ax_stats.set_xticklabels(services, rotation=45, ha='right', fontsize=8)
    ax_stats.grid(True, alpha=0.3, axis='y')
    
    # Add value labels on bars
    for bar, mean_val in zip(bars, means):
        height = bar.get_height()
        ax_stats.text(bar.get_x() + bar.get_width()/2., height + max(stds)*0.01,
                     f'{mean_val:.2f}', ha='center', va='bottom', fontsize=7)
    
    # Add overall title
    fig.suptitle("Comprehensive Utilization Analysis - All Services", 
                fontweight='bold', y=0.98)
    
    # Add legend for sidecars using canvas colors
    legend_elements = [plt.Rectangle((0,0),1,1, facecolor=sidecar_colors[sidecar], alpha=0.7, label=sidecar)
                      for sidecar in sidecar_names]
    fig.legend(handles=legend_elements, loc='upper right', bbox_to_anchor=(0.99, 0.95))
    
    try:
        plt.tight_layout()
        plt.subplots_adjust(top=0.93, right=0.95)
    except:
        # If tight_layout fails, just use subplots_adjust
        plt.subplots_adjust(left=0.1, right=0.95, top=0.93, bottom=0.1)
    
    # Save as PDF
    plt.savefig(f"{output_dir}/utilization_consolidated.pdf", dpi=300, bbox_inches='tight')
    plt.close()
    
    print(f"Consolidated utilization plot saved as PDF: {output_dir}/utilization_consolidated.pdf")

def plot_consolidated_box_plots(data, output_dir, file_format):
    """Create consolidated box plots with all services in one figure."""
    # Count services to determine appropriate figure size
    total_services = sum(len(services) for services in data.values())
    width = max(8, total_services * 0.8)  # Adjust width based on number of services
    
    fig, ax = create_canvas(nrows=1, ncols=1, width_in_inches=width, aspect_ratio=0.5)
    
    all_values = []
    all_labels = []
    colors = []
    
    # Use canvas color scheme
    sidecar_names = list(data.keys())
    sidecar_colors = {sidecar: color_list[i % len(color_list)] for i, sidecar in enumerate(sidecar_names)}
    
    for sidecar_name, services in data.items():
        for service_name, service_data in services.items():
            all_values.append(service_data["values"])
            display_name = service_name.split('/')[-1] if '/' in service_name else service_name
            all_labels.append(f"{sidecar_name}\n{display_name}")
            colors.append(sidecar_colors[sidecar_name])
    
    if all_values:
        box_plot = ax.boxplot(all_values, labels=all_labels, patch_artist=True)
        
        # Color the boxes using canvas colors
        for patch, color in zip(box_plot['boxes'], colors):
            patch.set_facecolor(color)
            patch.set_alpha(0.7)
        
        ax.set_title("Average Req In Distribution - All Services", fontweight='bold')
        ax.set_ylabel("Average Req In")
        ax.set_xlabel("Sidecar / Service")
        ax.grid(True, alpha=0.3)
        
        # Rotate labels for better readability
        plt.setp(ax.get_xticklabels(), rotation=45, ha='right')
        
        # Add legend for sidecars
        legend_elements = [plt.Rectangle((0,0),1,1, facecolor=sidecar_colors[sidecar], alpha=0.7, label=sidecar)
                          for sidecar in sidecar_names]
        ax.legend(handles=legend_elements, loc='upper right')
        
        try:
            plt.tight_layout()
        except:
            plt.subplots_adjust(left=0.1, right=0.9, top=0.9, bottom=0.2)
        
        plt.savefig(f"{output_dir}/utilization_boxplots_all.{file_format}", dpi=300, bbox_inches='tight')
        plt.close()
        
        print(f"Consolidated box plots saved: {output_dir}/utilization_boxplots_all.{file_format}")

def plot_consolidated_time_series(data, output_dir, file_format):
    """Create consolidated time series plots with all services in one figure."""
    fig, ax = create_canvas(nrows=1, ncols=1, width_in_inches=10, aspect_ratio=0.6, marker_size=6, line_width=1)
    
    # Create a unique combination of colors and markers for each service
    sidecar_names = list(data.keys())
    
    # Count total services to create unique combinations
    total_services = sum(len(services) for services in data.values())
    
    # Create unique color-marker combinations
    service_styles = []
    for i in range(total_services):
        color_idx = i % len(color_list)
        marker_idx = i % len(marker_list)
        service_styles.append((color_list[color_idx], marker_list[marker_idx]))
    
    style_idx = 0
    for sidecar_name, services in data.items():
        for service_name, service_data in services.items():
            values = service_data["values"]
            time_points = range(len(values))
            display_name = service_name.split('/')[-1] if '/' in service_name else service_name
            
            # Get unique color and marker for this service
            color, marker = service_styles[style_idx]
            
            ax.plot(time_points, values, 
                   color=color, 
                   label=f"{sidecar_name} - {display_name}",
                   marker=marker, alpha=0.8)
            style_idx += 1
    
    ax.set_title("Average Req In Over Time - All Services", fontweight='bold')
    ax.set_xlabel("Measurement Index")
    ax.set_ylabel("Average Req In")
    ax.grid(True, alpha=0.3)
    
    # Add major grid lines for every 1 unit in y-axis
    from matplotlib.ticker import MultipleLocator
    ax.yaxis.set_major_locator(MultipleLocator(1))
    ax.yaxis.set_minor_locator(MultipleLocator(0.5))
    ax.grid(True, which='major', alpha=0.5)
    ax.grid(True, which='minor', alpha=0.2)
    
    ax.legend(bbox_to_anchor=(1.05, 1), loc='upper left')
    
    try:
        plt.tight_layout()
    except:
        plt.subplots_adjust(left=0.1, right=0.8, top=0.9, bottom=0.1)
    
    plt.savefig(f"{output_dir}/utilization_timeseries_all.{file_format}", dpi=300, bbox_inches='tight')
    plt.close()
    
    print(f"Consolidated time series saved: {output_dir}/utilization_timeseries_all.{file_format}")

def plot_consolidated_summary_stats(data, output_dir, file_format):
    """Create consolidated summary statistics with all services in one figure."""
    # Count services to determine appropriate figure size
    total_services = sum(len(services) for services in data.values())
    height = max(8, total_services * 0.3)  # Adjust height based on number of services
    
    fig, (ax1, ax2) = create_canvas(nrows=2, ncols=1, width_in_inches=8, aspect_ratio=height/8)
    
    services = []
    means = []
    maxs = []
    colors = []
    
    # Use canvas colors
    sidecar_names = list(data.keys())
    sidecar_colors = {sidecar: color_list[i % len(color_list)] for i, sidecar in enumerate(sidecar_names)}
    
    for sidecar_name, sidecar_services in data.items():
        for service_name, service_data in sidecar_services.items():
            display_name = service_name.split('/')[-1] if '/' in service_name else service_name
            services.append(f"{sidecar_name}\n{display_name}")
            means.append(service_data["mean"])
            maxs.append(service_data["max"])
            colors.append(sidecar_colors[sidecar_name])
    
    x_pos = range(len(services))
    
    # Mean utilization
    bars1 = ax1.bar(x_pos, means, color=colors, alpha=0.7, edgecolor='black', linewidth=0.5)
    ax1.set_title("Mean Average Req In", fontweight='bold')
    ax1.set_ylabel("Average Req In")
    ax1.set_xticks(x_pos)
    ax1.set_xticklabels(services, rotation=45, ha='right')
    ax1.grid(True, alpha=0.3, axis='y')
    
    # Add value labels on bars
    for bar, mean_val in zip(bars1, means):
        height = bar.get_height()
        ax1.text(bar.get_x() + bar.get_width()/2., height + max(means)*0.01,
                f'{mean_val:.2f}', ha='center', va='bottom', fontsize=8)
    
    # Max utilization
    bars2 = ax2.bar(x_pos, maxs, color=colors, alpha=0.7, edgecolor='black', linewidth=0.5)
    ax2.set_title("Maximum Average Req In", fontweight='bold')
    ax2.set_ylabel("Average Req In")
    ax2.set_xticks(x_pos)
    ax2.set_xticklabels(services, rotation=45, ha='right')
    ax2.grid(True, alpha=0.3, axis='y')
    
    # Add value labels on bars
    for bar, max_val in zip(bars2, maxs):
        height = bar.get_height()
        ax2.text(bar.get_x() + bar.get_width()/2., height + max(maxs)*0.01,
                f'{max_val:.2f}', ha='center', va='bottom', fontsize=8)
    
    # Add overall title and legend
    fig.suptitle("Summary Statistics - All Services", fontweight='bold')
    
    # Add legend for sidecars
    legend_elements = [plt.Rectangle((0,0),1,1, facecolor=sidecar_colors[sidecar], alpha=0.7, label=sidecar)
                      for sidecar in sidecar_names]
    fig.legend(handles=legend_elements, loc='center left', bbox_to_anchor=(1.02, 0.5))
    
    try:
        plt.tight_layout()
        plt.subplots_adjust(right=0.85)
    except:
        plt.subplots_adjust(left=0.1, right=0.85, top=0.90, bottom=0.15)
    
    plt.savefig(f"{output_dir}/utilization_summary_all.{file_format}", dpi=300, bbox_inches='tight')
    plt.close()
    
    print(f"Consolidated summary statistics saved: {output_dir}/utilization_summary_all.{file_format}")

def plot_consolidated_histograms(data, output_dir, file_format):
    """Create consolidated histograms with all services in subplots."""
    # Count total services
    total_services = sum(len(services) for services in data.values())
    
    if total_services == 0:
        print("No services to plot histograms for.")
        return
    
    # Calculate subplot grid
    n_cols = min(4, total_services)
    n_rows = (total_services + n_cols - 1) // n_cols
    
    # Use canvas for consistent styling
    width = max(10, n_cols * 3)
    fig, axes = create_canvas(nrows=n_rows, ncols=n_cols, width_in_inches=width, aspect_ratio=0.8)
    
    # Handle the axes properly - create_canvas returns different structures based on subplot count
    if n_rows == 1 and n_cols == 1:
        axes_list = [axes]
    elif n_rows == 1 or n_cols == 1:
        # For single row or column, axes might be 1D array
        axes_list = axes.flatten() if hasattr(axes, 'flatten') else axes
        if not isinstance(axes_list, list):
            axes_list = list(axes_list) if hasattr(axes_list, '__iter__') else [axes_list]
    else:
        # For multiple rows and columns
        axes_list = axes.flatten() if hasattr(axes, 'flatten') else axes
        if not isinstance(axes_list, list):
            axes_list = list(axes_list) if hasattr(axes_list, '__iter__') else [axes_list]
    
    # Use canvas colors
    sidecar_names = list(data.keys())
    sidecar_colors = {sidecar: color_list[i % len(color_list)] for i, sidecar in enumerate(sidecar_names)}
    
    idx = 0
    for sidecar_name, services in data.items():
        for service_name, service_data in services.items():
            if idx >= len(axes_list):
                break
                
            ax = axes_list[idx]
            values = service_data["values"]
            display_name = service_name.split('/')[-1] if '/' in service_name else service_name
            
            # Create histogram with better bin handling
            n, bins, patches = ax.hist(values, bins=20, alpha=0.7, edgecolor='black', linewidth=0.5,
                                     color=sidecar_colors[sidecar_name])
            ax.set_title(f"{sidecar_name} - {display_name}", fontweight='bold')
            ax.set_xlabel("Average Req In")
            ax.set_ylabel("Frequency")
            ax.grid(True, alpha=0.3)
            
            # Set clearer x-axis ticks for better bin range visibility
            min_val, max_val = min(values), max(values)
            if max_val - min_val > 0:
                # Create 5-6 evenly spaced ticks
                tick_range = np.linspace(min_val, max_val, 6)
                ax.set_xticks(tick_range)
                ax.set_xticklabels([f'{x:.2f}' for x in tick_range], rotation=45)
            
            # Add statistics text
            stats_text = f"μ={service_data['mean']:.2f}\nσ={service_data['std']:.2f}\nN={service_data['count']}"
            ax.text(0.95, 0.95, stats_text, transform=ax.transAxes, 
                   verticalalignment='top', horizontalalignment='right',
                   bbox=dict(boxstyle='round', facecolor='white', alpha=0.8),
                   fontsize=9)
            
            idx += 1
    
    # Hide empty subplots
    for i in range(total_services, len(axes_list)):
        axes_list[i].set_visible(False)
    
    fig.suptitle("Average Req In Histograms - All Services", fontweight='bold')
    
    # Add legend for sidecars
    legend_elements = [plt.Rectangle((0,0),1,1, facecolor=sidecar_colors[sidecar], alpha=0.7, label=sidecar)
                      for sidecar in sidecar_names]
    fig.legend(handles=legend_elements, loc='upper right', bbox_to_anchor=(0.99, 0.95))
    
    try:
        plt.tight_layout()
        plt.subplots_adjust(top=0.90, right=0.95)
    except:
        # If tight_layout fails, just use subplots_adjust
        plt.subplots_adjust(left=0.1, right=0.95, top=0.90, bottom=0.1)
    
    plt.savefig(f"{output_dir}/utilization_histograms_all.{file_format}", dpi=300, bbox_inches='tight')
    plt.close()
    
    print(f"Consolidated histograms saved: {output_dir}/utilization_histograms_all.{file_format}")

if __name__ == "__main__":
    main()
