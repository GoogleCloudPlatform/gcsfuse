import csv
from datetime import datetime
import matplotlib.pyplot as plt
import matplotlib.dates as mdates
import os

def generate_gantt_chart_from_csv(csv_file_path):
    """
    Generates a Gantt chart from process event data in a CSV file.

    The CSV file must have the following header:
    Command,Event Type,Event Time Epoch,Event Time Readable

    Args:
        csv_file_path (str): The path to the CSV file.

    Returns:
        matplotlib.figure.Figure: The Matplotlib figure object for the Gantt chart.

    Raises:
        FileNotFoundError: If the CSV file does not exist.
        ValueError: If the CSV is malformed (header, row structure, data types),
                    contains duplicate entries, or has unmatched START/END events.
    """
    events = []
    processed_raw_rows = set()  # For duplicate detection

    if not os.path.exists(csv_file_path):
        raise FileNotFoundError(f"Error: The file '{csv_file_path}' was not found.")
    if os.path.getsize(csv_file_path) == 0:
        raise ValueError("Error: The CSV file is empty.")

    try:
        with open(csv_file_path, 'r', newline='', encoding='utf-8') as infile:
            reader = csv.reader(infile)
            try:
                header = next(reader)
            except StopIteration:
                raise ValueError("Error: CSV file is empty or contains no header.")

            expected_header = ['Command', 'Event Type', 'Event Time Epoch', 'Event Time Readable']
            expected_columns = len(expected_header)

            if header != expected_header:
                # Looser check for just column names if order/case doesn't strictly matter for some use cases
                # For this implementation, we'll be strict.
                raise ValueError(
                    f"CSV header malformed. Expected '{','.join(expected_header)}', "
                    f"got '{','.join(header)}'."
                )

            for i, row in enumerate(reader):
                line_number = i + 2  # 1-based index, +1 for header

                # Check for duplicate raw rows
                row_tuple = tuple(row)
                if row_tuple in processed_raw_rows:
                    raise ValueError(f"Duplicate entry found at line {line_number}: {row}")
                processed_raw_rows.add(row_tuple)

                if len(row) != expected_columns:
                    raise ValueError(
                        f"Malformed entry at line {line_number}: Expected {expected_columns} columns, "
                        f"got {len(row)}. Row: {row}"
                    )

                command, event_type, event_time_epoch_str, event_time_readable = row

                # Validate event_type
                if event_type not in ["START", "END"]:
                    raise ValueError(
                        f"Malformed entry at line {line_number}: Invalid event type '{event_type}'. "
                        f"Must be 'START' or 'END'. Row: {row}"
                    )

                # Validate and convert event_time_epoch
                try:
                    event_time_epoch = float(event_time_epoch_str)
                except ValueError:
                    raise ValueError(
                        f"Malformed entry at line {line_number}: Event time epoch '{event_time_epoch_str}' "
                        f"is not a valid number. Row: {row}"
                    )

                events.append({
                    'command': command,
                    'type': event_type,
                    'time': event_time_epoch,
                    'readable_time': event_time_readable,
                    'line_number': line_number  # Store line number for better error messages later
                })
        
        if not events: # Only header was present
            raise ValueError("CSV file contains only a header and no data rows.")

    except Exception as e:
        # Catch any other potential read errors not already specifically handled
        # Re-raise or wrap if necessary to maintain consistent error type (ValueError often suitable for data issues)
        if not isinstance(e, (FileNotFoundError, ValueError)):
            raise ValueError(f"Error reading or parsing CSV file: {e}")
        else:
            raise # Re-raise the specific FileNotFoundError or ValueError


    # Sort events by time for correct START/END processing
    events.sort(key=lambda x: x['time'])

    tasks = {}  # {command: [{'start':datetime, 'end':datetime, 'duration':float}]}
    open_start_times = {}  # {command_name: [(start_time_epoch, line_number), ...]}
    task_y_pos = {}  # {command_name: y_position_on_chart}
    y_ticks_labels = []
    y_pos_counter = 0

    for event in events:
        command = event['command']
        event_type = event['type']
        event_time = event['time']
        line_number = event['line_number']
        readable_time = event['readable_time']

        if command not in task_y_pos:
            task_y_pos[command] = y_pos_counter
            y_ticks_labels.append(command)
            y_pos_counter += 1

        if event_type == "START":
            if command not in open_start_times:
                open_start_times[command] = []
            open_start_times[command].append((event_time, line_number, readable_time))
        elif event_type == "END":
            if command not in open_start_times or not open_start_times[command]:
                raise ValueError(
                    f"Unmatched END event for command '{command}' at time {readable_time} "
                    f"(line {line_number}). No corresponding START event found or all STARTs "
                    f"for this command were already matched."
                )
            
            # Match with the earliest start time for this command (FIFO)
            start_time_tuple = open_start_times[command].pop(0)
            start_time_epoch = start_time_tuple[0]
            # start_line_number = start_time_tuple[1] # Available if needed

            if event_time < start_time_epoch:
                 raise ValueError(
                    f"END event for command '{command}' at time {readable_time} (line {line_number}) "
                    f"occurs before its matched START event at time "
                    f"{datetime.fromtimestamp(start_time_epoch).strftime('%Y-%m-%d %H:%M:%S')} "
                    f"(line {start_time_tuple[1]}). Ensure events are chronologically consistent or CSV is pre-sorted if pairs are interleaved."
                )

            if command not in tasks:
                tasks[command] = []
            tasks[command].append({
                'start': datetime.fromtimestamp(start_time_epoch),
                'end': datetime.fromtimestamp(event_time),
                'duration': event_time - start_time_epoch
            })

    # After processing all events, check for any unmatched START events
    for command_name, start_tuples_list in open_start_times.items():
        if start_tuples_list:
            first_unmatched_start_tuple = start_tuples_list[0]
            # LIFO for error reporting: first_unmatched_start_tuple = start_tuples_list[0]
            # FIFO for error reporting: first_unmatched_start_tuple = start_tuples_list[0] (already, as we append and would pop from front)
            # The specific start event details are in the tuple
            raise ValueError(
                f"Unmatched START event for command '{command_name}'. "
                f"First unmatched was at {first_unmatched_start_tuple[2]} (line {first_unmatched_start_tuple[1]})."
            )

    if not tasks:
        # This state should ideally not be reached if there were events,
        # as unmatched STARTs/ENDs should have been caught.
        # If events list was populated but tasks is empty, it implies a logic issue
        # or a very specific valid CSV that results in no plottable intervals (e.g. only STARTs, which are errors).
        # Given the rigorous checks, this might mean a valid CSV that just doesn't define any tasks.
        # However, the requirement is to abort on "unmatched start or end".
        # An empty 'tasks' after processing events that should have formed tasks is an error.
        # This case is covered by the unmatched START/END checks.
        # If events was not empty but tasks is, it implies an error caught above.
        # If events was empty (only header), that's caught earlier.
        # So this specific check might be redundant but harmless.
        print("Warning: No plottable tasks were formed. This may indicate an issue if data rows were present but did not form valid pairs.")
        # Consider if an error should be raised here if 'events' was non-empty.
        # For now, the previous checks for unmatched starts/ends should cover this.

    # Plotting
    fig, ax = plt.subplots(figsize=(15, max(6, len(y_ticks_labels) * 0.6))) # Adjusted figsize
    ax.set_title('Process Gantt Chart', fontsize=16)
    ax.set_xlabel('Time', fontsize=12)
    ax.set_ylabel('Commands', fontsize=12)

    # Define a color map or a list of colors
    # Using a colormap like 'viridis' or 'tab20' for more distinct colors
    num_unique_commands = len(y_ticks_labels)
    colors = plt.get_cmap('tab20', num_unique_commands if num_unique_commands > 0 else 1)


    min_time_overall = None
    max_time_overall = None

    for idx, command_name in enumerate(y_ticks_labels):
        if command_name in tasks:
            for task_interval in tasks[command_name]:
                start_dt = task_interval['start']
                end_dt = task_interval['end']

                if min_time_overall is None or start_dt < min_time_overall:
                    min_time_overall = start_dt
                if max_time_overall is None or end_dt > max_time_overall:
                    max_time_overall = end_dt

                ax.barh(y=task_y_pos[command_name],
                        width=(end_dt - start_dt),
                        left=start_dt,
                        height=0.5, # Bar height
                        color=colors(idx % colors.N), # Cycle through colors from cmap
                        edgecolor='grey', # Bar edge color
                        alpha=0.75) # Bar opacity

    if not y_ticks_labels:
        # This case should ideally be prevented by earlier checks if data was expected.
        # If the CSV was valid but genuinely empty of tasks (e.g. only headers, or data that correctly forms no tasks),
        # then an empty plot is the result.
        ax.set_yticks([])
        ax.set_yticklabels([])
        print("Note: The Gantt chart is empty as no valid task intervals were defined in the CSV.")
    else:
        ax.set_yticks(range(len(y_ticks_labels)))
        ax.set_yticklabels(y_ticks_labels, fontsize=10)
        ax.grid(True, which='major', axis='x', linestyle=':', linewidth=0.5, color='gray')
        ax.invert_yaxis() # Commands typically displayed from top to bottom

        # Format x-axis to show readable dates/times
        if min_time_overall and max_time_overall:
            ax.xaxis.set_major_formatter(mdates.DateFormatter('%Y-%m-%d %H:%M:%S'))
            plt.xticks(rotation=30, ha='right', fontsize=10) # Rotate for readability
            
            # Calculate padding based on the total duration
            duration_overall = (max_time_overall - min_time_overall)
            padding = duration_overall * 0.05 # 5% padding on each side
            if padding.total_seconds() == 0: # Handle case with single point in time or very short duration
                padding = timedelta(minutes=1)

            ax.set_xlim(min_time_overall - padding, max_time_overall + padding)
        else: # No tasks, min_time/max_time not set.
            # Set to a default range or leave to matplotlib's default for an empty plot
             pass


    plt.tight_layout(pad=1.5) # Add padding to ensure everything fits
    return fig

if __name__ == '__main__':
    file_path = "tools/integration_tests/command_times.csv"
    try:
        fig = generate_gantt_chart_from_csv(file_path)
        print(f"SUCCESS: Gantt chart generated for {file_path}.")
        # To display the plot:
        # fig.show()
        # input("Press Enter to close plot and continue...") # Pauses for viewing
        # plt.close(fig)
        # To save the plot:
        output_filename = f"tools/integration_tests/gantt_chart.png"
        fig.savefig(output_filename)
        print(f"Saved chart to {output_filename}")
        plt.close(fig) # Close the figure to free memory
    except (ValueError, FileNotFoundError) as e:
        print(f"EXPECTED ERROR for {file_path}: {e}")
    except Exception as e:
        print(f"UNEXPECTED ERROR for {file_path}: {type(e).__name__} - {e}")
    