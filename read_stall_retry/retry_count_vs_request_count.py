import pandas as pd
import logging
import time

# Run read_stall_retry/stalled_read_req_retry_logs.sh before running this script to get the log files

# Move to the directory that has log files
os.chdir(os.path.expanduser('~/vipinydv-redstall-logs'))

# Set up logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s')
logger = logging.getLogger()

# Define a list of filenames
filenames = ['fastenvironment-readstall-genericread-logs', 'fastenvironment-readstall-filecache-logs', 'fastenvironment-readstall-paralleldownload-logs']  # Replace with your list of filenames (without extension)

# Initialize an empty list to store the request codes
request_codes = []

# Loop through each filename in the list
for filename in filenames:
    logger.info(f"Processing file: {filename}.csv")

    # Initialize the request codes list for each file
    request_codes.clear()

    # Read the file in chunks and log time taken
    chunk_size = 10000  # You can experiment with chunk size
    start_time = time.time()  # Start time for file reading
    logger.info(f"Starting to read the file {filename}.csv in chunks...")

    for chunk in pd.read_csv(f'{filename}.csv', header=None, names=['timestamp', 'log_message'], chunksize=chunk_size):
        chunk_time_start = time.time()  # Time at start of each chunk
        logger.info(f"Processing a chunk of {len(chunk)} rows.")
        
        # Extract request codes in chunks and log the time
        mask_start_time = time.time()  # Start time for mask application
        mask = chunk['log_message'].str.contains(r'\(0x[0-9a-fA-F]+\)', regex=True)
        logger.info(f"Time taken for mask application: {time.time() - mask_start_time:.2f} seconds.")
        
        extract_start_time = time.time()  # Start time for regex extraction
        chunk['request_code'] = chunk.loc[mask, 'log_message'].str.extract(r'\(0x([0-9a-fA-F]+)\)', expand=False)
        logger.info(f"Time taken for regex extraction: {time.time() - extract_start_time:.2f} seconds.")
        
        # Append the extracted request codes to the list and log
        request_codes.extend(chunk['request_code'].dropna())
        logger.info(f"Time taken for appending request codes: {time.time() - chunk_time_start:.2f} seconds.")

    logger.info(f"Total time for reading and processing {filename}: {time.time() - start_time:.2f} seconds.")

    # Count occurrences of each request code
    count_start_time = time.time()  # Start time for counting occurrences
    request_code_counts = pd.Series(request_codes).value_counts()
    logger.info(f"Time taken for counting request codes: {time.time() - count_start_time:.2f} seconds.")

    # Count how many request codes have the same frequency
    frequency_counts = request_code_counts.value_counts().sort_index()

    # Log the distinct frequencies
    logger.info(f"Distinct frequencies (number of unique counts): {len(frequency_counts)}")

    # Save the frequency counts to a CSV file
    output_filename = f'{filename}-generated.csv'
    
    # Convert frequency_counts to DataFrame and reset index properly
    frequency_counts_df = frequency_counts.reset_index(name='retry_count')  # Ensure column name for frequency count is unique
    frequency_counts_df.columns = ['retry_count', 'num_requests_with_that_retry_count']
    
    # Save the DataFrame to CSV
    frequency_counts_df.to_csv(output_filename, index=False)
    logger.info(f"Results saved to '{output_filename}'.")
