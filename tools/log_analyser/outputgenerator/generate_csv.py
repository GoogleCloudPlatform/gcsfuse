import csv
import statistics as stats
import numpy as np
import gspread


def calls_data_writer(obj, call_type, name, worksheet):
    data = []
    for i in range(len(obj)):
        call_data = [name, obj[i].call_name, call_type, obj[i].calls_made, obj[i].calls_returned, obj[i].total_response_time]
        if len(obj[i].response_times) == 0:
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
            call_data.append(0)
        else:
            call_data.append(int(stats.mean(obj[i].response_times)))
            call_data.append(int(stats.median(obj[i].response_times)))
            call_data.append(int(np.percentile(obj[i].response_times, 90)))
            call_data.append(int(max(obj[i].response_times)))
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
        row.append((obj.closing_time - obj.opening_time) + 1e-9*(obj.closing_time_nano - obj.opening_time_nano))
        row.append(obj.opening_time + 1e-9*obj.opening_time_nano)
        row.append(obj.closing_time + 1e-9*obj.closing_time_nano)
        row.append(obj.open_pos)
        row.append(obj.close_pos)
        data.append(row)
    worksheet.append_rows(data)


def read_pattern_writer(global_data, worksheet):
    data = []
    for handle in global_data.handle_name_map.keys():
        name = global_data.handle_name_map[handle]
        obj = global_data.name_object_map[name].handles[handle]
        if len(obj.read_pattern) > 1:
            last_read = obj.read_pattern[1]
            streak = 1
            type_map = {"r": "random", "s": "sequential"}
            for i in range(2, len(obj.read_pattern)):
                if obj.read_pattern[i] != last_read:
                    row = [handle, type_map[last_read], streak]
                    data.append(row)
                    last_read = obj.read_pattern[i]
                    streak = 1
                else:
                    streak += 1
            row = [handle, type_map[last_read], streak]
            data.append(row)
    if len(data) > 0:
        worksheet.append_rows(data)



def main_csv_generator(global_data):
    call_data = [['obj_name', 'call_name', 'call_type', 'calls_sent', 'calls_responded', 'total_response_time', 'average_response_time', 'median_response_time', 'p90_response_time', 'max_response_time']]
    handle_data = [['file_name', 'handle', 'total_reads', 'total_writes', 'average_read_size', 'average_read_response_time', 'average_write_size', 'average_write_response_time', 'total_request_time', 'opening_time', 'closing_time', 'opened_handles', 'closed_handles']]
    pattern_data = [['handle', 'read_type', 'number_of_consecutive_reads']]
    cred_file = input("Enter the credential file (with path): ")
    ldap = input("Enter your ldap: ")
    gc = gspread.service_account(filename=cred_file)
    sheet = gc.create('sample_sheet2')
    sheet.share(ldap + '@google.com', perm_type='user', role='writer')
    worksheet1 = sheet.add_worksheet(title='call_data', rows='1', cols='1')
    worksheet2 = sheet.add_worksheet(title='handle_data', rows='1', cols='1')
    worksheet3 = sheet.add_worksheet(title='read_pattern', rows='1', cols='1')
    try:
        worksheet = sheet.worksheet("Sheet1")
        sheet.del_worksheet(worksheet)
    except gspread.exceptions.WorksheetNotFound:
        pass
    worksheet1.clear()
    worksheet1.append_rows(call_data)
    calls_data_writer(global_data.gcalls.gcs_calls, "GCS", "global", worksheet1)
    calls_data_writer(global_data.gcalls.kernel_calls, "kernel", "global", worksheet1)
    worksheet2.clear()
    worksheet2.append_rows(handle_data)
    handle_data_writer(global_data, worksheet2)
    worksheet3.clear()
    worksheet3.append_rows(pattern_data)
    read_pattern_writer(global_data, worksheet3)
    print(f"Sheet created at: https://docs.google.com/spreadsheets/d/{sheet.id}/edit?usp=sharing")
