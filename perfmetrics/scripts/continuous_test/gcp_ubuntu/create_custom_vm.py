import argparse
import sys
import subprocess

DEFAULT_VM_NAME = "perf-tests-vm"
DEFAULT_MACHINE_TYPE = "n2-standard-16"
DEFAULT_IMAGE_FAMILY = "ubuntu-2004-lts"
DEFAULT_IMAGE_PROJECT = "ubuntu-os-cloud"
DEFAULT_BOOT_DISK_SIZE = "50GiB"
DEFAULT_ZONE = "us-west1-a"
DEFAULT_STARTUP_SCRIPT = "gs://anushkadhn-onb-bucket/custom_vm_startup_script.sh"


def _parse_arguments(argv):
    """Parses the arguments provided to the script via command line.

    Args:
      argv: List of arguments received by the script.

    Returns:
      A class containing the parsed arguments.
    """
    argv = sys.argv
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
        '--boot_disk_size',
        help='Provide boot disk size of the vm instance',
        action='store',
        default=DEFAULT_BOOT_DISK_SIZE,
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
                      --boot-disk-size={args.boot_disk_size}\
                      --zone={args.zone}\
                      --metadata=startup-script-url={args.startup_script}",
                                shell=True)
    except subprocess.CalledProcessError as e:
        print(e.output)
