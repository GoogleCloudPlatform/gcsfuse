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

import argparse

def parse_args() -> argparse.Namespace:
    """Parse the arguments and invoke the necessary steps."""
    parser = argparse.ArgumentParser(description="SSIOG arguments")
    parser.add_argument(
        "--prefix",
        type=str,
        nargs="+",
        help=(
            "Use the files starting with the given prefix(es)."
            + " Use gs://... when using direct GCS access."
        )
    )
    parser.add_argument(
            "--object-count-limit",
        type=int,
        help="Limit the number of objects.",
        default=1_000_000,
    )
    parser.add_argument(
        "--epochs",
        type=int,
        help="Number of epochs.",
        default=4,
    )
    parser.add_argument(
        "--steps",
        type=int,
        help="Number of steps.",
        default=2_000,
    )
    parser.add_argument(
        "--sample-size",
        type=int,
        help="Sample size in bytes.",
        default=1024,
    )
    parser.add_argument(
        "--batch-size",
        type=int,
        help="Batch size in number of samples.",
        default=1024,
    )
    parser.add_argument(
        "--read-order",
        type=str,
        nargs="+",
        help="Sampling order strategy (Sequential, FileRandom, FullRandom).",
        default=["Sequential"],
    )
    parser.add_argument(
        "--background-queue-maxsize",
        type=int,
        help="Maximum size for the threaded queue.",
        default=2048,
    )
    parser.add_argument(
        "--background-threads",
        type=int,
        help="Number of background threads.",
        default=16,
    )
    parser.add_argument(
        "--group-coordinator-address",
        type=str,
        help="The coordinator (rank==0) address.",
        default="localhost",
    )
    parser.add_argument(
        "--group-coordinator-port",
        type=str,
        help="The coordinator (rank==0) port.",
        default="4567",
    )
    parser.add_argument(
        "--group-member-id",
        type=int,
        help="The id within the group. Also known as the process rank.",
        default=0,
    )
    parser.add_argument(
        "--group-size",
        type=int,
        help="The process group size.",
        default=1,
    )
    parser.add_argument(
        "--label",
        type=str,
        help="Label to distinguish this run.",
        default="ssiog-benchmark",
    )
    parser.add_argument(
        "--log-metrics",
        type=bool,
        help="If enabled, sample latency is logged as csv.",
        default=False,
    )
    parser.add_argument(
        "--metrics-file",
        type=str,
        help="Log the metrics on the file.",
        default="metrics.csv",
    )
    parser.add_argument(
        "--export-metrics",
        type=bool,
        help="If enabled, then exports the otlp metrics.",
        default=False,
    )
    parser.add_argument(
        "--exporter-type",
        type=str,
        choices=["console", "cloud"],
        help="Exporter type.",
        default="cloud"
    )
    parser.add_argument(
        "--log-file",
        type=str,
        help="Log file path, if not given then writes to stdout.",
        default="",
    )
    parser.add_argument(
      "--log-level",
      type=str,
      default="INFO",
      choices=["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"],
      help="Set the logging level",
    )
    parser.add_argument(
        "--clear-pagecache-after-epoch",
        type=bool,
        help="Only clears page cache not dentries and inode cache",
        default=True,
    )

    return parser.parse_args()