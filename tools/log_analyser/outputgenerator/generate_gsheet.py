import statistics as stats
import numpy as np
import gspread
import heapq
import json


def get_top_file_gcscalls_stats(global_data, call_type):
    """
    it writes the top file stats for some calls (all GCS calls, 10 kernel calls)

    checks whether it is GCS or kernel and fixes the length of iterations
    then it maintains a priority queue of size 5 (because we need top 5 files)
    for each file we push an entry corresponding to it and then check if the size is greater than 5
    if yes we remove the entry with least calls completed
    :param global_data: containing all the analyzed data
    :param call_type: GCS/kernel
    :return:returns a list of lists containing tuples of the form [calls completed, response time, file/dir name]
    response time for kernel calls is 0, as we are not maintaining that
    """
    top_file_num = []
    top_files = []
    if call_type == "GCS":
        call_len = 8
    else:
        call_len = 10
    for i in range(call_len):
        top_files.append([])
    for name in global_data.name_object_map.keys():
        if call_type == "GCS":
            obj = global_data.name_object_map[name].gcs_calls.calls
        else:
            obj = global_data.name_object_map[name].kernel_calls.calls
        for i in range(call_len):
            heapq.heappush(top_files[i], (obj[i].calls_returned, int(obj[i].total_response_time), name))
            if len(top_files[i]) > 5:
                heapq.heappop(top_files[i])

    for i in range(call_len):
        file_list = []
        while len(top_files[i]) > 0:
            tup = heapq.heappop(top_files[i])
            if tup[0] != 0:
                file_list.append(tup)
        top_file_num.append(json.dumps(list(reversed(file_list))))

    return top_file_num


def calls_data_writer(global_data, obj, call_type, name, worksheet):
    """
    writes the relevant calls data to the google sheet,
    calls the get_top_file_gcscalls_stats function
    :param global_data: contains all the analyzed information
    :param obj: list containing the objects of Calls type
    :param call_type: GCS/kernel
    :param name: file/dir name or maybe global flag
    :param worksheet: the worksheet to which the data is to be written
    """
    data = []
    top_file_num = get_top_file_gcscalls_stats(global_data, call_type)
    for i in range(len(obj)):
        call_data = [name, obj[i].call_name, call_type, obj[i].calls_made, obj[i].calls_returned, obj[i].total_response_time/1e3]
        if len(obj[i].response_times) == 0:
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
        else:
            call_data.append(int(stats.mean(obj[i].response_times)))
            call_data.append(int(stats.median(obj[i].response_times)))
            call_data.append(int(np.percentile(obj[i].response_times, 90)))
            call_data.append(int(np.percentile(obj[i].response_times, 95)))
            call_data.append(int(max(obj[i].response_times)))
        if call_type == "kernel" and i > 5:
            call_data.append(top_file_num[i-6])
        elif call_type == "GCS":
            call_data.append(top_file_num[i])
        else:
            call_data.append(json.dumps([]))
        data.append(call_data)
    worksheet.append_rows(data)


def handle_data_writer(global_data, worksheet):
    """
    iterates over all the registered file handle and writes their information
    :param global_data: all analyzed data
    :param worksheet: worksheet in which the data is to be written
    """
    data = []
    for handle in global_data.handle_name_map.keys():
        name = global_data.handle_name_map[handle]
        obj = global_data.name_object_map[name].handles[handle]
        row = [name, handle, obj.total_reads, obj.total_writes]
        if obj.total_reads != 0:
            row.append(float(obj.total_read_size/obj.total_reads))
            row.append(stats.mean(obj.read_times))
        else:
            row.append(0)
            row.append(0)
        if obj.total_writes != 0:
            row.append(float(obj.total_write_size/obj.total_writes))
            row.append(stats.mean(obj.write_times))
        else:
            row.append(0)
            row.append(0)
        row.append(1e3*((obj.closing_time - obj.opening_time) + 1e-9*(obj.closing_time_nano - obj.opening_time_nano)))
        row.append(obj.opening_time + 1e-9*obj.opening_time_nano)
        row.append(obj.closing_time + 1e-9*obj.closing_time_nano)
        row.append(obj.open_pos)
        row.append(obj.close_pos)
        data.append(row)
    worksheet.append_rows(data)


def get_pattern_rows(handle, data, obj_bytes_list, obj_ranges_list, obj_pattern, op):
    """
    it creates rows for read/write patterns for a file handle
    :param handle:for which rows are to be created
    :param data: a list containing all the rows to be written in a worksheet
    :param obj_bytes_list: contains all the bytes that were read/written
    :param obj_ranges_list: contains intervals of write/read
    :param obj_pattern: patterns for read/write
    :param op:read/write
    """
    if len(obj_pattern) > 0:
        data.append([handle, op, "first " + op, 1, obj_bytes_list[0], obj_bytes_list[0], json.dumps([obj_bytes_list[0]]), json.dumps([obj_ranges_list[0]])])
        # data.append([handle, op, "first " + op, 1, obj_bytes_list[0], obj_bytes_list[0]])
    if len(obj_pattern) > 1:
        read_ranges = [obj_ranges_list[1]]
        read_bytes = [obj_bytes_list[1]]
        byte_sum = obj_bytes_list[1]
        last_read = obj_pattern[1]
        streak = 1
        type_map = {"r": "random", "s": "sequential"}
        for i in range(2, len(obj_pattern)):
            if obj_pattern[i] != last_read or streak == 500:
                json_read_bytes = json.dumps(read_bytes)
                json_read_ranges = json.dumps(read_ranges)
                avg_byte_size = np.mean(read_bytes)
                row = [handle, op, type_map[last_read], streak, avg_byte_size, byte_sum, json_read_bytes, json_read_ranges]
                # row = [handle, op, type_map[last_read], streak, avg_byte_size, byte_sum]
                data.append(row)
                last_read = obj_pattern[i]
                streak = 1
                read_bytes.clear()
                read_ranges.clear()
                read_bytes.append(obj_bytes_list[i])
                read_ranges.append(obj_ranges_list[i])
                byte_sum = obj_bytes_list[i]
            else:
                streak += 1
                byte_sum += obj_bytes_list[i]
                read_bytes.append(obj_bytes_list[i])
                read_ranges.append(obj_ranges_list[i])
        json_read_bytes = json.dumps(read_bytes)
        json_read_ranges = json.dumps(read_ranges)
        avg_byte_size = np.mean(obj_bytes_list)
        row = [handle, op, type_map[last_read], streak, avg_byte_size, byte_sum, json_read_bytes, json_read_ranges]
        # row = [handle, op, type_map[last_read], streak, avg_byte_size, byte_sum]
        data.append(row)


def read_pattern_writer(global_data, worksheet):
    """
    it calls the get_pattern_rows with appropriate arguments
    and then writes to the google sheet
    """
    data = []
    for handle in global_data.handle_name_map.keys():
        name = global_data.handle_name_map[handle]
        obj = global_data.name_object_map[name].handles[handle]
        get_pattern_rows(str(handle), data, obj.read_bytes, obj.read_ranges, obj.read_pattern, "read")
        get_pattern_rows(str(handle), data, obj.write_bytes, obj.write_ranges, obj.write_pattern, "write")

    for name in global_data.name_object_map.keys():
        obj = global_data.name_object_map[name]
        get_pattern_rows(name, data, obj.read_bytes, obj.read_ranges, obj.read_pattern, "read")
    if len(data) > 0:
        worksheet.append_rows(data)


def max_entry_writer(global_data, worksheet):
    """
    writes entries with max response times along with the file/dir name for some GCS calls
    """
    data = []
    while len(global_data.max_read_entries) > 0:
        response_time, obj_name = heapq.heappop(global_data.max_read_entries)
        row = ["Read", response_time, obj_name]
        data.append(row)
    while len(global_data.max_listobjects_entries) > 0:
        response_time, obj_name = heapq.heappop(global_data.max_listobjects_entries)
        row = ["ListObjects", response_time, obj_name]
        data.append(row)
    while len(global_data.max_createobject_entries) > 0:
        response_time, obj_name = heapq.heappop(global_data.max_createobject_entries)
        row = ["CreateObject", response_time, obj_name]
        data.append(row)
    worksheet.append_rows(data)


def main_gsheet_generator(global_data):
    """
    creates a google sheet after taking a credential file and ldap
    calls appropriate functions for writing data, shares the sheet with user over mail
    and also outputs the link
    :param global_data: all the analyzed
    """
    call_data = [['obj_name', 'call_name', 'call_type', 'calls_sent', 'calls_responded', 'total_response_time', 'average_response_time', 'median_response_time', 'p90_response_time', 'p95_response_time', 'max_response_time', 'top_file_num']]
    handle_data = [['file_name', 'handle', 'total_reads', 'total_writes', 'average_read_size', 'average_read_response_time', 'average_write_size', 'average_write_response_time', 'total_request_time', 'opening_time', 'closing_time', 'opened_handles', 'closed_handles']]
    pattern_data = [['handle', 'op', 'op_type', 'number_of_consecutive_ops', 'mean_op_size', 'total_op_size', 'op_bytes', 'op_range']]
    max_entry_data = [['call_type', 'response_time', 'object_name']]
    cred_file = input("Enter the credential file (with path): ")
    ldap = input("Enter your ldap (to give access to the sheet generated): ")
    gc = gspread.service_account(filename=cred_file)
    sheet = gc.create('sample_sheet2')
    worksheet1 = sheet.add_worksheet(title='call_data', rows='1', cols='1')
    worksheet2 = sheet.add_worksheet(title='handle_data', rows='1', cols='1')
    worksheet3 = sheet.add_worksheet(title='read_pattern', rows='1', cols='1')
    worksheet4 = sheet.add_worksheet(title='max_entries', rows='1', cols='1')
    worksheet5 = sheet.add_worksheet(title ='faulty logs', rows='1',cols='1')
    try:
        worksheet = sheet.worksheet("Sheet1")
        sheet.del_worksheet(worksheet)
    except gspread.exceptions.WorksheetNotFound:
        pass
    worksheet1.clear()
    worksheet1.append_rows(call_data)
    calls_data_writer(global_data, global_data.gcalls.gcs_calls, "GCS", "global", worksheet1)
    calls_data_writer(global_data, global_data.gcalls.kernel_calls, "kernel", "global", worksheet1)
    worksheet2.clear()
    worksheet2.append_rows(handle_data)
    handle_data_writer(global_data, worksheet2)
    worksheet3.clear()
    worksheet3.append_rows(pattern_data)
    read_pattern_writer(global_data, worksheet3)
    worksheet4.clear()
    worksheet4.append_rows(max_entry_data)
    max_entry_writer(global_data, worksheet4)
    worksheet5.clear()
    worksheet5.append_row(["message"])
    log_rows = []
    for log in global_data.faulty_logs:
        log_rows.append([log])
    worksheet5.append_rows(log_rows)
    sheet.share(ldap + '@google.com', perm_type='user', role='writer')
    print(f"Sheet created at: https://docs.google.com/spreadsheets/d/{sheet.id}/edit?usp=sharing")
