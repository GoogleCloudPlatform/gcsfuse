import datetime
from pytz import timezone
import matplotlib.pyplot as plt
from io import BytesIO
import base64


def get_val(message, key, delim, direction, offset):
    # offset contains adjustments needed for spaces and key lengths\
    try:
        if message.find(key) == -1:
            print("Error parsing log with message:", message)
            return None
        if direction == "fwd":
            start_index = message.find(key)+len(key)+offset
        else:
            start_index = message.rfind(key)+len(key)+offset
        if message.find(delim, start_index) == -1:
            print("Error parsing log with message:", message)
            return None
        end_index = message.find(delim, start_index)
        return message[start_index:end_index]
    except ValueError as e:
        print("Error parsing log with message:", message)
        return None


def give_dir_tag(global_data, inode):
    name = global_data.inode_name_map[inode]
    global_data.name_object_map[name].is_dir = True


def get_points(itr, tup, y_axis, curr_time):
    while itr < len(tup) and ((tup[itr][0][0] < curr_time[0]) or ((tup[itr][0][0] == curr_time[0]) and (tup[itr][0][1] < curr_time[1]))):
        itr += 1
    if itr == len(tup) or (tup[itr][0][0] > curr_time[0] or ((tup[itr][0][0] == curr_time[0]) and (tup[itr][0][1] > curr_time[1]))):
        if itr != 0:
            y_axis.append(tup[itr-1][1])
        else:
            y_axis.append(0)
    elif tup[itr][0][0] == curr_time[0] and tup[itr][0][1] == curr_time[1]:
        y_axis.append(tup[itr][1])


def epoch_to_iso(epoch_time):
    int_epoch = int(epoch_time)
    dec_epoch = round(epoch_time - int_epoch, 3)
    datetime_obj = datetime.datetime.fromtimestamp(int_epoch, datetime.timezone.utc)
    ist_tz = timezone('Asia/Kolkata')
    ist_datetime = datetime_obj.astimezone(ist_tz)
    hours = ist_datetime.hour % 24
    minutes = ist_datetime.minute
    seconds = ist_datetime.second
    nanoseconds = int(dec_epoch*1e3)
    iso_time = f"{hours:02d}:{minutes:02d}:{seconds:02d}:{nanoseconds:03d}"
    return iso_time


def iso_to_epoch(timestamp_str):
    try:
        datetime_obj = datetime.datetime.fromisoformat(timestamp_str)
        seconds = int(datetime_obj.timestamp())
        nanos = datetime_obj.microsecond * 1000
        return {"seconds": seconds, "nanos": nanos}
    except ValueError as e:
        print(f"Error parsing timestamp: {e}")
        return None


def plot_pattern(x_axis, y_axis, char_seq):
    color_map = plt.cm.tab10  # Use a colormap for variety
    colors = color_map(range(2))
    bar_width = 0.35  # Adjust bar width as desired
    total_bars = len(y_axis)  # Get total number of bars
    # positions = np.arange(total_bars)  # Create positions using numpy.arange

    # plt.figure(figsize=(8, 6))  # Adjust figure size as desired
    plt.bar(x_axis, y_axis, bar_width, color=colors)
    plt.xlabel("Character Occurrence")
    plt.ylabel("Count")
    plt.title("Individual Character Occurrences in RLE String")
    plt.xticks(x_axis, [f"{char_seq[i]} ({y_axis[i]})" for i in range(len(y_axis))], rotation=45, ha='right')  # Set x-axis labels with char and count
    plt.legend()  # Add legend for color-coded characters
    y_offset = 0.2
    for i, value in enumerate(y_axis):
        plt.text(x_axis[i], value + y_offset, str(value), ha='center', va='bottom', fontsize=10)
    plt.tight_layout()  # Adjust layout to avoid overlapping elements
    # plt.show()
    return plt
def print_in_html(title, html_content, plt, container):
    # html_content.write(f"<h4>{title}</h4>")
    buffer = BytesIO()
    plt.savefig(buffer, format='png', bbox_inches='tight')
    chart_data = base64.b64encode(buffer.getvalue()).decode('ascii')
    if container:
        html_content.write(f"<img src='data:image/png;base64,{chart_data}' alt='{title}' class='graph'>")
    else:
        html_content.write(f"<img src='data:image/png;base64,{chart_data}' alt='{title}'>")
    plt.close()


def update_global_kernel_calls(obj, end_time, start_time):
    obj.calls_returned += 1
    obj.total_response_time += 1e3*((end_time[0] - start_time[0]) + 1e-9*(end_time[1] - start_time[1]))
    obj.response_times.append(1e3*((end_time[0] - start_time[0]) + 1e-9*(end_time[1] - start_time[1])))








