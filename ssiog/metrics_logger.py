#!/usr/bin/env python3
# Copyright 2024 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import csv
import time
import queue
from queue import Queue
from threading import Thread


class AsyncMetricsLogger:
    """
    A class to asynchronously log metrics to a CSV file.
    """
    def __init__(self, file_name="metrics.csv", flush_interval=5):
        self.file_name = file_name
        self.flush_interval = flush_interval
        self.queue = Queue()
        self._shutdown = False
        self.writer_thread = Thread(target=self._writer_loop, daemon=True)
        self.writer_thread.start()

    def _writer_loop(self):
        with open(self.file_name, "w", newline="") as csvfile:
            writer = csv.writer(csvfile)
            writer.writerow(["timestamp", "sample_lat"])

            while True:
                try:
                    metrics = []
                    while True:
                        metrics.append(self.queue.get_nowait())
                except queue.Empty:
                    pass

                if metrics:
                    writer.writerows(metrics)
                    csvfile.flush()
                    
                # Get all remaining metrics from the queue if shutdown.
                if self._shutdown:
                    try:
                        metrics = []
                        while True:
                            metrics.append(self.queue.get_nowait())
                    except queue.Empty:
                        pass
                    
                    if metrics:
                        writer.writerows(metrics)
                        csvfile.flush()
                    break

                time.sleep(self.flush_interval)
                


    def log_metric(self, sample_lat):
        """
        Logs a metric data point asynchronously.
        """
        timestamp = time.time()
        self.queue.put([timestamp, sample_lat])
        
    def close(self):
        """
        Signals the writer thread to shut down and flushes any remaining metrics.
        """
        self._shutdown = True  # Set the shutdown flag
        self.writer_thread.join()  # Wait for the thread to finish


class NoOpMetricsLogger:
    """
    A no-op metrics logger that mimics the behavior of a real metrics logger 
    but doesn't actually record any metrics. Useful for testing or 
    disabling metrics without code changes.
    """
    def __init__(self, file_name="metrics.csv"):
        pass  # Ignore any arguments

    def log_metric(self, sample_lat):
        pass  # Do nothing when log_metric is called
    
    def close(self):
        pass  # Do nothing when close is called