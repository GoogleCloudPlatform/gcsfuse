import csv
import statistics as stats
import numpy as np
import gspread
import heapq
import json


def get_top_file_gcscalls_stats(global_data):
    top_file_num = []
    top_file_response_time = []
    top_files = [[], [], [], [], [], [], [], []]
    for name in global_data.name_object_map.keys():
        obj = global_data.name_object_map[name].gcs_calls.calls
        for i in range(8):
            heapq.heappush(top_files[i], (obj[i].calls_returned, name))
            if len(top_files[i]) > 5:
                heapq.heappop(top_files[i])

    for i in range(8):
        total_num = 0
        total_time = 0
        for j in range(len(top_files[i])):
            total_num += top_files[i][j][0]
            total_time += global_data.name_object_map[top_files[i][j][1]].gcs_calls.calls[i].total_response_time
        top_file_num.append(total_num)
        top_file_response_time.append(total_time)
        # print(total_num, total_time)

    return top_file_num, top_file_response_time


def calls_data_writer(global_data, obj, call_type, name, worksheet):
    data = []
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
        if call_type == "GCS":
            top_file_num, top_file_response_time = get_top_file_gcscalls_stats(global_data)
            call_data.append(top_file_num[i])
            call_data.append(top_file_response_time[i])
        else:
            call_data.append(0)
            call_data.append(0)
        data.append(call_data)
    worksheet.append_rows(data)


def handle_data_writer(global_data, worksheet):
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
    if len(obj_pattern) > 0:
        data.append([handle, op, "first " + op, 1, obj_bytes_list[0], obj_bytes_list[0], json.dumps([obj_bytes_list[0]]), json.dumps([obj_ranges_list[0]])])
    if len(obj_pattern) > 1:
        read_ranges = [obj_ranges_list[1]]
        read_bytes = [obj_bytes_list[1]]
        byte_sum = obj_bytes_list[1]
        last_read = obj_pattern[1]
        streak = 1
        type_map = {"r": "random", "s": "sequential"}
        for i in range(2, len(obj_pattern)):
            if obj_pattern[i] != last_read:
                json_read_bytes = json.dumps(read_bytes)
                json_read_ranges = json.dumps(read_ranges)
                avg_byte_size = np.mean(obj_bytes_list)
                row = [handle, op, type_map[last_read], streak, avg_byte_size, byte_sum, json_read_bytes, json_read_ranges]
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
        data.append(row)


def read_pattern_writer(global_data, worksheet):
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
    call_data = [['obj_name', 'call_name', 'call_type', 'calls_sent', 'calls_responded', 'total_response_time', 'average_response_time', 'median_response_time', 'p90_response_time', 'p95_response_time', 'max_response_time', 'top_file_num', 'top_file_response_time']]
    handle_data = [['file_name', 'handle', 'total_reads', 'total_writes', 'average_read_size', 'average_read_response_time', 'average_write_size', 'average_write_response_time', 'total_request_time', 'opening_time', 'closing_time', 'opened_handles', 'closed_handles']]
    pattern_data = [['handle', 'op', 'op_type', 'number_of_consecutive_ops', 'mean_op_size', 'total_op_size', 'op_bytes', 'op_range']]
    max_entry_data = [['call_type', 'response_time', 'object_name']]
    cred_file = input("Enter the credential file (with path): ")
    ldap = input("Enter your ldap: ")
    gc = gspread.service_account(filename=cred_file)
    sheet = gc.create('sample_sheet2')
    worksheet1 = sheet.add_worksheet(title='call_data', rows='1', cols='1')
    worksheet2 = sheet.add_worksheet(title='handle_data', rows='1', cols='1')
    worksheet3 = sheet.add_worksheet(title='read_pattern', rows='1', cols='1')
    worksheet4 = sheet.add_worksheet(title='max_entries', rows='1', cols='1')
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
    sheet.share(ldap + '@google.com', perm_type='user', role='writer')
    print(f"Sheet created at: https://docs.google.com/spreadsheets/d/{sheet.id}/edit?usp=sharing")
