import utility
import matplotlib.pyplot as plt
import statistics as stats
from tabulate import tabulate
import webbrowser
from io import StringIO
import base64
from io import  BytesIO
import math


def time_series_open_close(open_tup, close_tup, html_content):
    x_axis = []
    y_axis = []
    last_time = open_tup[0][0][0]
    for i in range(1, len(open_tup)):
        if open_tup[i][0][0] != last_time:
            x_axis.append(open_tup[i-1][0][0])
            y_axis.append(open_tup[i-1][1])
            last_time = open_tup[i][0][0]
    x_axis.append(open_tup[len(open_tup)-1][0][0])
    y_axis.append(open_tup[len(open_tup)-1][1])
    plt.figure(figsize=(15, 6))
    plt.xlabel('time')
    plt.ylabel('num of handles')
    plt.title('number of handles opened time series')
    plt.plot(x_axis, y_axis, marker='s', markersize=3, markerfacecolor='red')
    plt.xticks(x_axis, rotation=90, fontsize=6)
    for i, value in enumerate(y_axis):
        plt.text(x_axis[i], value+0.1, str(value), ha='center', va='bottom', fontsize=6)
    buffer = BytesIO()
    plt.savefig(buffer, format='png', bbox_inches='tight')
    chart_data = base64.b64encode(buffer.getvalue()).decode('ascii')
    html_content.write(f"<img src='data:image/png;base64,{chart_data}'>")
    plt.close()

    x_axis.clear()
    y_axis.clear()
    last_time = close_tup[0][0][0]
    for i in range(1, len(close_tup)):
        if open_tup[i][0][0] != last_time:
            x_axis.append(close_tup[i-1][0][0])
            y_axis.append(close_tup[i-1][1])
            last_time = close_tup[i][0][0]
    x_axis.append(close_tup[len(close_tup)-1][0][0])
    y_axis.append(close_tup[len(close_tup)-1][1])
    plt.figure(figsize=(15, 6))
    plt.xlabel('time')
    plt.ylabel('num of handles')
    plt.title('number of handles closed time series')
    plt.plot(x_axis, y_axis, marker='s', markersize=3, markerfacecolor='red')
    plt.xticks(x_axis, rotation=90, fontsize=6)
    for i, value in enumerate(y_axis):
        plt.text(x_axis[i], value+0.1, str(value), ha='center', va='bottom', fontsize=6)
    buffer = BytesIO()
    plt.savefig(buffer, format='png', bbox_inches='tight')
    chart_data = base64.b64encode(buffer.getvalue()).decode('ascii')
    html_content.write(f"<img src='data:image/png;base64,{chart_data}'>")
    plt.close()
    # utility.print_in_html("", html_content, plt, False)


def open_close_handles(interval, freq, unit, open_tup, close_tup, html_content):
    title = "Handle opening closing:"
    html_content.write(f"<h3>{title}</title></head><body>")
    start_time = interval[0]
    end_time = interval[1]
    itr1 =0
    itr2 =0
    y_open = []
    y_close = []
    x_axis = []
    x_labels = []
    for i in range(len(open_tup)):
        if open_tup[i][0][0] > start_time[0]:
            itr1 = i
            break
        elif open_tup[i][0][0] == start_time[0] and open_tup[i][0][1] > start_time[1]:
            itr1 = i
            break
    for i in range(len(close_tup)):
        if close_tup[i][0][0] > start_time[0]:
            itr2 = i
            break
        elif close_tup[i][0][0] == start_time[0] and close_tup[i][0][1] > start_time[1]:
            itr2 = i
            break
    curr_time = start_time
    pos = 1
    while curr_time[0] < end_time[0] or (curr_time[0] == end_time[0] and curr_time[1] <= end_time[1]):
        # x_axis.append(curr_time[0] + 1e-9*curr_time[1])
        x_axis.append(pos)
        pos += 1
        x_labels.append(utility.epoch_to_iso(curr_time[0]+1e-9*curr_time[1]))
        utility.get_points(itr1, open_tup, y_open, curr_time)
        utility.get_points(itr2, close_tup, y_close, curr_time)
        if unit == "s":
            curr_time[0] += freq
        else:
            curr_time[1] += freq
            if curr_time[1] >= 1e9:
                curr_time[0] += 1
                curr_time[1] -= 1e9
    if unit == "ms":
        freq = freq*1e-9
    bar_offset = 0.4

    plt.bar(x_axis, y_open, width=0.4, label='opened handles')
    plt.bar([float(pos) + bar_offset for pos in x_axis], y_close, width=0.4, label='closed handles')

    plt.xlabel('time')
    plt.ylabel('Number of handles')
    plt.title('Opened and closed handles')
    plt.xticks([x_axis[i] + bar_offset / 2 for i in range(len(x_axis))], x_labels, rotation=75, fontsize=7)  # Adjust x-axis tick positions
    plt.legend()
    plt.ylim(0, max(y_open[len(y_open)-1]+2, 5))
    bar_width = 0.4
    for i, value in enumerate(y_open):
        plt.text(x_axis[i], value, str(value), ha='center', va='bottom', fontsize=6)
    for i, value in enumerate(y_close):
        plt.text(x_axis[i] + bar_width, value, str(value), ha='center', va='bottom', fontsize=6)
    utility.print_in_html("", html_content, plt, False)
    # plt.show()



def gcs_calls_plot(gcs_calls_obj, html_content):
    x_axis = []
    x_labels = []
    y_made = []
    y_ret = []
    y_response = []
    y_response_avg = []
    y_response_p10 = []
    y_response_p50 = []
    y_response_max = []
    # fig, ((ax1, ax2), (ax3, ax4)) = plt.subplots(2, 2, figsize=(10, 6))
    # fig, (ax1, ax2) = plt.subplots(1, 2)
    for i in range(8):
        x_axis.append(2*i + 1)
        x_labels.append(gcs_calls_obj.calls[i].call_name)
        y_made.append(gcs_calls_obj.calls[i].calls_made)
        y_ret.append(gcs_calls_obj.calls[i].calls_returned)
        y_response.append(int(gcs_calls_obj.calls[i].total_response_time))
        if len(gcs_calls_obj.calls[i].response_times) == 0:
            y_response_avg.append(0)
            y_response_p50.append(0)
            y_response_p10.append(0)
            y_response_max.append(0)
        else:
            y_response_avg.append(int(stats.mean(gcs_calls_obj.calls[i].response_times)))
            y_response_p50.append(int(stats.median(gcs_calls_obj.calls[i].response_times)))
            y_response_max.append(int(max(gcs_calls_obj.calls[i].response_times)))
            sorted_list = sorted(gcs_calls_obj.calls[i].response_times)
            index_p10 = math.ceil(0.9*len(sorted_list)) - 1
            # index_p5 = ceil(0.95*len(sorted_list))
            y_response_p10.append(int(sorted_list[index_p10]))
            # y_response_p5.append(gcs_calls_obj.calls[i].response_times[index_p5])

    html_content.write("<div class='graph-container'>")

    bar_width = 0.7
    plt.bar(x_axis, y_made, width=bar_width, label='sent')
    plt.bar([float(pos) + bar_width for pos in x_axis], y_ret, width=bar_width, label='responded')
    plt.xlabel('call type')
    plt.ylabel('Number of calls')
    plt.title('Distribution of GCS calls')
    plt.xticks([x_axis[i] + bar_width / 2 for i in range(len(x_axis))], x_labels, rotation=45)  # Adjust x-axis tick positions
    plt.legend()
    for i, value in enumerate(y_made):
        plt.text(x_axis[i], value, str(value), ha='center', va='bottom', fontsize=6)
    for i, value in enumerate(y_ret):
        plt.text(x_axis[i] + bar_width, value, str(value), ha='center', va='bottom', fontsize=6)
    utility.print_in_html("distribution of gcs calls", html_content, plt, True)

    plt.bar(x_axis, y_response, width=bar_width)
    plt.xlabel('call type')
    plt.ylabel('total time')
    plt.title('Distribution of response time for GCS calls')
    plt.xticks(x_axis, x_labels, rotation=45)
    for i, value in enumerate(y_response):
        plt.text(x_axis[i], value, str(value), ha='center', va='bottom', fontsize=6)
    # ax2.legend()
    utility.print_in_html("distribution of gcs calls", html_content, plt, True)

    html_content.write("</div>")
    html_content.write("<br><br>")

    html_content.write("<div class='graph-container'>")

    bar_width = 0.7
    plt.bar([x_axis[i] for i in range(len(x_axis))], y_response_avg, width=bar_width, label='average', color='blue')
    plt.bar([x_axis[i] + bar_width for i in range(len(x_axis))], y_response_p50, width=bar_width, label='median(p50)', color='green')
    plt.xlabel('call type')
    plt.ylabel('time(ms)')
    plt.title('avg, p50 of response times for gcs calls')
    plt.legend()
    plt.xticks([x_axis[i] + bar_width/2 for i in range(len(x_axis))], x_labels, rotation=45)
    y_offset = 0.0  # Adjust offset for better positioning
    for i, value in enumerate(y_response_avg):
        plt.text(x_axis[i], value + y_offset, str(value), ha='center', va='bottom', fontsize=6)

    for i, value in enumerate(y_response_p50):
        plt.text(x_axis[i] + bar_width, value + y_offset, str(value), ha='center', va='bottom', fontsize=6)

    utility.print_in_html("distribution of gcs calls", html_content, plt, True)

    plt.bar([x_axis[i] for i in range(len(x_axis))], y_response_p10, width=bar_width, label='p10', color='orange')
    plt.bar([x_axis[i] + bar_width for i in range(len(x_axis))], y_response_max, width=bar_width, label='max', color='blue')
    plt.xlabel('call type')
    plt.ylabel('time(ms)')
    plt.title('p10, max of response times for gcs calls')
    plt.legend()
    plt.xticks([x_axis[i] + bar_width/2 for i in range(len(x_axis))], x_labels, rotation=45)
    for i, value in enumerate(y_response_p10):
        plt.text(x_axis[i], value + y_offset, str(value), ha='center', va='bottom', fontsize=6)
    for i, value in enumerate(y_response_max):
        plt.text(x_axis[i] + bar_width, value + y_offset, str(value), ha='center', va='bottom', fontsize=6)
    utility.print_in_html("distribution of gcs calls", html_content, plt, True)
    html_content.write("</div>")



def bytes_to_fro_gcs(bytes_to, bytes_from, html_content):
    # Sample data (replace with your actual data)
    data = [bytes_to, bytes_from]  # List of data values for each slice
    labels = ['bytes to gcs', 'bytes from gcs']  # List of labels for each slice

    # Create the pie chart with actual values
    plt.pie(data, labels=labels, autopct=lambda pct: f"{int(pct * sum(data) / 100)}")

    # Customize the plot (optional)
    # plt.title('')
    # Display the plot
    # plt.show()


    title = "Bytes to/from GCS"
    html_content.write(f"<h4>{title}</h4>")
    buffer = BytesIO()
    plt.savefig(buffer, format='png', bbox_inches='tight')
    chart_data = base64.b64encode(buffer.getvalue()).decode('ascii')
    html_content.write(f"<img src='data:image/png;base64,{chart_data}' alt='{title}' width='300' height='300'>")
    plt.close()

    # Add chart image to HTML
    # html_content.write(f"<img src='data:image/png;base64,{chart_image_data}' alt='{chart_data['title']}' width='500' height='300'>")


def kernel_calls_plot(kernel_calls_obj, html_content):
    x_axis = []
    y_axis = []
    x_labels = []
    total = len(kernel_calls_obj.calls)
    for i in range(total):
        x_axis.append(i)
        x_labels.append(kernel_calls_obj.calls[i].call_name)
        y_axis.append(kernel_calls_obj.calls[i].calls_made)

    bar_offset = 0.4
    # plt.figure(figsize=(10, 8))
    plt.bar(x_axis, y_axis, width=0.4, label='made')
    # plt.bar([float(pos) + bar_offset for pos in x_axis], y_ret, width=0.4, label='returned')
    plt.xlabel('call type')
    plt.ylabel('Number of calls')
    plt.title('Distribution of kernel calls')
    plt.xticks([x_axis[i] for i in range(len(x_axis))], x_labels, rotation=45)  # Adjust x-axis tick positions
    # plt.legend()
    # y_offset = 0.0
    for i, value in enumerate(y_axis):
        plt.text(x_axis[i], value, str(value), ha='center', va='bottom', fontsize=10)
    # plt.show()

    title = "kernel calls"
    html_content.write(f"<h4>{title}</h4>")
    buffer = BytesIO()
    plt.savefig(buffer, format='png', bbox_inches='tight')
    chart_data = base64.b64encode(buffer.getvalue()).decode('ascii')
    html_content.write(f"<img src='data:image/png;base64,{chart_data}' alt='{title}' width='500' height='500'>")
    plt.close()


def operation_breakdown(handles, html_content):
    for val in handles:
        obj = handles[val]
        data = [["Opening time", round(obj.opening_time + 1e-9*obj.opening_time_nano, 3)], ["Closing time", round(obj.closing_time + 1e-9*obj.closing_time_nano, 3)], ["Total reads", obj.total_reads], ["Total writes", obj.total_writes], ["Total request time", str(round(obj.closing_time - obj.opening_time + 1e-9*(obj.closing_time_nano- obj.opening_time_nano), 3)) + "sec"]]
        if obj.total_reads != 0:
            data.append(["avg read size", str(round(obj.total_read_size/obj.total_reads, 3)) + "bytes"])
            data.append(["avg read response time", str(round(stats.mean(obj.read_times), 6)) + "sec"])
        if obj.total_writes != 0:
            data.append(["avg write size", str(round(obj.total_write_size/obj.total_writes, 3)) + "bytes"])
            data.append(["avg write response time", str(round(stats.mean(obj.write_times), 6)) + "sec"])

        # Create table string using tabulate
        if obj.total_reads != 0 or obj.total_writes != 0:
            # table_string = tabulate(data, tablefmt="fancy_grid")
            title = "Handle " + str(val) + ":"
            html_content.write(f"<h4>{title}</title></head><body>")
            if len(obj.read_pattern) > 1:
                html_content.write("<div class='graph-container'>")
            html_table_content = "<table style='height: 200px;'>"
            for row in data:
                html_table_content += "<tr>"
                for cell in row:
                    html_table_content += f"<td>{cell}</td>"
                html_table_content += "</tr>"
            html_table_content += "</table>"
            html_content.write(html_table_content)
        x_axis = []
        y_axis = []
        char_seq = []
        num_of_bars = 0
        bar_width = 0.4
        if len(obj.read_pattern) > 1:
            last_read = obj.read_pattern[1]
            streak = 1
            for i in range(2, len(obj.read_pattern)):
                if obj.read_pattern[i] != last_read:
                    # print(last_read, streak, end="\t")
                    num_of_bars += 1
                    x_axis.append(num_of_bars*bar_width)
                    y_axis.append(streak)
                    char_seq.append(last_read)

                    last_read = obj.read_pattern[i]
                    streak = 1
                else:
                    streak += 1

            # print(last_read, streak)
            num_of_bars += 1
            x_axis.append(num_of_bars*bar_width)
            y_axis.append(streak)
            char_seq.append(last_read)
            if len(y_axis):
                plot = utility.plot_pattern(x_axis, y_axis, char_seq)
                plot.xlim(0, max(num_of_bars*bar_width+1, 10*bar_width))
                utility.print_in_html("", html_content, plot, True)
            html_content.write("</div>")



def gen_output(global_data):
    # Generate HTML content
    html_content = StringIO()
    title = "Log Analysis Report"
    # html_content.write(f"<html><head><title>{title}</title></head><body>")
    html_content.write(f"""<html><head><title>{title}</title>
  <style>
    .graph-container {{
      display: flex;
      justify-content: space-around; /* Optional: Adjust horizontal spacing */
    }}
    .graph {{
      width: 40%; /* Adjust width as needed */
    }}
    .graph-and-table-container {{
    display: flex;
    justify-content: space-around;
    }}
  </style>
  </head><body>""")
    html_content.write(f"<h1>{title}</h1>")
    title = "Global Data:"
    html_content.write(f"<h2>{title}</title></head><body>")

    bytes_to_fro_gcs(global_data.bytes_to_gcs, global_data.bytes_from_gcs, html_content)
    kernel_calls_plot(global_data.kernel_calls, html_content)
    title = "File Specific Data:"
    html_content.write(f"<h2>{title}</title></head><body>")
    file = input("Enter file for which you want open handles graph:")
    title = "Filename-" + file + ":"
    html_content.write(f"<h2>{title}</title></head><body>")

    # file = input("Enter file for which you want distribution of gcs calls:")
    gcs_calls_plot(global_data.name_object_map[file].gcs_calls, html_content)
    kernel_calls_plot(global_data.name_object_map[file].kernel_calls, html_content)
    html_content.write("<br><br>")
    # interval = [[0, 0], [0, 0]]
    # interval[0][0] = int(input("Enter start time (sec):"))
    # interval[0][1] = int(input("Enter start time (ms):"))
    # interval[1][0] = int(input("Enter end time (sec):"))
    # interval[1][1] = int(input("Enter end time (ms):"))
    # unit = input("Enter the unit of time for period of bars (ms/s):")
    # freq = float(input("Enter time period for bars:"))
    # if unit == "ms":
    #     freq = freq*1e6
    # open_close_handles(interval, freq, unit, global_data.name_object_map[file].open_tup, global_data.name_object_map[file].close_tup, html_content)
    time_series_open_close(global_data.name_object_map[file].open_tup, global_data.name_object_map[file].close_tup, html_content)
    title = "Handle Specific Data:"
    html_content.write(f"<h3>{title}</title></head><body>")

    operation_breakdown(global_data.name_object_map[file].handles, html_content)


    html_content.write("</body></html>")
    report_html = html_content.getvalue()
    report_path = "/usr/local/google/home/patelvishvesh/tmp/log_analysis_report.html"
    with open(report_path, "w") as f:
        f.write(report_html)
    webbrowser.open(f"file://{report_path}")  # Open the local HTML file



