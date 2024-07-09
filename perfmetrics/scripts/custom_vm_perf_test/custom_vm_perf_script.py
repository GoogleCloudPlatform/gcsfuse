# Copyright 2024 Google Inc. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# To automate perf tests, this script creates a VM instance based on the
# flags passed.The command for running the script:
# python3 custom_vm_perf_script.py --vm_name=<vm_name> --machine_type=<machine-type>
# image_family=<image_family> --image_project=<image_project> --zone=<zone>
# --startup_script=<startup_script_filepath>

# For changing the bucket/spreadsheet id, change values in custom_vm_startup_script.sh

import argparse
import sys
import subprocess

DEFAULT_VM_NAME = "perf-tests-vm"
DEFAULT_MACHINE_TYPE = "n2-standard-96"
DEFAULT_IMAGE_FAMILY = "ubuntu-2004-lts"
DEFAULT_IMAGE_PROJECT = "ubuntu-os-cloud"
DEFAULT_ZONE = "us-west1-b"
DEFAULT_STARTUP_SCRIPT = "./../../custom_vm_startup_script.sh"
BOOT_DISK_SIZE = "100GiB"

def _parse_arguments(argv):
    """Parses the arguments provided to the script via command line.

    Args:
      argv: List of arguments received by the script.

    Returns:
      A class containing the parsed arguments.
    """

    if argv is None:
        argv = sys.argv[1:]

    parser = argparse.ArgumentParser()

    parser.add_argument(
        '--vm_name',
        help='Provide name of the vm instance',
        action='store',
        default=DEFAULT_VM_NAME,
        required=False,
    )

    parser.add_argument(
        '--machine_type',
        help='Provide machine type of the vm instance',
        action='store',
        default=DEFAULT_MACHINE_TYPE,
        required=False,
    )

    parser.add_argument(
        '--image_family',
        help='Provide image family of the vm instance',
        action='store',
        default=DEFAULT_IMAGE_FAMILY,
        required=False,
    )

    parser.add_argument(
        '--image_project',
        help='Provide image project of the vm instance',
        action='store',
        default=DEFAULT_IMAGE_PROJECT,
        required=False,
    )

    parser.add_argument(
        '--zone',
        help='Provide zone of the vm instance',
        action='store',
        default=DEFAULT_ZONE,
        required=False,
    )

    parser.add_argument(
        '--startup_script',
        help='Provide startup script for the vm instance',
        action='store',
        default=DEFAULT_STARTUP_SCRIPT,
        required=False,
    )

    return parser.parse_args(argv[1:])


if __name__ == '__main__':
    argv = sys.argv
    args = _parse_arguments(argv)
    # creating vm using gcloud command
    try:
        subprocess.check_output(f"gcloud compute instances create {args.vm_name}\
                      --machine-type={args.machine_type}\
                      --image-family={args.image_family}\
                      --image-project={args.image_project}\
                      --boot-disk-size={BOOT_DISK_SIZE}\
                      --zone={args.zone}\
                      --metadata-from-file=startup-script={args.startup_script}",
                                shell=True)
    except subprocess.CalledProcessError as e:
        print(e.output)
